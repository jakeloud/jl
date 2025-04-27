package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/server"
	"github.com/jakeloud/jl/setup"
)

func main() {
	dry := flag.Bool("dry", false, "Run in dry mode")
	daemon := flag.Bool("d", false, "Run in daemon mode")
	flag.Parse()
	entities.SetDry(*dry)

	if !(*daemon) {
		setup.Start(*dry)
		return
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Println("Started server")
	log.Fatal(http.ListenAndServe(":666", nil))
}
