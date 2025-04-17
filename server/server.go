package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/jakeloud/jl/api"
	"github.com/jakeloud/jl/entities"
)

//go:embed dist
var staticFiles embed.FS

func Start() error {
	staticFS := fs.FS(staticFiles)
	staticContent, err := fs.Sub(staticFS, "dist")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(staticContent))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fileServer.ServeHTTP(w, r)
		} else if r.Method == http.MethodPost {
			api.API(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return entities.Start(nil)
}
