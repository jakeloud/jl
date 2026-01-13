package api

import (
	"encoding/json"
	"log/slog"
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
	Path          string                 `json:"path"`
        Table   string `json:"table"`
}

func API(w http.ResponseWriter, r *http.Request) {
	var body apiRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		slog.Info("invalid json")
		return
	}

	//slog.Info("API", "params", body)

	var err error

	switch body.Op {
	case "setJakeloudDomainOp":
		err = SetJakeloudDomain(body)
	case "setJakeloudAdditionalOp":
		err = SetJakeloudAdditional(body)
	case "registerOp":
		err = Register(body)
	case "getConfOp":
		result, err := GetConf(body)
		if err == nil && result != nil {
			data, err := json.Marshal(result)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			}
		}
	case "getAppOp":
		result, err := GetApp(body)
		if err == nil && result != nil {
			data, err := json.Marshal(result)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			}
		}
	case "queryDBOp":
		result, err := QueryDBOp(body)
                if err != nil {
                        slog.Info("queryDBOp", err)
                }
		if err == nil {
			data, err := json.Marshal(result)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			}
		}
	case "createAppOp":
		err = CreateApp(body)
	case "createDBConnectionOp":
		err = CreateDBConnection(body)
	case "deleteAppOp":
		err = DeleteApp(body)
	case "deleteDBConnectionOp":
		err = DeleteDBConnection(body)
	case "clearCacheOp":
		err = ClearCacheOp(body)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"noop"}`))
		return
	}
	if err != nil {
		slog.Info("API error", "err", err)
		http.Error(w, `{"message":"operation failed"}`, http.StatusInternalServerError)
	}
}
