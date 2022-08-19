package index

import (
	"github.com/blevesearch/bleve/v2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

var (
	log = logging.LoggerForModule()

	// StandardOptions is the search options map for a compliance standard
	StandardOptions = search.Walk(v1.SearchCategory_COMPLIANCE_STANDARD, "standard", (*v1.ComplianceStandard)(nil))
	// ControlOptions is the search options map for a compliance control
	ControlOptions = search.Walk(v1.SearchCategory_COMPLIANCE_CONTROL, "control", (*v1.ComplianceControl)(nil))
)

// Indexer is the interface for Compliance Search
//go:generate mockgen-wrapper
type Indexer interface {
	IndexStandard(standard *v1.ComplianceStandard) error
	DeleteStandard(id string) error
	DeleteControl(id string) error
	SearchStandards(q *v1.Query) ([]search.Result, error)
	SearchControls(q *v1.Query) ([]search.Result, error)
}

type controlWrapper struct {
	*v1.ComplianceControl `json:"control"`
	Type                  string `json:"type"`
}

type standardWrapper struct {
	*v1.ComplianceStandard `json:"standard"`
	Type                   string `json:"type"`
}

type indexer struct {
	indexer bleve.Index
}

// New returns a new indexer for compliance objects
func New(i bleve.Index) Indexer {
	return &indexer{
		indexer: i,
	}
}

// IndexStandard takes in a standard and indexes the standard, the groups, and the controls
func (i *indexer) IndexStandard(standard *v1.ComplianceStandard) error {
	batch := i.indexer.NewBatch()

	if err := batch.Index(standard.GetMetadata().GetId(), &standardWrapper{ComplianceStandard: standard, Type: v1.SearchCategory_COMPLIANCE_STANDARD.String()}); err != nil {
		return err
	}
	for _, c := range standard.GetControls() {
		if err := batch.Index(c.GetId(), &controlWrapper{ComplianceControl: c, Type: v1.SearchCategory_COMPLIANCE_CONTROL.String()}); err != nil {
			return err
		}
	}
	return i.indexer.Batch(batch)
}

func (i *indexer) DeleteStandard(id string) error {
	return i.indexer.Delete(id)
}

func (i *indexer) DeleteControl(id string) error {
	return i.indexer.Delete(id)
}

// SearchStandards searches standards
func (i *indexer) SearchStandards(q *v1.Query) ([]search.Result, error) {
	return blevesearch.RunSearchRequest(v1.SearchCategory_COMPLIANCE_STANDARD, q, i.indexer, StandardOptions)
}

// SearchControls searches controls
func (i *indexer) SearchControls(q *v1.Query) ([]search.Result, error) {
	return blevesearch.RunSearchRequest(v1.SearchCategory_COMPLIANCE_CONTROL, q, i.indexer, ControlOptions)
}
