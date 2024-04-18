package services

import (
	"context"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/scanner/indexer"
)

// getClairIndexReport is a wrapper around indexer.GetIndexReport to return
// errox.NotFound when the report does not exist or if it is not successful.
func getClairIndexReport(ctx context.Context, indexer indexer.ReportGetter, hashID string) (*claircore.IndexReport, error) {
	ir, found, err := indexer.GetIndexReport(ctx, hashID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.Newf("report %q not found", hashID)
	}
	if !ir.Success {
		return nil, errox.NotFound.Newf("report failed in state %q: %s", ir.State, ir.Err)
	}
	return ir, nil
}

// TODO: We could merge both functions with a bit of work
func getClairIndexReportFromHash(ctx context.Context, indexer indexer.ReportGetter, manifestID string) (*claircore.IndexReport, error) {
	ir, found, err := indexer.GetIndexReportFromHash(ctx, manifestID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound.Newf("report with id %q not found", manifestID)
	}
	if !ir.Success {
		return nil, errox.NotFound.Newf("report failed in state %q: %s", ir.State, ir.Err)
	}
	return ir, nil
}
