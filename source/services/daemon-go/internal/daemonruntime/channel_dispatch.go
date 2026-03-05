package daemonruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"personalagent/runtime/internal/channelcheck"
	messagesadapter "personalagent/runtime/internal/channels/adapters/messages"
	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	shared "personalagent/runtime/internal/shared/contracts"
)

const (
	twilioChannelWorkerCapabilityCheck      = "channel.twilio.check"
	twilioChannelWorkerCapabilitySendSMS    = "channel.twilio.sms.send"
	twilioChannelWorkerCapabilityStartVoice = "channel.twilio.voice.start_call"
	messagesWorkerCapabilitySend            = "channel.messages.send"
	messagesWorkerCapabilityPollInbound     = "channel.messages.ingest_poll"
	channelRuntimeExecAddressKey            = "exec_address"
)

type ChannelWorkerDispatcher interface {
	CheckTwilio(ctx context.Context, request channelcheck.TwilioRequest) (channelcheck.TwilioResult, error)
	SendTwilioSMS(ctx context.Context, request twilioadapter.SMSAPIRequest) (twilioadapter.SMSAPIResponse, error)
	StartTwilioVoiceCall(ctx context.Context, request twilioadapter.VoiceCallRequest) (twilioadapter.VoiceCallResponse, error)
	SendMessages(ctx context.Context, request messagesadapter.SendRequest) (messagesadapter.SendResponse, error)
	PollMessagesInbound(ctx context.Context, request messagesadapter.InboundPollRequest) (messagesadapter.InboundPollResponse, error)
}

type TwilioWebhookSMSIngressRequest struct {
	WorkspaceID         string            `json:"workspace_id"`
	SignatureMode       string            `json:"signature_mode"`
	AuthToken           string            `json:"auth_token,omitempty"`
	RequestURL          string            `json:"request_url,omitempty"`
	SignatureValue      string            `json:"signature_value,omitempty"`
	MessageSID          string            `json:"message_sid"`
	ProviderAccount     string            `json:"provider_account,omitempty"`
	FromAddress         string            `json:"from_address"`
	ToAddress           string            `json:"to_address"`
	BodyText            string            `json:"body_text,omitempty"`
	ConfiguredSMSNumber string            `json:"configured_sms_number,omitempty"`
	ProviderPayload     map[string]string `json:"provider_payload,omitempty"`
}

type TwilioWebhookSMSIngressResult struct {
	WorkspaceID string `json:"workspace_id"`
	Accepted    bool   `json:"accepted"`
	Replayed    bool   `json:"replayed"`
	EventID     string `json:"event_id,omitempty"`
	ThreadID    string `json:"thread_id,omitempty"`
	MessageSID  string `json:"message_sid,omitempty"`
	Error       string `json:"error,omitempty"`
	StatusCode  int    `json:"status_code"`
}

type TwilioWebhookVoiceIngressRequest struct {
	WorkspaceID                string            `json:"workspace_id"`
	SignatureMode              string            `json:"signature_mode"`
	AuthToken                  string            `json:"auth_token,omitempty"`
	RequestURL                 string            `json:"request_url,omitempty"`
	SignatureValue             string            `json:"signature_value,omitempty"`
	ProviderEventID            string            `json:"provider_event_id,omitempty"`
	CallSID                    string            `json:"call_sid"`
	ProviderAccount            string            `json:"provider_account,omitempty"`
	FromAddress                string            `json:"from_address"`
	ToAddress                  string            `json:"to_address"`
	Direction                  string            `json:"direction,omitempty"`
	CallStatus                 string            `json:"call_status,omitempty"`
	TranscriptText             string            `json:"transcript_text,omitempty"`
	TranscriptDirection        string            `json:"transcript_direction,omitempty"`
	TranscriptAssistantEmitted bool              `json:"transcript_assistant_emitted,omitempty"`
	ConfiguredVoiceNumber      string            `json:"configured_voice_number,omitempty"`
	ProviderPayload            map[string]string `json:"provider_payload,omitempty"`
}

