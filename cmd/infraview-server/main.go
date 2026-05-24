package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ky3ow/infraview/internal/server"
	"github.com/ky3ow/infraview/pkg/auth"
	"github.com/ky3ow/infraview/pkg/storage"
)

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func appUsage(name string) {
	fmt.Fprintf(os.Stderr, "Usage: %s [token|serve]\n", name)
	fmt.Fprintf(os.Stderr, "Environment:\n	INFRAVIEW_ADDRESS: %s\n	INFRAVIEW_DB_PATH: %s\n",
		"listen address", "database path")
	os.Exit(1)
}

func main() {
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
		log.Fatalf("storage error: %v", err)
	}

	if *noAuth {
		log.Println("AUTH DISABLED — THIS IS FOR LOCAL DEVELOPMENT ONLY, NOT FOR PRODUCTION")
	}

	srv := server.New(*addr, store, *noAuth)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func runToken(name string, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s token generate --type read|write", name)
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
		log.Fatalf("storage error: %v", err)
	}

	// Delete any existing token of the same type (only one active at a time)
	if err := store.DeleteTokensByType(string(t.Type())); err != nil {
		log.Fatalf("failed to revoke previous token: %v", err)
	}

	if err := store.CreateToken(string(t.Type()), t.Hash()); err != nil {
		log.Fatalf("failed to store token: %v", err)
	}

	// Output raw token to stdout for automation to capture
	fmt.Println(t.Raw())
}
