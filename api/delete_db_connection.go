package api

import (
	"github.com/jakeloud/jl/entities"
)

func DeleteDBConnection(params apiRequest) error {
	db, err := entities.GetDB(params.Name)
	if err != nil {
		return nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return err
	}
	if !authenticated || params.Name == "" {
		return nil
	}

	return db.Remove()
}