type TwilioWebhookVoiceIngressResult struct {
	WorkspaceID       string `json:"workspace_id"`
	Accepted          bool   `json:"accepted"`
	Replayed          bool   `json:"replayed"`
	ProviderEventID   string `json:"provider_event_id"`
	CallSID           string `json:"call_sid"`
	CallSessionID     string `json:"call_session_id,omitempty"`
	ThreadID          string `json:"thread_id,omitempty"`
	CallStatus        string `json:"call_status,omitempty"`
	StatusEventID     string `json:"status_event_id,omitempty"`
	TranscriptEventID string `json:"transcript_event_id,omitempty"`
	Error             string `json:"error,omitempty"`
	StatusCode        int    `json:"status_code"`
}

type SupervisorChannelWorkerDispatcher struct {
	supervisor      PluginSupervisor
	httpClient      *http.Client
	restartBackoff  time.Duration
	restartDeadline time.Duration
	resilience      *workerDispatchResilience
}

type channelWorkerExecuteError struct {
	Retryable bool
	Message   string
	Cause     error
}

func (e *channelWorkerExecuteError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return strings.TrimSpace(e.Message)
	}
	if e.Cause != nil {
		return strings.TrimSpace(e.Cause.Error())
	}
	return "channel worker execute failed"
}

func (e *channelWorkerExecuteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewSupervisorChannelWorkerDispatcher(supervisor PluginSupervisor) *SupervisorChannelWorkerDispatcher {
	return &SupervisorChannelWorkerDispatcher{
		supervisor:      supervisor,
		httpClient:      &http.Client{Timeout: 4 * time.Second},
		restartBackoff:  50 * time.Millisecond,
		restartDeadline: 4 * time.Second,
		resilience:      defaultWorkerDispatchResilience(),
	}
}

func (d *SupervisorChannelWorkerDispatcher) CheckTwilio(ctx context.Context, request channelcheck.TwilioRequest) (channelcheck.TwilioResult, error) {
	var result channelcheck.TwilioResult
	usedWorker, err := d.executeWithWorker(ctx, twilioChannelWorkerCapabilityCheck, "twilio_check", request, &result, true)
	if usedWorker {
		return result, err
	}
	return channelcheck.TwilioResult{}, fmt.Errorf("twilio check worker is not available")
}

func (d *SupervisorChannelWorkerDispatcher) SendTwilioSMS(ctx context.Context, request twilioadapter.SMSAPIRequest) (twilioadapter.SMSAPIResponse, error) {
	var result twilioadapter.SMSAPIResponse
	usedWorker, err := d.executeWithWorker(ctx, twilioChannelWorkerCapabilitySendSMS, "twilio_sms_send", request, &result, true)
	if usedWorker {
		return result, err
	}
	return twilioadapter.SMSAPIResponse{}, fmt.Errorf("twilio sms worker is not available")
}

func (d *SupervisorChannelWorkerDispatcher) StartTwilioVoiceCall(ctx context.Context, request twilioadapter.VoiceCallRequest) (twilioadapter.VoiceCallResponse, error) {
	var result twilioadapter.VoiceCallResponse
	usedWorker, err := d.executeWithWorker(ctx, twilioChannelWorkerCapabilityStartVoice, "twilio_voice_start_call", request, &result, true)
	if usedWorker {
		return result, err
	}
	return twilioadapter.VoiceCallResponse{}, fmt.Errorf("twilio voice worker is not available")
}

func (d *SupervisorChannelWorkerDispatcher) SendMessages(ctx context.Context, request messagesadapter.SendRequest) (messagesadapter.SendResponse, error) {
	var result messagesadapter.SendResponse
	usedWorker, err := d.executeWithWorker(ctx, messagesWorkerCapabilitySend, "messages_send", request, &result, true)
	if usedWorker {
		return result, err
	}
	return messagesadapter.SendResponse{}, fmt.Errorf("messages worker is not available")
}

func (d *SupervisorChannelWorkerDispatcher) PollMessagesInbound(ctx context.Context, request messagesadapter.InboundPollRequest) (messagesadapter.InboundPollResponse, error) {
	var result messagesadapter.InboundPollResponse
	// Polling happens on a fixed cadence via daemon runtime loops; avoid manual restart storms on repeated source-read failures.
	usedWorker, err := d.executeWithWorker(ctx, messagesWorkerCapabilityPollInbound, "messages_poll_inbound", request, &result, false)
	if usedWorker {
		return result, err
	}
	return messagesadapter.InboundPollResponse{}, fmt.Errorf("messages inbound poll worker is not available")
}

func (d *SupervisorChannelWorkerDispatcher) executeWithWorker(
	ctx context.Context,
	capability string,
	operation string,
	payload any,
	output any,
	allowManualRestart bool,
) (bool, error) {
	status, found, err := d.resolveWorker(capability)
	if err != nil {
		return true, err
	}
	if !found {
		return false, nil
	}

	if d.resilience == nil {
		d.resilience = defaultWorkerDispatchResilience()
	}

	latestStatus := status
	_, err = executeWorkerDispatchWithResilience(ctx, workerDispatchAttemptSpec[struct{}]{
		resilience: d.resilience,
		workerID:   status.PluginID,
		execute: func(attemptCtx context.Context) (struct{}, error) {
			return struct{}{}, d.executeWorkerOperation(attemptCtx, latestStatus, operation, payload, output)
		},
		isRetryable: channelWorkerErrorRetryable,
		recoverWorker: func(retryCtx context.Context) {
			if recoveredStatus, recovered := d.tryRecoverChannelWorker(retryCtx, latestStatus, capability, allowManualRestart); recovered {
				latestStatus = recoveredStatus
			}
		},
		exhaustedErrMsg: "channel worker dispatch failed without explicit error",
	})
	if err != nil {
		return true, err
	}
	return true, nil
}

func (d *SupervisorChannelWorkerDispatcher) tryRecoverChannelWorker(
	ctx context.Context,
	currentStatus PluginWorkerStatus,
	capability string,
	allowManualRestart bool,
) (PluginWorkerStatus, bool) {
	latestStatus, waitErr := d.waitForRunningChannelWorker(ctx, currentStatus.PluginID, capability)
	if waitErr == nil {
		return latestStatus, true
	}
	if allowManualRestart && d.supervisor != nil {
		if restartErr := d.supervisor.RestartWorker(ctx, currentStatus.PluginID); restartErr == nil {
			restartedStatus, restartWaitErr := d.waitForRunningChannelWorkerAfter(ctx, currentStatus.PluginID, capability, currentStatus)
			if restartWaitErr == nil {
				return restartedStatus, true
			}
		}
	}
	return currentStatus, false
}

func (d *SupervisorChannelWorkerDispatcher) resolveWorker(capability string) (PluginWorkerStatus, bool, error) {
	if d.supervisor == nil {
		return PluginWorkerStatus{}, false, nil
	}
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return PluginWorkerStatus{}, false, fmt.Errorf("channel worker capability is required")
	}

	statuses := d.supervisor.ListWorkers()
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].PluginID < statuses[j].PluginID
	})
	for _, status := range statuses {
		if status.State != PluginWorkerStateRunning {
			continue
		}
		if channelWorkerExecAddress(status.Metadata) == "" {
			continue
		}
		if !channelSupportsCapability(status.Metadata, capability) {
			continue
		}
		return status, true, nil
	}
	return PluginWorkerStatus{}, false, nil
}

