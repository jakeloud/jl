package api

import (
	"errors"
	"fmt"

	"github.com/jakeloud/jl/entities"
)

func ConfirmProjectLiveness(params apiRequest) error {
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated {
		return errors.New("authentication required")
	}
	if params.Name == "" || params.Release < 1 {
		return errors.New("project name and release are required")
	}
	if err := entities.ConfirmRelease(params.Name, params.Release); err != nil {
		return fmt.Errorf("failed to confirm release: %v", err)
	}
	return nil
}
