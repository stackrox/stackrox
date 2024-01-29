package stats

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/postgres"
)

// PGStatStatement is the slimmed down data model for a single row in pg_stat_statements
type PGStatStatement struct {
	TotalExecTimeMS  float64
	MaxExecTimeMS    float64
	MeanExecTimeMS   float64
	StddevExecTimeMS float64
	Calls            int64
	Rows             int64
	Query            string
}

// PGStatStatements is a wrapper around PGStatStatement
type PGStatStatements struct {
	Statements []*PGStatStatement
	Error      string
}

// GetPGStatStatements returns a statements struct that wraps the results from the query to pg_stat_statements
func GetPGStatStatements(ctx context.Context, db postgres.DB, limit int) *PGStatStatements {
	var statements PGStatStatements
	rows, err := db.Query(ctx, "select total_exec_time, max_exec_time, mean_exec_time, stddev_exec_time, calls, rows, substr(query, 1, 1000) from pg_stat_statements order by total_exec_time desc limit $1", limit)
	if err != nil {
		statements.Error = err.Error()
		return &statements
	}
	defer rows.Close()

	for rows.Next() {
		var statement PGStatStatement
		if err := rows.Scan(&statement.TotalExecTimeMS, &statement.MaxExecTimeMS, &statement.MeanExecTimeMS, &statement.StddevExecTimeMS, &statement.Calls, &statement.Rows, &statement.Query); err != nil {
			statements.Error = errors.Wrap(err, "error scanning rows from pg_stat_statements").Error()
			return &statements
		}
		statements.Statements = append(statements.Statements, &statement)
	}
	if err := rows.Err(); err != nil {
		statements.Error = err.Error()
	}
	return &statements
}

// ResetPGStatStatements resets the pg_stat_statements via pg_stat_statements_reset()
func ResetPGStatStatements(ctx context.Context, db postgres.DB) error {
	_, err := db.Exec(ctx, "select pg_stat_statements_reset()")
	return err
}
