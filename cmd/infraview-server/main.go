package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ky3ow/infraview/internal/server"
	"github.com/ky3ow/infraview/pkg/auth"
	"github.com/ky3ow/infraview/pkg/log"
	"github.com/ky3ow/infraview/pkg/storage"
)

var logger = log.New("main")

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func appUsage(name string) {
	fmt.Fprintf(os.Stderr, "Usage: %s [token|serve]\n", name)
	fmt.Fprintf(os.Stderr, "Environment:\n")
	fmt.Fprintf(os.Stderr, "\tINFRAVIEW_ADDRESS: listen address\n")
	fmt.Fprintf(os.Stderr, "\tINFRAVIEW_DB_PATH: database path\n")
	fmt.Fprintf(os.Stderr, "\tLOG_LEVEL:         log level (debug|info|warn|error)\n")
	fmt.Fprintf(os.Stderr, "\tLOG_FORMAT:        log format (text|json)\n")
	os.Exit(1)
}

func main() {
	if err := log.Init(env("LOG_LEVEL", "warn"), env("LOG_FORMAT", "json")); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	name := os.Args[0]
	if len(os.Args) < 2 {
		runServer(name, os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "token":
		runToken(name, os.Args[2:])
	case "serve":
		runServer(name, os.Args[2:])
	case "-h", "-help":
		appUsage(name)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		appUsage(name)
	}
}

func runServer(name string, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	noAuth := fs.Bool("no-auth", false, "disable authentication (local dev only)")
	addr := fs.String("addr", env("INFRAVIEW_ADDRESS", ":8080"), "listen address")
	dbPath := fs.String("db", env("INFRAVIEW_DB_PATH", "./infraview.db"), "database path")
	fs.Parse(args)

	store, err := storage.New(*dbPath)
	if err != nil {
		logger.Error("failed to open storage", "error", err)
		os.Exit(1)
	}

	if *noAuth {
		logger.Warn("auth disabled — for local development only")
	}

	srv := server.New(*addr, store, *noAuth)
	logger.Info("starting server", "addr", *addr)
	if err := srv.Start(); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func runToken(name string, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s token generate --type read|write\n", name)
		os.Exit(1)
	}

	if args[0] != "generate" {
		fmt.Fprintf(os.Stderr, "Unknown token subcommand: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage of token: %s token generate\n", name)
		os.Exit(1)
	}

	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	tokenType := fs.String("type", "", "token type: read|write")
	dbPath := fs.String("db", env("INFRAVIEW_DB_PATH", "./infraview.db"), "database path")
	fs.Parse(args[1:])

	if *tokenType != "read" && *tokenType != "write" {
		fmt.Fprintf(os.Stderr, "Unknown token type: \"%s\"\n", *tokenType)
		fs.Usage()
		os.Exit(1)
	}

	t := auth.Generate(auth.Type(*tokenType))

	store, err := storage.New(*dbPath)
	if err != nil {
		logger.Error("failed to open storage", "error", err)
		os.Exit(1)
	}

	// Delete any existing token of the same type (only one active at a time)
	if err := store.DeleteTokensByType(string(t.Type())); err != nil {
		logger.Error("failed to revoke previous token", "error", err)
		os.Exit(1)
	}

	if err := store.CreateToken(string(t.Type()), t.Hash()); err != nil {
		logger.Error("failed to store token", "error", err)
		os.Exit(1)
	}

	// Output raw token to stdout for automation to capture
	fmt.Println(t.Raw())
}
