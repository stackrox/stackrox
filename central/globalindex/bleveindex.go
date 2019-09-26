package globalindex

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/analysis/analyzer/standard" // Import the standard analyzer so that it can be referred to from proto files
	"github.com/blevesearch/bleve/index/scorch"
	"github.com/blevesearch/bleve/index/store/moss"
	"github.com/blevesearch/bleve/index/upsidedown"
	bleveMapping "github.com/blevesearch/bleve/mapping"
	complianceMapping "github.com/stackrox/rox/central/compliance/search"
	"github.com/stackrox/rox/central/globalindex/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/blevehelper"
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
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	return initializeIndices(filepath.Join(tmpDir, scorchPath))
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

// InitializeIndices initializes the index in the specified path.
func InitializeIndices(scorchPath string) (bleve.Index, error) {
	globalIndex, err := initializeIndices(scorchPath)
	if err != nil {
		return nil, err
	}
	go startMonitoring(globalIndex, scorchPath)
	return globalIndex, nil
}

func initializeIndices(scorchPath string) (bleve.Index, error) {
	kvconfig := map[string]interface{}{
		// Persist the index
		"unsafe_batch": false,
	}

	var globalIndex bleve.Index
	if _, err := os.Stat(filepath.Join(scorchPath, "index_meta.json")); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		globalIndex, err = bleve.NewUsing(scorchPath, mapping.GetIndexMapping(), scorch.Name, scorch.Name, kvconfig)
		if err != nil {
			return nil, err
		}
	} else {
		globalIndex, err = bleve.OpenUsing(scorchPath, kvconfig)
		if err != nil {
			return nil, err
		}

		// This implies that the index mapping has changed and therefore we should reindex everything
		// This can only happen on upgrades
		if !compareMappings(globalIndex.Mapping(), mapping.GetIndexMapping()) {
			log.Info("[STARTUP] Found new index mapping. Removing index and rebuilding")
			if err := globalIndex.Close(); err != nil {
				log.Errorf("error closing global index: %v", err)
			}
			if err := os.RemoveAll(scorchPath); err != nil {
				log.Errorf("error removing scorch path: %v", err)
			}
			return initializeIndices(scorchPath)
		}
	}
	globalIndex.SetName(blevehelper.GlobalIndexName)

	return globalIndex, nil
}

// compareMappings marshals the index mappings into JSON (which is sorted and deterministic) and then compares the bytes
// this will determine if the index mapping has changed and the index needs to be rebuilt
func compareMappings(im1, im2 bleveMapping.IndexMapping) bool {
	bytes1, err := json.Marshal(im1)
	if err != nil {
		return false
	}
	bytes2, err := json.Marshal(im2)
	if err != nil {
		return false
	}
	return bytes.Equal(bytes1, bytes2)
}
