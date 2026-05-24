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

func main() {
	if len(os.Args) < 2 {
		runServer(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "token":
		runToken(os.Args[2:])
	case "serve":
		runServer(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "usage: infraview-server [token|serve]")
		os.Exit(1)
	}
}

func runServer(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	noAuth := fs.Bool("no-auth", false, "disable authentication (local dev only)")
	addr := fs.String("addr", ":8080", "listen address")
	fs.Parse(args)

	store, err := storage.New("./infraview.db")
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

func runToken(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: infraview-server token generate --type read|write")
		os.Exit(1)
	}

	if args[0] != "generate" {
		fmt.Fprintf(os.Stderr, "unknown token subcommand: %s\n", args[0])
		os.Exit(1)
	}

	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	tokenType := fs.String("type", "", "token type: read or write")
	fs.Parse(args[1:])

	if *tokenType != "read" && *tokenType != "write" {
		fmt.Fprintln(os.Stderr, "--type must be 'read' or 'write'")
		os.Exit(1)
	}

	t := auth.Generate(auth.Type(*tokenType))

	store, err := storage.New("./infraview.db")
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
