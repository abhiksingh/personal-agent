package daemonruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	twilioadapter "personalagent/runtime/internal/channels/adapters/twilio"
	"personalagent/runtime/internal/transport"
)

func (s *CommTwilioService) ServeTwilioWebhook(ctx context.Context, request transport.TwilioWebhookServeRequest) (transport.TwilioWebhookServeResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	listenAddress := strings.TrimSpace(request.ListenAddress)
	if listenAddress == "" {
		listenAddress = "127.0.0.1:8088"
	}

	signatureMode := strings.ToLower(strings.TrimSpace(request.SignatureMode))
	if signatureMode == "" {
		signatureMode = twilioWebhookSignatureModeStrict
	}
	if signatureMode != twilioWebhookSignatureModeStrict && signatureMode != twilioWebhookSignatureModeBypass {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("unsupported --signature-mode %q", request.SignatureMode)
	}

	voiceResponseMode := strings.ToLower(strings.TrimSpace(request.VoiceResponseMode))
	if voiceResponseMode == "" {
		voiceResponseMode = twilioWebhookVoiceResponseJSON
	}
	if voiceResponseMode != twilioWebhookVoiceResponseJSON && voiceResponseMode != twilioWebhookVoiceResponseTwiML {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("unsupported --voice-response-mode %q", request.VoiceResponseMode)
	}

	cloudflaredMode := strings.ToLower(strings.TrimSpace(request.CloudflaredMode))
	if cloudflaredMode == "" {
		cloudflaredMode = twilioWebhookCloudflaredModeAuto
	}
	if cloudflaredMode != twilioWebhookCloudflaredModeAuto &&
		cloudflaredMode != twilioWebhookCloudflaredModeOff &&
		cloudflaredMode != twilioWebhookCloudflaredModeRequired {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("unsupported --cloudflared-mode %q", request.CloudflaredMode)
	}
	cloudflaredStartupTimeout := time.Duration(request.CloudflaredStartupTimeoutMS) * time.Millisecond
	if cloudflaredStartupTimeout <= 0 {
		cloudflaredStartupTimeout = twilioWebhookCloudflaredStartupTimeout
	}

	assistantOptions := twilioWebhookAssistantOptions{
		Enabled:      request.AssistantReplies,
		TaskClass:    normalizeTaskClass(request.AssistantTaskClass),
		SystemPrompt: strings.TrimSpace(request.AssistantSystemPrompt),
		MaxHistory:   maxInt(1, request.AssistantMaxHistory),
		ReplyTimeout: time.Duration(request.AssistantReplyTimeoutMS) * time.Millisecond,
	}
	if assistantOptions.ReplyTimeout <= 0 {
		assistantOptions.ReplyTimeout = 12 * time.Second
	}

	config, err := s.twilioStore.Get(ctx, workspace)
	if err != nil {
		return transport.TwilioWebhookServeResponse{}, err
	}

	authToken := ""
	if signatureMode == twilioWebhookSignatureModeStrict {
		creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
		if err != nil {
			return transport.TwilioWebhookServeResponse{}, err
		}
		authToken = creds.AuthToken
	}

	resolvedSMSPath := normalizeDaemonWebhookPath(firstNonEmpty(strings.TrimSpace(request.SMSPath), defaultDaemonTwilioWebhookSMSPath()))
	resolvedVoicePath := normalizeDaemonWebhookPath(firstNonEmpty(strings.TrimSpace(request.VoicePath), defaultDaemonTwilioWebhookVoicePath()))
	if resolvedSMSPath == resolvedVoicePath {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("--sms-path and --voice-path must be different")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(resolvedSMSPath, func(writer http.ResponseWriter, httpRequest *http.Request) {
		response, statusCode := s.handleTwilioWebhookSMS(
			writer,
			httpRequest,
			workspace,
			config,
			signatureMode,
			authToken,
			assistantOptions,
		)
		writeWebhookJSON(writer, statusCode, response)
	})
	mux.HandleFunc(resolvedVoicePath, func(writer http.ResponseWriter, httpRequest *http.Request) {
		response, statusCode, twiml := s.handleTwilioWebhookVoice(
			writer,
			httpRequest,
			workspace,
			config,
			signatureMode,
			authToken,
			assistantOptions,
			voiceResponseMode,
			strings.TrimSpace(request.VoiceGreeting),
			strings.TrimSpace(request.VoiceFallback),
		)
		if voiceResponseMode == twilioWebhookVoiceResponseTwiML && statusCode < 400 {
			writeWebhookTwiML(writer, statusCode, twiml)
			return
		}
		writeWebhookJSON(writer, statusCode, response)
	})

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("listen %s: %w", listenAddress, err)
	}
	defer listener.Close()

	server := newTwilioWebhookHTTPServer(mux)
	defer server.Close()

	serveErrCh := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err == nil || err == http.ErrServerClosed {
			serveErrCh <- nil
			return
		}
		serveErrCh <- err
	}()

	servedAddress := listener.Addr().String()
	localSMSWebhookURL := fmt.Sprintf("http://%s%s", servedAddress, resolvedSMSPath)
	localVoiceWebhookURL := fmt.Sprintf("http://%s%s", servedAddress, resolvedVoicePath)
	response := transport.TwilioWebhookServeResponse{
		WorkspaceID:          workspace,
		SignatureMode:        signatureMode,
		ListenAddress:        servedAddress,
		LocalSMSWebhookURL:   localSMSWebhookURL,
		LocalVoiceWebhookURL: localVoiceWebhookURL,
		SMSWebhookURL:        localSMSWebhookURL,
		VoiceWebhookURL:      localVoiceWebhookURL,
		CloudflaredMode:      cloudflaredMode,
		AssistantReplies:     assistantOptions.Enabled,
		AssistantTaskClass:   assistantOptions.TaskClass,
		VoiceResponseMode:    voiceResponseMode,
	}
	warnings := make([]string, 0, 2)
	if signatureMode == twilioWebhookSignatureModeBypass {
		warnings = append(warnings, "signature validation bypass enabled for local development")
	}

	if cloudflaredMode != twilioWebhookCloudflaredModeOff {
		tunnelSession, tunnelWarning, err := startTwilioCloudflaredTunnel(ctx, servedAddress, cloudflaredStartupTimeout)
		if tunnelSession != nil {
			defer func() {
				closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer closeCancel()
				_ = tunnelSession.Close(closeCtx)
			}()
			response.CloudflaredAvailable = true
			response.CloudflaredActive = true
			response.CloudflaredBinaryPath = tunnelSession.BinaryPath
			response.CloudflaredDryRun = tunnelSession.DryRun
			response.PublicBaseURL = tunnelSession.PublicBaseURL
			response.SMSWebhookURL = strings.TrimRight(tunnelSession.PublicBaseURL, "/") + resolvedSMSPath
			response.VoiceWebhookURL = strings.TrimRight(tunnelSession.PublicBaseURL, "/") + resolvedVoicePath
		}
		if err != nil {
			if cloudflaredMode == twilioWebhookCloudflaredModeRequired {
				return transport.TwilioWebhookServeResponse{}, err
			}
			if !errors.Is(err, errTwilioCloudflaredNotInstalled) {
				response.CloudflaredAvailable = true
			}
			if errors.Is(err, errTwilioCloudflaredNotInstalled) {
				warnings = append(warnings, "cloudflared is not installed; webhook URLs remain local-only")
			} else {
				warnings = append(warnings, err.Error())
			}
		}
		if strings.TrimSpace(tunnelWarning) != "" {
			warnings = append(warnings, strings.TrimSpace(tunnelWarning))
		}
	}
	if len(warnings) > 0 {
		response.Warning = strings.Join(warnings, "; ")
	}

	runFor := time.Duration(request.RunForMS) * time.Millisecond
	if runFor < 0 {
		return transport.TwilioWebhookServeResponse{}, fmt.Errorf("--run-for must be >= 0")
	}

	if runFor > 0 {
		timer := time.NewTimer(runFor)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		case err := <-serveErrCh:
			if err != nil {
				return transport.TwilioWebhookServeResponse{}, fmt.Errorf("webhook server error: %w", err)
			}
		}
	} else {
		select {
		case <-ctx.Done():
		case err := <-serveErrCh:
			if err != nil {
				return transport.TwilioWebhookServeResponse{}, fmt.Errorf("webhook server error: %w", err)
			}
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	select {
	case err := <-serveErrCh:
		if err != nil {
			return transport.TwilioWebhookServeResponse{}, fmt.Errorf("webhook server shutdown error: %w", err)
		}
	default:
	}
	return response, nil
}

func (s *CommTwilioService) ReplayTwilioWebhook(ctx context.Context, request transport.TwilioWebhookReplayRequest) (transport.TwilioWebhookReplayResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	if len(request.Params) == 0 {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("fixture params are required")
	}

	kind := strings.ToLower(strings.TrimSpace(request.Kind))
	if kind != "sms" && kind != "voice" {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("fixture kind must be sms or voice (got %q)", request.Kind)
	}

	resolvedBaseURL := strings.TrimRight(strings.TrimSpace(request.BaseURL), "/")
	if resolvedBaseURL == "" {
		resolvedBaseURL = "http://127.0.0.1:8088"
	}
	targetPath := normalizeDaemonWebhookPath(firstNonEmpty(strings.TrimSpace(request.SMSPath), defaultDaemonTwilioWebhookSMSPath()))
	if kind == "voice" {
		targetPath = normalizeDaemonWebhookPath(firstNonEmpty(strings.TrimSpace(request.VoicePath), defaultDaemonTwilioWebhookVoicePath()))
	}
	targetURL := resolvedBaseURL + targetPath
	allowlistRaw := os.Getenv(twilioWebhookReplayTargetAllowlistEnv)
	if err := validateTwilioWebhookReplayTarget(ctx, targetURL, allowlistRaw); err != nil {
		return transport.TwilioWebhookReplayResponse{}, err
	}
	resolvedRequestURL := firstNonEmpty(strings.TrimSpace(request.RequestURL), targetURL)

	signatureMode := strings.ToLower(strings.TrimSpace(request.SignatureMode))
	if signatureMode == "" {
		signatureMode = twilioWebhookSignatureModeStrict
	}
	if signatureMode != twilioWebhookSignatureModeStrict && signatureMode != twilioWebhookSignatureModeBypass {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("unsupported --signature-mode %q", request.SignatureMode)
	}

	signature := ""
	if signatureMode == twilioWebhookSignatureModeStrict {
		config, err := s.twilioStore.Get(ctx, workspace)
		if err != nil {
			return transport.TwilioWebhookReplayResponse{}, err
		}
		creds, err := s.resolveTwilioWorkspaceCredentials(ctx, workspace, config)
		if err != nil {
			return transport.TwilioWebhookReplayResponse{}, err
		}
		resolvedSignature, err := twilioadapter.ComputeRequestSignature(creds.AuthToken, resolvedRequestURL, request.Params)
		if err != nil {
			return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("compute signature: %w", err)
		}
		signature = resolvedSignature
	}

	form := url.Values{}
	for key, value := range request.Params {
		form.Set(key, value)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("build replay request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpRequest.Header.Set(twilioWebhookRequestURLOverrideHeader, resolvedRequestURL)
	httpRequest.Header.Set(twilioWebhookReplayMarkerHeader, "1")
	if signature != "" {
		httpRequest.Header.Set("X-Twilio-Signature", signature)
	}

	timeout := time.Duration(request.HTTPTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	httpClient := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return validateTwilioWebhookReplayTarget(request.Context(), request.URL.String(), allowlistRaw)
		},
	}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("execute replay request: %w", err)
	}
	defer httpResponse.Body.Close()

	bodyBytes, truncated, err := readBoundedHTTPResponseBody(httpResponse.Body, twilioWebhookReplayMaxResponseBytes)
	if err != nil {
		return transport.TwilioWebhookReplayResponse{}, fmt.Errorf("read replay response: %w", err)
	}

	response := transport.TwilioWebhookReplayResponse{
		WorkspaceID:      workspace,
		Kind:             kind,
		TargetURL:        targetURL,
		RequestURL:       resolvedRequestURL,
		SignatureMode:    signatureMode,
		SignaturePresent: signature != "",
		StatusCode:       httpResponse.StatusCode,
		ResponseBody:     strings.TrimSpace(string(bodyBytes)),
	}
	if truncated {
		return response, fmt.Errorf(
			"twilio webhook replay response exceeded max size of %d bytes",
			twilioWebhookReplayMaxResponseBytes,
		)
	}
	return response, nil
}
