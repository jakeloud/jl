package api

import (
	"strings"

	"github.com/jakeloud/jl/entities"
	"github.com/jakeloud/jl/logger"
)

func ClearCacheOp(params apiRequest) error {
	conf, err := entities.GetConf()
	if err != nil {
		return err
	}

	if params.Email == "" {
		return nil
	}
	if len(conf.Users) > 0 {
		isAuth, err := entities.IsAuthenticated(params.Email, params.Password)
		if err != nil || !isAuth {
			return nil
		}
	}

	res, err := entities.ClearCache()
	if err != nil {
		return err
	}

	res = strings.TrimSpace(res)
	lastLine := res
	if idx := strings.LastIndex(res, "\n"); idx != -1 {
		lastLine = res[idx+1:]
	}

	lastLine = strings.ReplaceAll(lastLine, ".", "\\.")
	logMessage := "*Clearing cache* " + lastLine

	if err := logger.Log(logMessage); err != nil {
		return err
	}

	return nil
}
