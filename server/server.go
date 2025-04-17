package server

import (
	"embed"
	"net/http"

	"github.com/jakeloud/jl/api"
	"github.com/jakeloud/jl/entities"
)

//go:embed static
var staticFiles embed.FS

var fileCache = make(map[string]string)

func Start() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			file := r.URL.Path
			if file == "/" {
				file = "/index.html"
			}

			if _, ok := fileCache[file]; !ok {
				data, err := staticFiles.ReadFile("static" + file)
				if err != nil {
					fileCache[file] = "404 Not Found"
				} else {
					fileCache[file] = string(data)
				}
			}

			w.Write([]byte(fileCache[file]))
		} else if r.Method == http.MethodPost {
			api.API(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return entities.Start(nil)
}
