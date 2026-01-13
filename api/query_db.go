package api

import (
	"github.com/jakeloud/jl/entities"
        "log"
)


type QueryResult struct {
        Tables []string `json:"tables"`
        Rows []interface{} `json:"rows"`
        TotalRows int `json:"total"`
}

func QueryDBOp(params apiRequest) (res QueryResult, err error) {
	db, err := entities.GetDB(params.Name)
	if err != nil {
		return res, nil
	}

	authenticated, err := entities.IsAuthenticated(params.Email, params.Password)
	if err != nil {
		return res, err
	}
	if !authenticated || params.Name == "" {
		return res, nil
	}

        table := params.Table
        path := db.Path

        log.Printf("query %s; get table \"%s\"", path, table)

	return res, nil
}
