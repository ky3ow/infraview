package main

import (
	"log"

	"github.com/ky3ow/infraview/internal/server"
	"github.com/ky3ow/infraview/pkg/storage"
)

func main() {
	addr := ":8080"

	store, err := storage.New("./infraview.db")
	if err != nil {
		log.Fatalf("storage error: %v", err)
	}

	srv := server.New(addr, store)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
