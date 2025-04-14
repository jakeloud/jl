package api

import (
	"strings"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
)

// ClearCacheOp clears the Docker cache if the user is authenticated.
func ClearCacheOp(params struct {
	Email    string
	Password string
}) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	if params.Email == "" || (len(conf.Users) > 0 && !entities.IsAuthenticated(params.Email, params.Password)) {
		return nil // Silently return as per original logic
	}

	res, err := entities.ClearCache()
	if err != nil {
		return err
	}

	// Trim and extract the last line
	res = strings.TrimSpace(res)
	lastLine := res
	if idx := strings.LastIndex(res, "\n"); idx != -1 {
		lastLine = res[idx+1:]
	}

	// Escape periods for MarkdownV2
	lastLine = strings.ReplaceAll(lastLine, ".", "\\.")
	logMessage := "*Clearing cache* " + lastLine

	if err := logger.Log(logMessage); err != nil {
		return err
	}

	return nil
}
