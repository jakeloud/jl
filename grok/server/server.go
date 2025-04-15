package server

import (
	"embed"
	"encoding/json"
	"io"
	"net/http"

	"github.com/jakeloud/jl/api"
	"github.com/jakeloud/jl/entities"
)

//go:embed static
var staticFiles embed.FS

// fileCache stores static file contents in memory.
var fileCache = make(map[string]string)

// Start initializes and starts the HTTP server.
func Start() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Handle GET requests for static files
			pathname := r.URL.Path
			file := "/index.html"
			if pathname != "/" {
				file = pathname
			}

			// Load file into cache if not already present
			if _, ok := fileCache[file]; !ok {
				data, err := staticFiles.ReadFile("static" + file)
				if err != nil {
					fileCache[file] = "404 Not Found"
				} else {
					fileCache[file] = string(data)
				}
			}

			// Write cached content
			w.Write([]byte(fileCache[file]))
			return
		}

		if r.Method == http.MethodPost {
			// Handle POST requests
			// Limit body size to 1024 bytes
			r.Body = http.MaxBytesReader(w, r.Body, 1024)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var parsedBody map[string]interface{}
			if len(body) > 0 {
				if err := json.Unmarshal(body, &parsedBody); err != nil {
					parsedBody = make(map[string]interface{})
				}
			} else {
				parsedBody = make(map[string]interface{})
			}

			// Delegate to API handler
			api.API(w, r)
			return
		}

		// Unsupported methods
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	return nil

	// Start the server using entities.Start
	return entities.Start(nil)
}
