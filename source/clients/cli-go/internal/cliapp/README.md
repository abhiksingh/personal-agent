# CLI App Package

`internal/cliapp` is the canonical CLI implementation package consumed by:

- `source/clients/cli-go/cmd/personal-agent` (client-owned module path)

This package remains intentionally centralized until command-family decomposition into `internal/{commands,client,auth,output}` is complete.
