package transport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type registeredControlRoute struct {
	path    string
	handler string
}

func TestMutatingRoutesRequireExplicitScopeAndRateLimitPolicyDecision(t *testing.T) {
	routes := collectRegisteredControlRoutesFromSource(t)
	handlerMethods := collectHandlerAllowedMethodsFromSource(t)
	mutatingMethods := map[string]struct{}{
		http.MethodPost:   {},
		http.MethodPut:    {},
		http.MethodPatch:  {},
		http.MethodDelete: {},
	}

	for _, route := range routes {
		methods := handlerMethods[route.handler]
		if len(methods) == 0 {
			t.Fatalf("registered route %s (%s) is missing explicit method guard extraction", route.path, route.handler)
		}
		for _, method := range methods {
			if _, mutating := mutatingMethods[method]; !mutating {
				continue
			}
			request := httptest.NewRequest(method, "http://localhost"+route.path, nil)
			requiredScopes := requiredScopesForRoute(request)
			if len(requiredScopes) == 0 {
				t.Fatalf("mutating route %s %s is missing explicit auth-scope policy", method, route.path)
			}
			if !hasWriteScope(requiredScopes) {
				continue
			}
			decision := controlRouteRateLimitPolicyDecisionForRoute(method, route.path)
			if !decision.enforced && !decision.allowlisted {
				t.Fatalf("mutating write route %s %s is missing rate-limit policy decision", method, route.path)
			}
			if decision.enforced && strings.TrimSpace(decision.endpointKey) == "" {
				t.Fatalf("enforced rate-limit policy for %s %s must declare endpoint key", method, route.path)
			}
			if decision.allowlisted && strings.TrimSpace(decision.reason) == "" {
				t.Fatalf("allowlisted rate-limit policy for %s %s must declare rationale", method, route.path)
			}
		}
	}
}

func TestRateLimitPolicyTablesReferenceRegisteredMutatingRoutes(t *testing.T) {
	routes := collectRegisteredControlRoutesFromSource(t)
	handlerMethods := collectHandlerAllowedMethodsFromSource(t)
	routesByPath := map[string]registeredControlRoute{}
	for _, route := range routes {
		normalizedPath := strings.TrimSpace(route.path)
		if _, exists := routesByPath[normalizedPath]; exists {
			t.Fatalf("duplicate registered route path %s", normalizedPath)
		}
		routesByPath[normalizedPath] = route
	}

	for _, rule := range controlRouteRateLimitPolicyRules {
		route, ok := routesByPath[strings.TrimSpace(rule.path)]
		if !ok {
			t.Fatalf("rate-limit policy route %s %s is not registered", rule.method, rule.path)
		}
		if !containsMethod(handlerMethods[route.handler], rule.method) {
			t.Fatalf("rate-limit policy route %s %s is not supported by handler %s methods %v", rule.method, rule.path, route.handler, handlerMethods[route.handler])
		}
		request := httptestRequest(rule.method, "http://localhost"+strings.TrimSpace(rule.path))
		requiredScopes := requiredScopesForRoute(request)
		if !hasWriteScope(requiredScopes) {
			t.Fatalf("rate-limit policy route %s %s must map to mutating write scope, got %v", rule.method, rule.path, requiredScopes)
		}
	}

	for _, rule := range controlRouteRateLimitPolicyAllowlist {
		route, ok := routesByPath[strings.TrimSpace(rule.path)]
		if !ok {
			t.Fatalf("rate-limit allowlist route %s %s is not registered", rule.method, rule.path)
		}
		if !containsMethod(handlerMethods[route.handler], rule.method) {
			t.Fatalf("rate-limit allowlist route %s %s is not supported by handler %s methods %v", rule.method, rule.path, route.handler, handlerMethods[route.handler])
		}
		request := httptestRequest(rule.method, "http://localhost"+strings.TrimSpace(rule.path))
		requiredScopes := requiredScopesForRoute(request)
		if !hasWriteScope(requiredScopes) {
			t.Fatalf("rate-limit allowlist route %s %s must map to mutating write scope, got %v", rule.method, rule.path, requiredScopes)
		}
		if strings.TrimSpace(rule.reason) == "" {
			t.Fatalf("rate-limit allowlist route %s %s must include rationale", rule.method, rule.path)
		}
	}
}

func collectRegisteredControlRoutesFromSource(t *testing.T) []registeredControlRoute {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read transport directory: %v", err)
	}
	routePattern := regexp.MustCompile(`mux\.HandleFunc\("([^"]+)",\s*s\.(\w+)\)`)
	routeSet := map[string]registeredControlRoute{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "server_routes_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read route file %s: %v", name, err)
		}
		matches := routePattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			path := strings.TrimSpace(match[1])
			handler := strings.TrimSpace(match[2])
			if path == "" || handler == "" {
				continue
			}
			routeSet[path] = registeredControlRoute{
				path:    path,
				handler: handler,
			}
		}
	}
	if len(routeSet) == 0 {
		t.Fatalf("no registered control routes discovered from source")
	}
	routes := make([]registeredControlRoute, 0, len(routeSet))
	for _, route := range routeSet {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].path == routes[j].path {
			return routes[i].handler < routes[j].handler
		}
		return routes[i].path < routes[j].path
	})
	return routes
}

