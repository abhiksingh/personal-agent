package finder

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	errSearchLimitReached = errors.New("finder search traversal limit reached")
)

type finderSearchCandidate struct {
	Path  string
	Score int
	IsDir bool
}

func semanticFindCandidates(query string, rootPath string) ([]finderSearchCandidate, string, error) {
	normalizedQuery := strings.TrimSpace(strings.ToLower(query))
	if normalizedQuery == "" {
		return nil, "", fmt.Errorf("finder query is required")
	}
	tokens := finderQueryTokenRegex.FindAllString(normalizedQuery, -1)
	if len(tokens) == 0 {
		return nil, "", fmt.Errorf("finder query must include alphanumeric tokens")
	}

	resolvedRoot, err := resolveSearchRoot(rootPath)
	if err != nil {
		return nil, "", err
	}

	candidates := make([]finderSearchCandidate, 0, 32)
	visitedEntries := 0
	walkErr := filepath.WalkDir(resolvedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		visitedEntries++
		if visitedEntries > defaultSearchMaxEntries {
			return errSearchLimitReached
		}

		if d.IsDir() && shouldSkipFinderDir(resolvedRoot, path, d.Name()) {
			return filepath.SkipDir
		}
		if depth := finderPathDepth(resolvedRoot, path); depth > defaultSearchMaxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		candidate, ok := scoreFinderCandidate(path, d.IsDir(), normalizedQuery, tokens)
		if !ok {
			return nil
		}
		candidates = append(candidates, candidate)
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errSearchLimitReached) {
		return nil, resolvedRoot, fmt.Errorf("walk finder search root %s: %w", resolvedRoot, walkErr)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Path < candidates[j].Path
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > maxFinderMatches {
		candidates = candidates[:maxFinderMatches]
	}
	return candidates, resolvedRoot, nil
}

func resolveSearchRoot(explicitRoot string) (string, error) {
	root := strings.TrimSpace(explicitRoot)
	if root == "" {
		root = strings.TrimSpace(os.Getenv("PA_FINDER_SEARCH_ROOT"))
	}
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", fmt.Errorf("finder search root unavailable")
		}
		root = home
	}
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("finder search root must be absolute: %s", root)
	}
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("finder search root inaccessible: %w", err)
	}
	return root, nil
}

func finderPathDepth(root string, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	parts := strings.Split(rel, string(filepath.Separator))
	depth := 0
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			depth++
		}
	}
	return depth
}

func shouldSkipFinderDir(root string, path string, name string) bool {
	if path == root {
		return false
	}
	lowerName := strings.ToLower(strings.TrimSpace(name))
	if lowerName == "" {
		return false
	}
	if strings.HasPrefix(lowerName, ".") {
		return true
	}
	switch lowerName {
	case "node_modules", "library", "system", "private", "volumes":
		return true
	default:
		return false
	}
}

func scoreFinderCandidate(path string, isDir bool, normalizedQuery string, tokens []string) (finderSearchCandidate, bool) {
	cleanPath := filepath.Clean(path)
	lowerPath := strings.ToLower(cleanPath)
	lowerName := strings.ToLower(filepath.Base(cleanPath))

	if !finderCandidateMatches(lowerName, lowerPath, normalizedQuery, tokens) {
		return finderSearchCandidate{}, false
	}

	score := 0
	if lowerName == normalizedQuery {
		score += 120
	}
	if strings.Contains(lowerName, normalizedQuery) {
		score += 70
	}
	if strings.Contains(lowerPath, normalizedQuery) {
		score += 25
	}
	for _, token := range tokens {
		if strings.Contains(lowerName, token) {
			score += 12
		}
		if strings.Contains(lowerPath, token) {
			score += 4
		}
	}
	if !isDir {
		score += 6
	}
	if score <= 0 {
		return finderSearchCandidate{}, false
	}
	return finderSearchCandidate{
		Path:  cleanPath,
		Score: score,
		IsDir: isDir,
	}, true
}

func finderCandidateMatches(lowerName string, lowerPath string, normalizedQuery string, tokens []string) bool {
	if strings.Contains(lowerName, normalizedQuery) {
		return true
	}
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !strings.Contains(lowerPath, token) {
			return false
		}
	}
	return true
}
