package api

import (
	"fmt"

	"github.com/jakeloud/jl/entities"
)

func CreateDBConnection(params apiRequest) error {
	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return fmt.Errorf("authentication check failed: %v", err)
	}
	if !authenticated || params.Path == "" || params.Name == "" {
		return nil
	}

	// Create new DB instance
	db := entities.DB{
		Name: params.Name,
		Path: params.Path,
	}

	// Save the DB
	if err := db.Save(); err != nil {
		return fmt.Errorf("failed to save db: %v", err)
	}

	return nil
}
