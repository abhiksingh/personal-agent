package daemonruntime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

func startTwilioCloudflaredTunnel(ctx context.Context, listenAddress string, startupTimeout time.Duration) (*twilioCloudflaredTunnelSession, string, error) {
	if parseWebhookTruthy(os.Getenv("PA_CLOUDFLARED_DRY_RUN")) {
		return &twilioCloudflaredTunnelSession{
			BinaryPath:    twilioCloudflaredBinaryName,
			PublicBaseURL: twilioCloudflaredDryRunPublicBaseURL,
			DryRun:        true,
		}, "cloudflared dry-run mode enabled; using synthetic public webhook base URL", nil
	}

	binaryPath, err := resolveTwilioCloudflaredBinaryPath()
	if err != nil {
		return nil, "", err
	}
	if err := runTwilioCloudflaredVersionSanityCheck(ctx, binaryPath); err != nil {
		return nil, "", fmt.Errorf("cloudflared sanity check failed: %w", err)
	}

	session, err := launchTwilioCloudflaredTunnel(ctx, binaryPath, listenAddress, startupTimeout)
	if err != nil {
		return nil, "", err
	}
	return session, "", nil
}

func runTwilioCloudflaredVersionSanityCheck(ctx context.Context, binaryPath string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	command := exec.CommandContext(checkCtx, binaryPath, "version")
	output, err := command.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}

func launchTwilioCloudflaredTunnel(ctx context.Context, binaryPath string, listenAddress string, startupTimeout time.Duration) (*twilioCloudflaredTunnelSession, error) {
	if startupTimeout <= 0 {
		startupTimeout = twilioWebhookCloudflaredStartupTimeout
	}

	localURL := "http://" + strings.TrimSpace(listenAddress)
	command := exec.CommandContext(ctx, binaryPath, "tunnel", "--no-autoupdate", "--url", localURL)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open cloudflared stdout pipe: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("open cloudflared stderr pipe: %w", err)
	}

	session := &twilioCloudflaredTunnelSession{
		BinaryPath: binaryPath,
		cmd:        command,
		waitCh:     make(chan error, 1),
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start cloudflared tunnel process: %w", err)
	}
	go func() {
		session.waitCh <- command.Wait()
	}()

	lineCh := make(chan string, 64)
	scanOutput := func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			select {
			case lineCh <- line:
			default:
			}
		}
	}
	go scanOutput(stdout)
	go scanOutput(stderr)

	startTimer := time.NewTimer(startupTimeout)
	defer startTimer.Stop()
	for {
		select {
		case line := <-lineCh:
			publicBaseURL := extractTwilioCloudflaredPublicBaseURL(line)
			if publicBaseURL == "" {
				continue
			}
			session.PublicBaseURL = publicBaseURL
			return session, nil
		case err := <-session.waitCh:
			if err == nil {
				return nil, fmt.Errorf("cloudflared tunnel exited before publishing public URL")
			}
			return nil, fmt.Errorf("cloudflared tunnel exited before publishing public URL: %w", err)
		case <-startTimer.C:
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = session.Close(closeCtx)
			closeCancel()
			return nil, fmt.Errorf("cloudflared tunnel startup timed out after %s", startupTimeout.Round(time.Millisecond))
		case <-ctx.Done():
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = session.Close(closeCtx)
			closeCancel()
			return nil, ctx.Err()
		}
	}
}

func extractTwilioCloudflaredPublicBaseURL(line string) string {
	for _, token := range strings.Fields(line) {
		candidate := strings.Trim(token, "\"'()[]{}<>,")
		if !strings.HasPrefix(strings.ToLower(candidate), "https://") {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil || strings.TrimSpace(parsed.Host) == "" {
			continue
		}
		return parsed.Scheme + "://" + parsed.Host
	}
	return ""
}

func (s *twilioCloudflaredTunnelSession) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}

	var closeErr error
	s.closeOnce.Do(func() {
		if s.DryRun {
			return
		}

		s.mu.Lock()
		command := s.cmd
		waitCh := s.waitCh
		s.mu.Unlock()

		if command == nil || waitCh == nil {
			return
		}
		select {
		case <-waitCh:
			return
		default:
		}
		if command.Process == nil {
			return
		}

		_ = command.Process.Signal(os.Interrupt)
		waitTimeout := 2 * time.Second
		if ctx != nil {
			if deadline, ok := ctx.Deadline(); ok {
				remaining := time.Until(deadline)
				if remaining > 0 && remaining < waitTimeout {
					waitTimeout = remaining
				}
			}
		}
		timer := time.NewTimer(waitTimeout)
		defer timer.Stop()
		var done <-chan struct{}
		if ctx != nil {
			done = ctx.Done()
		}

		select {
		case <-waitCh:
			return
		case <-timer.C:
		case <-done:
		}

		if err := command.Process.Kill(); err != nil {
			closeErr = err
			return
		}
		select {
		case <-waitCh:
		default:
		}
	})
	return closeErr
}
