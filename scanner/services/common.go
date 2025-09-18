package services

import (
	"context"
	"fmt"

	"github.com/quay/claircore"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
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

// parseIndexReport will generate an index report from a Contents payload.
func parseIndexReport(contents *v4.Contents) (*claircore.IndexReport, error) {
	ir, err := mappers.ToClairCoreIndexReport(contents)
	if err != nil {
		// Validation should have captured all conversion errors.
		return nil, fmt.Errorf("internal error: %w", err)
	}
	return ir, nil
}
