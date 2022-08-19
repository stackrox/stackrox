package globalindex

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"  // Import the keyword analyzer so that it can be referred to from proto files
	_ "github.com/blevesearch/bleve/v2/analysis/analyzer/standard" // Import the standard analyzer so that it can be referred to from proto files
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/blevesearch/bleve/v2/index/upsidedown"
	"github.com/blevesearch/bleve/v2/index/upsidedown/store/gtreap"
	bleveMapping "github.com/blevesearch/bleve/v2/mapping"
	complianceMapping "github.com/stackrox/rox/central/compliance/search"
	"github.com/stackrox/rox/central/globalindex/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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

	log = logging.LoggerForModule()
)

// IndexPersisted describes if the index should be persisted
type IndexPersisted int

const (
	// PersistedIndex means that the index should be persisted
	PersistedIndex IndexPersisted = iota
	// EphemeralIndex means that the index will be rebuilt on Central start
	EphemeralIndex
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

// TempInitializeIndices initializes the index under the tmp system folder in the specified path.
func TempInitializeIndices(scorchPath string) (bleve.Index, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}
	return initializeIndices(filepath.Join(tmpDir, scorchPath), EphemeralIndex, "")
}

// MemOnlyIndex returns a temporary mem-only index.
func MemOnlyIndex() (bleve.Index, error) {
	return bleve.NewUsing("", mapping.GetIndexMapping(), upsidedown.Name, gtreap.Name, nil)
}

// InitializeIndices initializes the index in the specified path.
func InitializeIndices(name, scorchPath string, persisted IndexPersisted, typeString string) (bleve.Index, error) {
	globalIndex, err := initializeIndices(scorchPath, persisted, typeString)
	if err != nil {
		return nil, err
	}
	go startMonitoring(globalIndex, name, scorchPath)
	return globalIndex, nil
}

func initializeIndices(scorchPath string, indexPersisted IndexPersisted, typeString string) (bleve.Index, error) {
	kvconfig := map[string]interface{}{
		// Determines if we should persist the index
		// false means persisted and true means *not* persisted
		"unsafe_batch": indexPersisted == EphemeralIndex,
	}

	if _, err := os.Stat(filepath.Join(scorchPath, "index_meta.json")); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		globalIndex, err := bleve.NewUsing(scorchPath, mapping.GetIndexMapping(), scorch.Name, scorch.Name, kvconfig)
		if err != nil {
			return nil, err
		}
		globalIndex.SetName(scorchPath)
		return globalIndex, nil
	}

	//nolint:staticcheck // SA4023 globalIndex being always non-nil is not documented behavior.
	globalIndex, err := bleve.OpenUsing(scorchPath, kvconfig)
	if err != nil {
		log.Errorf("Error opening Bleve index: %q %v. Removing index and retrying from scratch...", scorchPath, err)
		//nolint:staticcheck // SA4023 globalIndex being always non-nil is not documented behavior.
		if globalIndex != nil {
			_ = globalIndex.Close()
		}
		if err := os.RemoveAll(scorchPath); err != nil {
			log.Panicf("error removing scorch path: %v", err)
		}
		return initializeIndices(scorchPath, indexPersisted, "")
	}

	// This implies that the index mapping has changed and therefore we should reindex everything
	// This can only happen on upgrades
	if !compareMappings(globalIndex.Mapping(), mapping.GetIndexMapping(), typeString) {
		log.Info("[STARTUP] Found new index mapping. Removing index and rebuilding")
		if err := globalIndex.Close(); err != nil {
			log.Errorf("error closing global index: %v", err)
		}
		if err := os.RemoveAll(scorchPath); err != nil {
			log.Errorf("error removing scorch path: %v", err)
		}
		return initializeIndices(scorchPath, indexPersisted, "")
	}

	return globalIndex, nil
}

// compareMappings marshals the index mappings into JSON (which is sorted and deterministic) and then compares the bytes
// this will determine if the index mapping has changed and the index needs to be rebuilt
func compareMappings(im1, im2 bleveMapping.IndexMapping, typeString string) bool {
	bytes1, err := json.Marshal(im1)
	if err != nil {
		return false
	}
	bytes2, err := json.Marshal(im2)
	if err != nil {
		return false
	}

	// If a type string is passed, then there is a single object type in this index
	// so we can look to see if the doc map for that single type has changed. If it has not,
	// then there is no need to remove and reindex
	// In the case, where there are multiple objects in an index, do a full rebuild if the doc mapping
	// has changed at all (very frequently)
	if typeString == "" {
		return bytes.Equal(bytes1, bytes2)
	}

	var m1 map[string]interface{}
	if err := json.Unmarshal(bytes1, &m1); err != nil {
		return false
	}
	m1Types, ok := m1["types"].(map[string]interface{})
	if !ok {
		return false
	}
	// Exclude the other type mappings as they do not influence this index
	delete(m1, "types")

	var m2 map[string]interface{}
	if err := json.Unmarshal(bytes2, &m2); err != nil {
		return false
	}
	m2Types, ok := m2["types"].(map[string]interface{})
	if !ok {
		return false
	}
	// Exclude the other type mappings as they do not influence this index
	delete(m2, "types")

	// Return false if the specific typed doc map for this index has changed
	if !reflect.DeepEqual(m1Types[typeString], m2Types[typeString]) {
		return false
	}
	// Check if global variables have changed (e.g. default analyzer, etc)
	return reflect.DeepEqual(m1, m2)
}
