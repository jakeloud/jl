package main

import (
	"log"
  "net/http"

	"github.com/jakeloud/jl/server"
)

func main() {
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
  log.Fatal(http.ListenAndServe(":666", nil))
}
