package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/server"
)

func main() {
	dry := flag.Bool("dry", false, "Run in dry mode")
	flag.Parse()
	entities.SetDry(*dry)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Fatal(http.ListenAndServe(":666", nil))
}
