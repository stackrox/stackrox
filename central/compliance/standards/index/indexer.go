package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

var (
	// StandardOptions is the search options map for a compliance standard
	StandardOptions = search.Walk(v1.SearchCategory_COMPLIANCE_STANDARD, "standard", (*v1.ComplianceStandard)(nil))
	// ControlOptions is the search options map for a compliance control
	ControlOptions = search.Walk(v1.SearchCategory_COMPLIANCE_CONTROL, "control", (*v1.ComplianceControl)(nil))
)

// StandardIndexer implements the indexer for standards
//
//go:generate mockgen-wrapper
type StandardIndexer interface {
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}

// ControlIndexer implements the indexer for controls
//
//go:generate mockgen-wrapper
type ControlIndexer interface {
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
