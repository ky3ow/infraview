package main

import (
	"log"

	"github.com/ky3ow/infraview/internal/server"
)

func main() {
	addr := ":8080"
	srv := server.New(addr)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
