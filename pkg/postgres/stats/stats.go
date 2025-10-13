package stats

import (
	"context"
	"time"

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

// PGTupleStat is the slimmed down data model for a single row in pg_stat_user_tables
type PGTupleStat struct {
	NumLiveTuples int64
	NumDeadTuples int64
	Table         string
}

// PGTupleStats is a wrapper around PGTupleStat
type PGTupleStats struct {
	Tuples []*PGTupleStat
	Error  string
}

// GetPGTupleStats returns a tuple struct that wraps the results from the query to pg_stat_user_tables
func GetPGTupleStats(ctx context.Context, db postgres.DB, limit int) *PGTupleStats {
	var tuples PGTupleStats
	rows, err := db.Query(ctx, "SELECT n_live_tup, n_dead_tup, relname FROM pg_stat_user_tables order by n_dead_tup DESC limit $1", limit)
	if err != nil {
		tuples.Error = err.Error()
		return &tuples
	}
	defer rows.Close()

	for rows.Next() {
		var tuple PGTupleStat
		if err := rows.Scan(&tuple.NumLiveTuples, &tuple.NumDeadTuples, &tuple.Table); err != nil {
			tuples.Error = errors.Wrap(err, "error scanning rows from pg_stat_user_tables").Error()
			return &tuples
		}
		tuples.Tuples = append(tuples.Tuples, &tuple)
	}
	if err := rows.Err(); err != nil {
		tuples.Error = err.Error()
	}
	return &tuples
}

// PGAnalyzeStat is the slimmed down data model for a single row in pg_stat_all_tables
type PGAnalyzeStat struct {
	TableName       string
	LastAutoAnalyze *time.Time
	LastAnalyze     *time.Time
	LastAutoVacuum  *time.Time
	LastVacuum      *time.Time
}

// PGAnalyzeStats is a wrapper around PGAnalyzeStat
type PGAnalyzeStats struct {
	AnalyzeStats []*PGAnalyzeStat
	Error        string
}

// GetPGAnalyzeStats returns a tuple struct that wraps the results from the query to pg_stat_all_tables
func GetPGAnalyzeStats(ctx context.Context, db postgres.DB, limit int) *PGAnalyzeStats {
	var analyzeStats PGAnalyzeStats
	rows, err := db.Query(ctx, "SELECT relname, last_autoanalyze, last_analyze, last_autovacuum, last_vacuum FROM pg_stat_all_tables WHERE schemaname = $1 ORDER BY relname limit $2", "public", limit)
	if err != nil {
		analyzeStats.Error = err.Error()
		return &analyzeStats
	}
	defer rows.Close()

	for rows.Next() {
		var analyzeStat PGAnalyzeStat
		if err := rows.Scan(&analyzeStat.TableName, &analyzeStat.LastAutoAnalyze, &analyzeStat.LastAnalyze, &analyzeStat.LastAutoVacuum, &analyzeStat.LastVacuum); err != nil {
			analyzeStats.Error = errors.Wrap(err, "error scanning rows from pg_stat_all_tables").Error()
			return &analyzeStats
		}
		analyzeStats.AnalyzeStats = append(analyzeStats.AnalyzeStats, &analyzeStat)
	}
	if err := rows.Err(); err != nil {
		analyzeStats.Error = err.Error()
	}
	return &analyzeStats
}
