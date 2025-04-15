package api

import (
	"encoding/json"
	"net/http"
)

// apiRequest represents the expected JSON body structure.
type apiRequest struct {
	Op            string                 `json:"op"`
	Email         string                 `json:"email"`
	Password      string                 `json:"password"`
	Domain        string                 `json:"domain"`
	Name          string                 `json:"name"`
	Repo          string                 `json:"repo"`
	DockerOptions string                 `json:"dockerOptions"`
	Additional    map[string]interface{} `json:"additional"`
}

// API handles incoming HTTP requests by dispatching to the appropriate operation.
func API(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body apiRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	switch body.Op {
	case "setJakeloudDomainOp":
		err := SetJakeloudDomain(struct {
			Email    string
			Password string
			Domain   string
		}{
			Email:    body.Email,
			Password: body.Password,
			Domain:   body.Domain,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	case "setJakeloudAdditionalOp":
		err := SetJakeloudAdditional(struct {
			Additional map[string]interface{}
			Email      string
			Password   string
		}{
			Additional: body.Additional,
			Email:      body.Email,
			Password:   body.Password,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	case "registerOp":
		err := Register(struct {
			Password string
			Email    string
		}{
			Password: body.Password,
			Email:    body.Email,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	case "getConfOp":
		result, err := GetConf(struct {
			Email    string
			Password string
		}{
			Email:    body.Email,
			Password: body.Password,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
		if result != nil {
			data, _ := json.Marshal(result)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
	case "createAppOp":
		err := CreateApp(struct {
			Domain        string
			Repo          string
			Name          string
			DockerOptions string
			Password      string
			Email         string
		}{
			Domain:        body.Domain,
			Repo:          body.Repo,
			Name:          body.Name,
			DockerOptions: body.DockerOptions,
			Password:      body.Password,
			Email:         body.Email,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	case "deleteAppOp":
		err := DeleteApp(struct {
			Name     string
			Email    string
			Password string
		}{
			Name:     body.Name,
			Email:    body.Email,
			Password: body.Password,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	case "clearCacheOp":
		err := ClearCacheOp(struct {
			Email    string
			Password string
		}{
			Email:    body.Email,
			Password: body.Password,
		})
		if err != nil {
			http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
			return
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"noop"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
}
