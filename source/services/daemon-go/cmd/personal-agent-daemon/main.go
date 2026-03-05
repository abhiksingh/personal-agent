package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	options, err := parseDaemonCLIOptions(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(2)
	}

	if strings.TrimSpace(options.connectorWorker) != "" {
		if err := runConnectorWorker(options.connectorWorker, options.pluginID, options.workerHealthInterval, options.dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "connector worker failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if strings.TrimSpace(options.channelWorker) != "" {
		if err := runChannelWorker(options.channelWorker, options.pluginID, options.workerHealthInterval, options.dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "channel worker failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve daemon executable: %v\n", err)
		os.Exit(1)
	}
	config, warnings, err := buildDaemonRunConfig(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid daemon configuration: %v\n", err)
		os.Exit(2)
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
	}

	actionRequests := make(chan daemonLifecycleAction, 8)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	if err := runDaemonMainLoop(config, executable, actionRequests, signals, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "daemon runtime error: %v\n", err)
		os.Exit(1)
	}
}
