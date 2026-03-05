package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"personalagent/runtime/internal/persistence/migrator"

	_ "modernc.org/sqlite"
)

func main() {
	var dbPath string
	flag.StringVar(&dbPath, "db", "", "SQLite database path")
	flag.Parse()

	if dbPath == "" {
		fmt.Fprintln(os.Stderr, "missing required -db flag")
		os.Exit(1)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	applied, applyErr := migrator.Apply(ctx, db)
	if applyErr != nil {
		fmt.Fprintf(os.Stderr, "apply migrations: %v\n", applyErr)
		os.Exit(1)
	}

	if len(applied) == 0 {
		fmt.Fprintln(os.Stdout, "no pending migrations")
		return
	}

	for _, name := range applied {
		fmt.Fprintf(os.Stdout, "applied %s\n", name)
	}
}