func (d *SupervisorChannelWorkerDispatcher) executeWorkerOperation(ctx context.Context, status PluginWorkerStatus, operation string, payload any, output any) error {
	address := channelWorkerExecAddress(status.Metadata)
	if address == "" {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("channel worker %s has no execution endpoint", status.PluginID),
		}
	}
	authToken := strings.TrimSpace(status.execAuthToken)
	if authToken == "" {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("channel worker %s has no daemon-issued auth token", status.PluginID),
		}
	}

	requestBody, err := json.Marshal(channelWorkerExecuteRequest{
		Operation: strings.TrimSpace(operation),
		Payload:   payload,
	})
	if err != nil {
		return &channelWorkerExecuteError{
			Retryable: false,
			Message:   fmt.Sprintf("marshal channel worker request: %v", err),
			Cause:     err,
		}
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/execute", bytes.NewReader(requestBody))
	if err != nil {
		return &channelWorkerExecuteError{
			Retryable: false,
			Message:   fmt.Sprintf("build channel worker request: %v", err),
			Cause:     err,
		}
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+authToken)

	httpResponse, err := d.httpClient.Do(httpRequest)
	if err != nil {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   strings.TrimSpace(err.Error()),
			Cause:     err,
		}
	}
	defer httpResponse.Body.Close()

	rawBody, truncated, err := readBoundedHTTPResponseBody(httpResponse.Body, daemonWorkerRPCResponseBodyLimitBytes)
	if err != nil {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("read channel worker response: %v", err),
			Cause:     err,
		}
	}
	if truncated {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message: fmt.Sprintf(
				"read channel worker response: response exceeded max size of %d bytes",
				daemonWorkerRPCResponseBodyLimitBytes,
			),
		}
	}
	if httpResponse.StatusCode >= 400 {
		message := strings.TrimSpace(extractWorkerErrorMessage(rawBody))
		if message == "" {
			message = strings.TrimSpace(string(rawBody))
		}
		if message == "" {
			message = httpResponse.Status
		}
		return &channelWorkerExecuteError{
			Retryable: httpResponse.StatusCode >= 500,
			Message:   fmt.Sprintf("channel worker execute failed: %s", message),
		}
	}

	response := channelWorkerExecuteResponse{}
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   fmt.Sprintf("decode channel worker response: %v", err),
			Cause:     err,
		}
	}
	if strings.TrimSpace(response.Error) != "" {
		return &channelWorkerExecuteError{
			Retryable: false,
			Message:   strings.TrimSpace(response.Error),
		}
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return &channelWorkerExecuteError{
			Retryable: true,
			Message:   "channel worker response missing result",
		}
	}

	if err := json.Unmarshal(response.Result, output); err != nil {
		return &channelWorkerExecuteError{
			Retryable: false,
			Message:   fmt.Sprintf("decode channel worker result: %v", err),
			Cause:     err,
		}
	}
	return nil
}

