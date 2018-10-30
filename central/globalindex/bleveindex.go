package globalindex

import (
	"io/ioutil"
	"os"
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
	processIndicatorMapping "github.com/stackrox/rox/central/processindicator/index/mappings"
	secretOptions "github.com/stackrox/rox/central/secret/search/options"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

var (
	// CategoryToOptionsMap is a mapping from search categories to the options map for that category.
	CategoryToOptionsMap = map[v1.SearchCategory]map[search.FieldLabel]*v1.SearchField{
		v1.SearchCategory_ALERTS:             alertMapping.OptionsMap,
		v1.SearchCategory_DEPLOYMENTS:        deploymentMapping.OptionsMap,
		v1.SearchCategory_IMAGES:             imageMapping.OptionsMap,
		v1.SearchCategory_POLICIES:           policyMapping.OptionsMap,
		v1.SearchCategory_SECRETS:            secretOptions.Map,
		v1.SearchCategory_PROCESS_INDICATORS: processIndicatorMapping.OptionsMap,
	}

	logger = logging.LoggerForModule()
)

// TempInitializeIndices initializes the index under the tmp system folder in the specified path.
func TempInitializeIndices(mossPath string) (bleve.Index, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return InitializeIndices(filepath.Join(tmpDir, mossPath))
}

// MemOnlyIndex returns a temporary mem-only index.
func MemOnlyIndex() (bleve.Index, error) {
	return bleve.NewMemOnly(getIndexMapping())
}

// InitializeIndices initializes the index in the specified path.
func InitializeIndices(mossPath string) (bleve.Index, error) {
	indexMapping := getIndexMapping()

	kvconfig := map[string]interface{}{
		"mossLowerLevelStoreName": "mossStore",
	}

	// Bleve requires that the directory we provide is already empty.
	err := os.RemoveAll(mossPath)
	if err != nil {
		logger.Warnf("Could not clean up search index path %s: %v", mossPath, err)
	}
	globalIndex, err := bleve.NewUsing(mossPath, indexMapping, upsidedown.Name, moss.Name, kvconfig)
	if err != nil {
		return nil, err
	}
	go startMonitoring(mossPath)
	return globalIndex, nil
}

func getIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddCustomAnalyzer("single_term", singleTermAnalyzer())
	indexMapping.DefaultAnalyzer = "single_term" // Default to our analyzer

	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.TypeField = "Type"

	for category, optMap := range CategoryToOptionsMap {
		indexMapping.AddDocumentMapping(category.String(), blevesearch.DocumentMappingFromOptionsMap(optMap))
	}

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
