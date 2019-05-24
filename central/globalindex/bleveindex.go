package globalindex

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	_ "github.com/blevesearch/bleve/analysis/analyzer/standard" // Import the standard analyzer so that it can be referred to from proto files
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/tokenizer/whitespace"
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/blevesearch/bleve/mapping"
	alertMapping "github.com/stackrox/rox/central/alert/index/mappings"
	clusterMapping "github.com/stackrox/rox/central/cluster/index/mappings"
	complianceMapping "github.com/stackrox/rox/central/compliance/search"
	"github.com/stackrox/rox/central/compliance/standards/index"
	deploymentMapping "github.com/stackrox/rox/central/deployment/mappings"
	imageMapping "github.com/stackrox/rox/central/image/mappings"
	namespaceMapping "github.com/stackrox/rox/central/namespace/index/mappings"
	nodeMapping "github.com/stackrox/rox/central/node/index/mappings"
	policyMapping "github.com/stackrox/rox/central/policy/index/mappings"
	processIndicatorMapping "github.com/stackrox/rox/central/processindicator/index/mappings"
	processWhitelistMapping "github.com/stackrox/rox/central/processwhitelist/index/mappings"
	roleOptions "github.com/stackrox/rox/central/rbac/k8srole/search/mappings"
	roleBindingOptions "github.com/stackrox/rox/central/rbac/k8srolebinding/search/mappings"
	secretOptions "github.com/stackrox/rox/central/secret/search/mappings"
	serviceAccountOptions "github.com/stackrox/rox/central/serviceaccount/search/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/utils"
)

var (

	// SearchOptionsMap includes options maps that are not required for document mapping
	SearchOptionsMap = func() map[v1.SearchCategory][]search.FieldLabel {
		var searchMap = map[v1.SearchCategory][]search.FieldLabel{
			v1.SearchCategory_COMPLIANCE: complianceMapping.Options,
		}
		entityOptions := GetEntityOptionsMap()
		for k, v := range entityOptions {
			searchMap[k] = optionsMapToSlice(v)
		}
		return searchMap
	}

	log = logging.LoggerForModule()
)

// GetEntityOptionsMap is a mapping from search categories to the options
func GetEntityOptionsMap() map[v1.SearchCategory]search.OptionsMap {
	// EntityOptionsMap is a mapping from search categories to the options map for that category.
	// search document maps are also built off this map

	entityOptionsMap := map[v1.SearchCategory]search.OptionsMap{
		v1.SearchCategory_ALERTS:              alertMapping.OptionsMap,
		v1.SearchCategory_DEPLOYMENTS:         deploymentMapping.OptionsMap,
		v1.SearchCategory_IMAGES:              imageMapping.OptionsMap,
		v1.SearchCategory_POLICIES:            policyMapping.OptionsMap,
		v1.SearchCategory_SECRETS:             secretOptions.OptionsMap,
		v1.SearchCategory_PROCESS_INDICATORS:  processIndicatorMapping.OptionsMap,
		v1.SearchCategory_COMPLIANCE_STANDARD: index.StandardOptions,
		v1.SearchCategory_COMPLIANCE_CONTROL:  index.ControlOptions,
		v1.SearchCategory_CLUSTERS:            clusterMapping.OptionsMap,
		v1.SearchCategory_NAMESPACES:          namespaceMapping.OptionsMap,
		v1.SearchCategory_NODES:               nodeMapping.OptionsMap,
		v1.SearchCategory_PROCESS_WHITELISTS:  processWhitelistMapping.OptionsMap,
	}

	if features.K8sRBAC.Enabled() {
		entityOptionsMap[v1.SearchCategory_ROLES] = roleOptions.OptionsMap
		entityOptionsMap[v1.SearchCategory_ROLEBINDINGS] = roleBindingOptions.OptionsMap
		entityOptionsMap[v1.SearchCategory_SERVICE_ACCOUNTS] = serviceAccountOptions.OptionsMap
		entityOptionsMap[v1.SearchCategory_ROLES] = roleOptions.OptionsMap
		entityOptionsMap[v1.SearchCategory_ROLEBINDINGS] = roleBindingOptions.OptionsMap

	}

	return entityOptionsMap

}

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

// TempInitializeIndices initializes the index under the tmp system folder in the specified path.
func TempInitializeIndices(mossPath string) (bleve.Index, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return initializeIndices(filepath.Join(tmpDir, mossPath))
}

// MemOnlyIndex returns a temporary mem-only index.
func MemOnlyIndex() (bleve.Index, error) {
	return bleve.NewMemOnly(getIndexMapping())
}

// InitializeIndices initializes the index in the specified path.
func InitializeIndices(mossPath string) (bleve.Index, error) {
	globalIndex, err := initializeIndices(mossPath)
	if err != nil {
		return nil, err
	}
	go startMonitoring(globalIndex, mossPath)
	return globalIndex, nil
}

func initializeIndices(mossPath string) (bleve.Index, error) {
	indexMapping := getIndexMapping()

	kvconfig := map[string]interface{}{
		// This sounds scary. It's not. It just means that the persistence to disk is not guaranteed
		// which is fine for us because we replay on Central restart
		"unsafe_batch": true,
	}

	// Bleve requires that the directory we provide is already empty.
	err := os.RemoveAll(mossPath)
	if err != nil {
		log.Warnf("Could not clean up search index path %s: %v", mossPath, err)
	}
	globalIndex, err := bleve.NewUsing(mossPath, indexMapping, scorch.Name, scorch.Name, kvconfig)
	if err != nil {
		return nil, err
	}

	return globalIndex, nil
}

func getDefaultDocMapping() *mapping.DocumentMapping {
	return &mapping.DocumentMapping{
		Enabled: false,
		Dynamic: false,
	}
}

func getIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()
	utils.Must(indexMapping.AddCustomAnalyzer("single_term", singleTermAnalyzer()))
	indexMapping.DefaultAnalyzer = "single_term" // Default to our analyzer

	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.TypeField = "Type"
	indexMapping.DefaultMapping = getDefaultDocMapping()

	for category, optMap := range GetEntityOptionsMap() {
		indexMapping.AddDocumentMapping(category.String(), blevesearch.DocumentMappingFromOptionsMap(optMap.Original()))
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
