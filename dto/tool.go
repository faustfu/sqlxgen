package dto

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"io"
)

const (
	EQ = "="
)

func NamedQueryContext[T, S any](ctx context.Context, db DB, sql string, z S) ([]T, error) {
	rows, err := db.NamedQueryContext(ctx, sql, z)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	result := make([]T, 0)
	for rows.Next() {
		var row T
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		result = append(result, row)
	}

	return result, nil

}

type DB interface {
	NamedQueryContext(context.Context, string, interface{}) (*sqlx.Rows, error)
	TX
}

type TX interface {
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)
}

func Close(v io.Closer) {
	_ = v.Close()
}
