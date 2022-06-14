package image

import (
	"github.com/stackrox/rox/central/risk/multipliers/component/image"
	"github.com/stackrox/rox/central/risk/scorer/component"
	"github.com/stackrox/rox/generated/storage"
)

// NewImageComponentScorer returns a new scorer that encompasses multipliers for evaluating image component risk
func NewImageComponentScorer() component.Scorer {
	return component.NewComponentScorer(storage.RiskSubjectType_IMAGE_COMPONENT, image.NewVulnerabilities())
}
