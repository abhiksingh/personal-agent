package cliapp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"personalagent/runtime/internal/transport"
)

func validateCLIAuthTokenByRuntimeProfile(runtimeProfile string, authToken string, authTokenFile string) error {
	profile, err := normalizeCLIRuntimeProfile(runtimeProfile)
	if err != nil {
		return err
	}
	trimmedToken := strings.TrimSpace(authToken)
	if trimmedToken == "" {
		return fmt.Errorf("--auth-token is required")
	}
	if profile == cliRuntimeProfileProd {
		if strings.TrimSpace(authTokenFile) == "" {
			return fmt.Errorf("--auth-token-file is required for --runtime-profile=prod")
		}
		if len(trimmedToken) < 24 {
			return fmt.Errorf("--runtime-profile=prod requires auth token length >= 24")
		}
	}
	return nil
}

func validateCLITransportByRuntimeProfile(runtimeProfile string, listenerMode string, tlsInsecureSkipVerify bool) error {
	profile, err := normalizeCLIRuntimeProfile(runtimeProfile)
	if err != nil {
		return err
	}
	if profile != cliRuntimeProfileProd {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(listenerMode)) != string(transport.ListenerModeTCP) {
		return fmt.Errorf("--runtime-profile=prod requires --mode=tcp")
	}
	if tlsInsecureSkipVerify {
		return fmt.Errorf("--runtime-profile=prod does not allow --tls-insecure-skip-verify")
	}
	return nil
}

func resolveCLIRuntimeProfile(runtimeProfile string) (string, error) {
	return normalizeCLIRuntimeProfile(runtimeProfile)
}

func normalizeCLIRuntimeProfile(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return cliRuntimeProfileLocal, nil
	}
	switch normalized {
	case cliRuntimeProfileLocal:
		return cliRuntimeProfileLocal, nil
	case cliRuntimeProfileProd:
		return cliRuntimeProfileProd, nil
	default:
		return "", fmt.Errorf("unsupported --runtime-profile %q", raw)
	}
}

type cliTransportTLSOptions struct {
	ListenerMode        string
	TLSCAFile           string
	TLSClientCertFile   string
	TLSClientKeyFile    string
	TLSServerName       string
	TLSInsecureSkipCert bool
}

func buildCLITransportTLSConfig(options cliTransportTLSOptions) (*tls.Config, error) {
	mode := strings.ToLower(strings.TrimSpace(options.ListenerMode))
	if mode == "" {
		mode = string(transport.ListenerModeTCP)
	}

	caFile := strings.TrimSpace(options.TLSCAFile)
	clientCertFile := strings.TrimSpace(options.TLSClientCertFile)
	clientKeyFile := strings.TrimSpace(options.TLSClientKeyFile)
	serverName := strings.TrimSpace(options.TLSServerName)
	tlsRequested := caFile != "" || clientCertFile != "" || clientKeyFile != "" || serverName != "" || options.TLSInsecureSkipCert
	if !tlsRequested {
		return nil, nil
	}
	if mode != string(transport.ListenerModeTCP) {
		return nil, fmt.Errorf("tls options are only supported for tcp transport mode")
	}
	if (clientCertFile == "") != (clientKeyFile == "") {
		return nil, fmt.Errorf("--tls-client-cert-file and --tls-client-key-file must be provided together")
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: options.TLSInsecureSkipCert,
	}
	if serverName != "" {
		tlsConfig.ServerName = serverName
	}
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read --tls-ca-file: %w", err)
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("parse --tls-ca-file: no certificates found")
		}
		tlsConfig.RootCAs = roots
	}
	if clientCertFile != "" {
		certificate, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate/key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return tlsConfig, nil
}
