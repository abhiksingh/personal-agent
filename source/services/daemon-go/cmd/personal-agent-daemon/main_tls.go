package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"personalagent/runtime/internal/transport"
)

func validateDaemonTLSConfig(config daemonRunConfig) error {
	mode := strings.ToLower(strings.TrimSpace(config.listenerMode))
	if mode == "" {
		mode = string(transport.ListenerModeTCP)
	}

	certFile := strings.TrimSpace(config.tlsCertFile)
	keyFile := strings.TrimSpace(config.tlsKeyFile)
	clientCAFile := strings.TrimSpace(config.tlsClientCAFile)
	requireClientCert := config.tlsRequireClientCert
	tlsConfigured := certFile != "" || keyFile != "" || clientCAFile != "" || requireClientCert

	if mode != string(transport.ListenerModeTCP) && tlsConfigured {
		return fmt.Errorf("tls options are only supported for tcp listener mode")
	}
	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("--tls-cert-file and --tls-key-file must be provided together")
	}
	if requireClientCert && (certFile == "" || keyFile == "") {
		return fmt.Errorf("--tls-require-client-cert requires --tls-cert-file and --tls-key-file")
	}
	if requireClientCert && clientCAFile == "" {
		return fmt.Errorf("--tls-require-client-cert requires --tls-client-ca-file")
	}
	if clientCAFile != "" && !requireClientCert {
		return fmt.Errorf("--tls-client-ca-file requires --tls-require-client-cert")
	}
	return nil
}

func hasDaemonTLSServerConfig(config daemonRunConfig) bool {
	return strings.TrimSpace(config.tlsCertFile) != "" && strings.TrimSpace(config.tlsKeyFile) != ""
}

func buildDaemonTLSServerConfig(config daemonRunConfig) (*tls.Config, error) {
	certFile := strings.TrimSpace(config.tlsCertFile)
	keyFile := strings.TrimSpace(config.tlsKeyFile)
	if certFile == "" || keyFile == "" {
		return nil, nil
	}

	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load tls certificate/key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
	}
	if config.tlsRequireClientCert {
		clientCAs, err := loadCertPool(config.tlsClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("load client CA bundle: %w", err)
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = clientCAs
	}
	return tlsConfig, nil
}

func loadCertPool(path string) (*x509.CertPool, error) {
	pemData, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return nil, fmt.Errorf("no certificates parsed from PEM")
	}
	return pool, nil
}
