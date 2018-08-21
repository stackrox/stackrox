package globalindex

import (
	"io/ioutil"
	"path/filepath"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/tokenizer/whitespace"
	"github.com/blevesearch/bleve/index/store/moss"
	"github.com/blevesearch/bleve/index/upsidedown"
	"github.com/blevesearch/bleve/mapping"
	alertMapping "github.com/stackrox/rox/central/alert/index/mappings"
	deploymentMapping "github.com/stackrox/rox/central/deployment/index/mappings"
	imageMapping "github.com/stackrox/rox/central/image/index/mappings"
	policyMapping "github.com/stackrox/rox/central/policy/index/mappings"
	secretMapping "github.com/stackrox/rox/central/secret/index/mapping"
	"github.com/stackrox/rox/generated/api/v1"
)

// TempInitializeIndices initializes the index under the tmp system folder in the specified path.
func TempInitializeIndices(mossPath string) (bleve.Index, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return InitializeIndices(filepath.Join(tmpDir, mossPath))
}

// InitializeIndices initializes the index in the specified path.
func InitializeIndices(mossPath string) (bleve.Index, error) {
	indexMapping := getIndexMapping()

	kvconfig := map[string]interface{}{
		"mossLowerLevelStoreName": "mossStore",
	}

	globalIndex, err := bleve.NewUsing(mossPath, indexMapping, upsidedown.Name, moss.Name, kvconfig)
	if err != nil {
		return nil, err
	}
	return globalIndex, nil
}

func getIndexMapping() *mapping.IndexMappingImpl {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddCustomAnalyzer("single_term", singleTermAnalyzer())
	indexMapping.DefaultAnalyzer = "single_term" // Default to our analyzer

	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.TypeField = "Type"

	indexMapping.AddDocumentMapping(v1.SearchCategory_ALERTS.String(), alertMapping.DocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_IMAGES.String(), imageMapping.DocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_POLICIES.String(), policyMapping.DocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_DEPLOYMENTS.String(), deploymentMapping.DocumentMap)

	// Support indexing secrets and relationships.
	indexMapping.AddDocumentMapping(v1.SearchCategory_SECRETS.String(), secretMapping.DocumentMap)

	disabledSection := bleve.NewDocumentDisabledMapping()
	indexMapping.AddDocumentMapping("_all", disabledSection)

	return indexMapping
}

// This is the custom analyzer definition
func singleTermAnalyzer() map[string]interface{} {
	return map[string]interface{}{
		"type":         custom.Name,
		"char_filters": []string{},
		"tokenizer":    whitespace.Name,
		// Ignore case sensitivity
		"token_filters": []string{
			lowercase.Name,
		},
	}
}
