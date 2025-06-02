package singleton

import (
	"github.com/stackrox/rox/central/risk/scorer/component"
	"github.com/stackrox/rox/central/risk/scorer/component/image"
	"github.com/stackrox/rox/central/risk/scorer/component/node"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	imageScorerOnce sync.Once
	imageScorer     component.ImageScorer

	nodeScorerOnce sync.Once
	nodeScorer     component.Scorer
)

// GetImageScorer returns the singleton Scorer object to use when scoring image risk.
func GetImageScorer() component.ImageScorer {
	imageScorerOnce.Do(func() {
		imageScorer = image.NewImageComponentScorer()
	})
	return imageScorer
}

// GetNodeScorer returns the singleton Scorer object to use when scoring node risk.
func GetNodeScorer() component.Scorer {
	nodeScorerOnce.Do(func() {
		nodeScorer = node.NewNodeComponentScorer()
	})
	return nodeScorer
}