func collectHandlerAllowedMethodsFromSource(t *testing.T) map[string][]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read transport directory: %v", err)
	}
	directMethods := map[string][]string{}
	calledHandlersByHandler := map[string][]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "server_") || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, "server_routes_") {
			continue
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse handler file %s: %v", name, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil {
				continue
			}
			handlerName := strings.TrimSpace(function.Name.Name)
			if !strings.HasPrefix(handlerName, "handle") {
				continue
			}
			methods, calledHandlers := extractHandlerSignals(function)
			if len(methods) > 0 {
				directMethods[handlerName] = methods
			}
			if len(calledHandlers) > 0 {
				calledHandlersByHandler[handlerName] = calledHandlers
			}
		}
	}
	resolved := map[string][]string{}
	var resolve func(string, map[string]struct{}) []string
	resolve = func(handler string, stack map[string]struct{}) []string {
		if methods, ok := resolved[handler]; ok {
			return methods
		}
		if _, inStack := stack[handler]; inStack {
			return nil
		}
		stack[handler] = struct{}{}
		methodSet := map[string]struct{}{}
		for _, method := range directMethods[handler] {
			methodSet[method] = struct{}{}
		}
		for _, called := range calledHandlersByHandler[handler] {
			for _, method := range resolve(called, stack) {
				methodSet[method] = struct{}{}
			}
		}
		delete(stack, handler)
		methods := make([]string, 0, len(methodSet))
		for method := range methodSet {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		resolved[handler] = methods
		return methods
	}
	for handler := range directMethods {
		resolve(handler, map[string]struct{}{})
	}
	for handler := range calledHandlersByHandler {
		resolve(handler, map[string]struct{}{})
	}
	return resolved
}

func extractHandlerSignals(function *ast.FuncDecl) ([]string, []string) {
	methodSet := map[string]struct{}{}
	calledHandlerSet := map[string]struct{}{}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.IfStmt:
			if method, ok := methodFromMethodComparison(typed.Cond); ok {
				methodSet[method] = struct{}{}
			}
		case *ast.CallExpr:
			if method, ok := methodFromRequireAuthorizedMethodCall(typed); ok {
				methodSet[method] = struct{}{}
			}
			if calledHandler, ok := calledHandlerNameFromCall(typed); ok {
				calledHandlerSet[calledHandler] = struct{}{}
			}
		case *ast.SwitchStmt:
			if !isRequestMethodExpr(typed.Tag) {
				return true
			}
			for _, statement := range typed.Body.List {
				clause, ok := statement.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, expr := range clause.List {
					if method, ok := httpMethodNameFromExpr(expr); ok {
						methodSet[method] = struct{}{}
					}
				}
			}
		}
		return true
	})
	methods := make([]string, 0, len(methodSet))
	for method := range methodSet {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	calledHandlers := make([]string, 0, len(calledHandlerSet))
	for calledHandler := range calledHandlerSet {
		calledHandlers = append(calledHandlers, calledHandler)
	}
	sort.Strings(calledHandlers)
	return methods, calledHandlers
}

func methodFromMethodComparison(expr ast.Expr) (string, bool) {
	condition, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return "", false
	}
	if condition.Op != token.NEQ && condition.Op != token.EQL {
		return "", false
	}
	if isRequestMethodExpr(condition.X) {
		return httpMethodNameFromExpr(condition.Y)
	}
	if isRequestMethodExpr(condition.Y) {
		return httpMethodNameFromExpr(condition.X)
	}
	return "", false
}

func methodFromRequireAuthorizedMethodCall(call *ast.CallExpr) (string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(selector.Sel.Name) != "requireAuthorizedMethod" {
		return "", false
	}
	if len(call.Args) < 3 {
		return "", false
	}
	return httpMethodNameFromExpr(call.Args[2])
}

func calledHandlerNameFromCall(call *ast.CallExpr) (string, bool) {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || strings.TrimSpace(receiver.Name) != "s" {
		return "", false
	}
	name := strings.TrimSpace(selector.Sel.Name)
	if !strings.HasPrefix(name, "handle") {
		return "", false
	}
	return name, true
}

func isRequestMethodExpr(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || strings.TrimSpace(selector.Sel.Name) != "Method" {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	return strings.TrimSpace(identifier.Name) == "request"
}

func httpMethodNameFromExpr(expr ast.Expr) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if ok {
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || strings.TrimSpace(identifier.Name) != "http" {
			return "", false
		}
		switch strings.TrimSpace(selector.Sel.Name) {
		case "MethodGet":
			return http.MethodGet, true
		case "MethodPost":
			return http.MethodPost, true
		case "MethodPut":
			return http.MethodPut, true
		case "MethodPatch":
			return http.MethodPatch, true
		case "MethodDelete":
			return http.MethodDelete, true
		case "MethodHead":
			return http.MethodHead, true
		case "MethodOptions":
			return http.MethodOptions, true
		}
		return "", false
	}
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	decoded, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	normalized := strings.ToUpper(strings.TrimSpace(decoded))
	if normalized == "" {
		return "", false
	}
	return normalized, true
}

func httptestRequest(method string, url string) *http.Request {
	return httptest.NewRequest(method, url, nil)
}

func hasWriteScope(scopes []string) bool {
	for _, scope := range scopes {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(scope)), ":write") {
			return true
		}
	}
	return false
}

func containsMethod(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}
