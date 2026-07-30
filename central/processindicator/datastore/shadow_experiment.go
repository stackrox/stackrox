package datastore

// EXPERIMENT: This file is throwaway instrumentation for PR #22014 validation.
// It manages a shadow table that records both Jaccard and NEM pruning decisions
// side-by-side, enabling empirical comparison on real cluster data.

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	maxRetainedCycles = 10

	shadowTableDDL = `
CREATE TABLE IF NOT EXISTS process_indicators_shadow (
    id VARCHAR NOT NULL,
    prune_cycle_id INTEGER NOT NULL,
    pod_id VARCHAR,
    container_name VARCHAR,
    process_name VARCHAR,
    signal_args TEXT,
    signal_execfilepath VARCHAR,
    jaccard_would_prune BOOLEAN NOT NULL DEFAULT FALSE,
    nem_would_prune BOOLEAN NOT NULL DEFAULT FALSE,
    jaccard_similarity DOUBLE PRECISION,
    matched_anchor_id_jaccard VARCHAR,
    matched_anchor_id_nem VARCHAR,
    captured_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, prune_cycle_id)
);
CREATE INDEX IF NOT EXISTS idx_shadow_cycle ON process_indicators_shadow (prune_cycle_id);
`

	cleanupOldCyclesSQL = `DELETE FROM process_indicators_shadow WHERE prune_cycle_id < $1`
)

func initShadowTable(ctx context.Context, db postgres.DB) error {
	_, err := db.Exec(ctx, shadowTableDDL)
	return err
}

func cleanupOldCycles(ctx context.Context, db postgres.DB, currentCycle int) error {
	cutoff := currentCycle - maxRetainedCycles
	if cutoff <= 0 {
		return nil
	}
	_, err := db.Exec(ctx, cleanupOldCyclesSQL, cutoff)
	return err
}

func recordShadowResults(
	ctx context.Context,
	db postgres.DB,
	cycleID int,
	info processindicator.ProcessWithContainerInfo,
	processes []processindicator.IDAndArgs,
	jaccardResults []pruner.JaccardResult,
	nemResults []pruner.NEMResult,
) error {
	if len(processes) == 0 {
		return nil
	}

	jaccardByID := make(map[string]pruner.JaccardResult, len(jaccardResults))
	for _, r := range jaccardResults {
		jaccardByID[r.ID] = r
	}
	nemByID := make(map[string]pruner.NEMResult, len(nemResults))
	for _, r := range nemResults {
		nemByID[r.ID] = r
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO process_indicators_shadow (id, prune_cycle_id, pod_id, container_name, process_name, signal_args, jaccard_would_prune, nem_would_prune, jaccard_similarity, matched_anchor_id_jaccard, matched_anchor_id_nem) VALUES ")

	args := make([]interface{}, 0, len(processes)*11)
	for i, p := range processes {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 11
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10, base+11))

		jr := jaccardByID[p.ID]
		nr := nemByID[p.ID]

		var jacSim *float64
		if jr.WouldPrune {
			s := jr.Similarity
			jacSim = &s
		}

		args = append(args,
			p.ID,
			cycleID,
			info.PodID,
			info.ContainerName,
			info.ProcessName,
			truncate(p.Args, 4000),
			jr.WouldPrune,
			nr.WouldPrune,
			jacSim,
			nilIfEmpty(jr.MatchedAnchorID),
			nilIfEmpty(nr.MatchedAnchorID),
		)
	}

	sb.WriteString(" ON CONFLICT (id, prune_cycle_id) DO NOTHING")

	_, err := db.Exec(ctx, sb.String(), args...)
	return err
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
