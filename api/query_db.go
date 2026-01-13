package api

import (
	"database/sql"
	"fmt"
	"slices"

	_ "modernc.org/sqlite"

	"github.com/jakeloud/jl/entities"
)

type QueryResult struct {
	Tables []string      `json:"tables"`
	Rows   []interface{} `json:"rows"`
	Count  int           `json:"count"`
}

func showTables(tx *sql.Tx) (tables []string, err error) {
	rows, err := tx.Query(
		`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`,
	)
	if err != nil {
		return tables, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tables = append(tables, t)
	}

	return tables, err
}

func queryTable(tx *sql.Tx, table string) (res []interface{}, count int, err error) {
	query := fmt.Sprintf("SELECT count(*) FROM %s", table)
	err = tx.QueryRow(query).Scan(&count)
	if err != nil {
		return res, count, err
	}

	query = fmt.Sprintf("SELECT * FROM %s", table)
	rows, err := tx.Query(query)
	if err != nil {
		return res, count, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return res, count, err
	}

	for rows.Next() {
		values := make([]any, len(cols))
		pointers := make([]any, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		rows.Scan(pointers...)

		rowMap := make(map[string]any)
		for i, colName := range cols {
			rowMap[colName] = values[i]
		}

		res = append(res, rowMap)
	}

	return res, count, err
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

	path := db.Path
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return res, err
	}

	tx, err := database.Begin()
	if err != nil {
		return res, err
	}
	defer tx.Rollback()

	tables, err := showTables(tx)
	if err != nil {
		return res, err
	}

	res.Tables = tables

	table := params.Table
	if table == "" && len(tables) > 0 {
		table = tables[0]
	}
	if slices.Contains(tables, table) {
		rows, count, err := queryTable(tx, table)
		if err != nil {
			return res, err
		}
		res.Rows = rows
		res.Count = count
	}

	return res, nil
}