func (d *SupervisorChannelWorkerDispatcher) waitForRunningChannelWorker(ctx context.Context, pluginID string, capability string) (PluginWorkerStatus, error) {
	if d.supervisor == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin supervisor is required")
	}
	return waitForPluginWorkerStatus(
		ctx,
		func() (PluginWorkerStatus, bool) {
			return d.supervisor.WorkerStatus(pluginID)
		},
		func(status PluginWorkerStatus) bool {
			return status.State == PluginWorkerStateRunning &&
				channelWorkerExecAddress(status.Metadata) != "" &&
				channelSupportsCapability(status.Metadata, capability)
		},
		d.restartBackoff,
		d.restartDeadline,
		fmt.Errorf("timed out waiting for channel worker %s restart", pluginID),
	)
}

func (d *SupervisorChannelWorkerDispatcher) waitForRunningChannelWorkerAfter(ctx context.Context, pluginID string, capability string, previous PluginWorkerStatus) (PluginWorkerStatus, error) {
	if d.supervisor == nil {
		return PluginWorkerStatus{}, fmt.Errorf("plugin supervisor is required")
	}
	return waitForPluginWorkerStatus(
		ctx,
		func() (PluginWorkerStatus, bool) {
			return d.supervisor.WorkerStatus(pluginID)
		},
		func(status PluginWorkerStatus) bool {
			if status.State != PluginWorkerStateRunning ||
				channelWorkerExecAddress(status.Metadata) == "" ||
				!channelSupportsCapability(status.Metadata, capability) {
				return false
			}
			addressChanged := channelWorkerExecAddress(status.Metadata) != channelWorkerExecAddress(previous.Metadata)
			processChanged := status.ProcessID != previous.ProcessID || status.RestartCount != previous.RestartCount
			return addressChanged || processChanged
		},
		d.restartBackoff,
		d.restartDeadline,
		fmt.Errorf("timed out waiting for channel worker %s restart", pluginID),
	)
}

func channelWorkerExecAddress(metadata shared.AdapterMetadata) string {
	if len(metadata.Runtime) == 0 {
		return ""
	}
	return strings.TrimSpace(metadata.Runtime[channelRuntimeExecAddressKey])
}

func channelSupportsCapability(metadata shared.AdapterMetadata, capability string) bool {
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return false
	}
	for _, descriptor := range metadata.Capabilities {
		if strings.TrimSpace(descriptor.Key) == capability {
			return true
		}
	}
	return false
}

func channelWorkerErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	var executeErr *channelWorkerExecuteError
	if errors.As(err, &executeErr) {
		return executeErr.Retryable
	}
	return false
}

type channelWorkerExecuteRequest struct {
	Operation string `json:"operation"`
	Payload   any    `json:"payload"`
}

type channelWorkerExecuteResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error,omitempty"`
}
