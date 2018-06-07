package blevesearch

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/tokenizer/single"
	"github.com/blevesearch/bleve/index/store/moss"
	"github.com/blevesearch/bleve/index/upsidedown"
	"github.com/blevesearch/bleve/mapping"
)

var (
	logger = logging.LoggerForModule()
)

// Indexer is the Bleve implementation of Indexer
type Indexer struct {
	alertIndex      bleve.Index
	deploymentIndex bleve.Index
	imageIndex      bleve.Index
	policyIndex     bleve.Index
}

// NewIndexer creates a new Indexer based on Bleve
func NewIndexer(path string) (*Indexer, error) {
	b := &Indexer{}
	if err := b.initializeIndices(path); err != nil {
		return nil, err
	}
	return b, nil
}

// This is the custom analyzer definition
func singleTermAnalyzer() map[string]interface{} {
	return map[string]interface{}{
		"type":         custom.Name,
		"char_filters": []string{},
		// single tokenizer means that it takes each field string as a single token (e.g. "the quick brown fox" is not delimited by spaces)
		"tokenizer": single.Name,
		// Ignore case sensitivity
		"token_filters": []string{
			lowercase.Name,
		},
	}
}

func getIndexMapping() *mapping.IndexMappingImpl {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddCustomAnalyzer("single_term", singleTermAnalyzer())
	indexMapping.DefaultAnalyzer = "single_term" // Default to our analyzer

	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.TypeField = "Type"

	indexMapping.AddDocumentMapping(v1.SearchCategory_ALERTS.String(), alertDocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_IMAGES.String(), imageDocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_POLICIES.String(), policyDocumentMap)
	indexMapping.AddDocumentMapping(v1.SearchCategory_DEPLOYMENTS.String(), deploymentDocumentMap)

	disabledSection := bleve.NewDocumentDisabledMapping()
	indexMapping.AddDocumentMapping("_all", disabledSection)

	return indexMapping
}

func (b *Indexer) initializeIndices(mossPath string) error {
	indexMapping := getIndexMapping()

	kvconfig := map[string]interface{}{
		"mossLowerLevelStoreName": "mossStore",
	}

	allIndex, err := bleve.NewUsing(mossPath, indexMapping, upsidedown.Name, moss.Name, kvconfig)
	if err != nil {
		return err
	}

	b.alertIndex = allIndex
	b.deploymentIndex = allIndex
	b.imageIndex = allIndex
	b.policyIndex = allIndex
	return nil
}

// Close closes the open indexes
func (b *Indexer) Close() error {
	if err := b.alertIndex.Close(); err != nil {
		return err
	}
	if err := b.deploymentIndex.Close(); err != nil {
		return err
	}
	if err := b.imageIndex.Close(); err != nil {
		return err
	}
	return b.policyIndex.Close()
}
