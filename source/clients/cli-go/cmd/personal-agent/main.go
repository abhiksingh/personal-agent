package main

import (
	"os"

	"personalagent/runtime/cli/internal/cliapp"
)

func main() {
	os.Exit(cliapp.Run(os.Args[1:], os.Stdout, os.Stderr))
}
