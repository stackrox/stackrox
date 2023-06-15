package globalindex

import (
	"math"

	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/analysis/analyzer/keyword"  // Import the keyword analyzer so that it can be referred to from proto files
	_ "github.com/blevesearch/bleve/analysis/analyzer/standard" // Import the standard analyzer so that it can be referred to from proto files
	"github.com/blevesearch/bleve/index/store/moss"
	"github.com/blevesearch/bleve/index/upsidedown"
	complianceMapping "github.com/stackrox/rox/central/compliance/search"
	"github.com/stackrox/rox/central/globalindex/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (

	// SearchOptionsMap includes options maps that are not required for document mapping
	SearchOptionsMap = func() map[v1.SearchCategory][]search.FieldLabel {
		var searchMap = map[v1.SearchCategory][]search.FieldLabel{
			v1.SearchCategory_COMPLIANCE: complianceMapping.Options,
		}
		entityOptions := mapping.GetEntityOptionsMap()
		for k, v := range entityOptions {
			searchMap[k] = optionsMapToSlice(v)
		}
		return searchMap
	}
)

func optionsMapToSlice(options search.OptionsMap) []search.FieldLabel {
	labels := make([]search.FieldLabel, 0, len(options.Original()))
	for k, v := range options.Original() {
		if v.GetHidden() {
			continue
		}
		labels = append(labels, k)
	}
	return labels
}

func kvConfigForMoss() map[string]interface{} {
	return map[string]interface{}{
		"mossCollectionOptions": map[string]interface{}{
			"MaxPreMergerBatches": math.MaxInt32,
		},
	}
}

// MemOnlyIndex returns a temporary mem-only index.
func MemOnlyIndex() (bleve.Index, error) {
	return bleve.NewUsing("", mapping.GetIndexMapping(), upsidedown.Name, moss.Name, kvConfigForMoss())
}
