package daemonruntime

import (
	"errors"
	"strings"
)

type pluginWorkerRuntimeError struct {
	Operation string
	Source    string
	Detail    string
	Stderr    string
	Cause     error
}

func (e *pluginWorkerRuntimeError) Error() string {
	if e == nil {
		return ""
	}
	if detail := strings.TrimSpace(e.Detail); detail != "" {
		return detail
	}
	if e.Cause != nil {
		return strings.TrimSpace(e.Cause.Error())
	}
	return "plugin worker runtime error"
}

func (e *pluginWorkerRuntimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type pluginWorkerErrorContext struct {
	Message   string
	Source    string
	Operation string
	Stderr    string
}

func newPluginWorkerRuntimeError(operation string, source string, stderrTail []string, cause error) *pluginWorkerRuntimeError {
	stderr := collapsePluginWorkerStderrTail(stderrTail)
	detail := strings.TrimSpace("")
	if cause != nil {
		detail = strings.TrimSpace(cause.Error())
	}
	return &pluginWorkerRuntimeError{
		Operation: strings.TrimSpace(operation),
		Source:    strings.TrimSpace(source),
		Detail:    detail,
		Stderr:    stderr,
		Cause:     cause,
	}
}

func pluginWorkerErrorContextFrom(err error) pluginWorkerErrorContext {
	if err == nil {
		return pluginWorkerErrorContext{}
	}
	context := pluginWorkerErrorContext{
		Message: strings.TrimSpace(err.Error()),
	}
	var runtimeErr *pluginWorkerRuntimeError
	if errors.As(err, &runtimeErr) {
		context.Source = strings.TrimSpace(runtimeErr.Source)
		context.Operation = strings.TrimSpace(runtimeErr.Operation)
		context.Stderr = strings.TrimSpace(runtimeErr.Stderr)
		if context.Message == "" {
			context.Message = strings.TrimSpace(runtimeErr.Detail)
		}
	}
	return context
}

func pluginWorkerEventErrorContext(status PluginWorkerStatus, err error) pluginWorkerErrorContext {
	context := pluginWorkerErrorContextFrom(err)
	if context.Message == "" {
		context.Message = strings.TrimSpace(status.LastError)
	}
	if context.Source == "" {
		context.Source = strings.TrimSpace(status.LastErrorSource)
	}
	if context.Operation == "" {
		context.Operation = strings.TrimSpace(status.LastErrorOperation)
	}
	if context.Stderr == "" {
		context.Stderr = strings.TrimSpace(status.LastErrorStderr)
	}
	return context
}

func applyPluginWorkerErrorContext(status *PluginWorkerStatus, err error) {
	if status == nil {
		return
	}
	context := pluginWorkerErrorContextFrom(err)
	status.LastError = context.Message
	status.LastErrorSource = context.Source
	status.LastErrorOperation = context.Operation
	status.LastErrorStderr = context.Stderr
}

func clearPluginWorkerErrorContext(status *PluginWorkerStatus) {
	if status == nil {
		return
	}
	status.LastError = ""
	status.LastErrorSource = ""
	status.LastErrorOperation = ""
	status.LastErrorStderr = ""
}

func appendPluginWorkerStderrTail(existing []string, line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return existing
	}
	updated := append(existing, trimmed)
	if len(updated) > pluginWorkerErrorStderrTailMaxLines {
		updated = updated[len(updated)-pluginWorkerErrorStderrTailMaxLines:]
	}
	return updated
}

func collapsePluginWorkerStderrTail(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(joined) <= pluginWorkerErrorStderrTailMaxChars {
		return joined
	}
	trimmed := joined[len(joined)-pluginWorkerErrorStderrTailMaxChars:]
	return strings.TrimSpace(trimmed)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
