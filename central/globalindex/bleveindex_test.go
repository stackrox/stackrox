package globalindex

import (
	"encoding/json"
	"testing"

	bleveMapping "github.com/blevesearch/bleve/mapping"
	"github.com/stackrox/rox/central/globalindex/mapping"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareMapping(t *testing.T) {
	indexMapping := mapping.GetIndexMapping()

	tmpDir := t.TempDir()

	index, err := TempInitializeIndices(tmpDir)
	require.NoError(t, err)

	assert.True(t, compareMappings(indexMapping, index.Mapping(), ""))
	assert.True(t, compareMappings(indexMapping, index.Mapping(), v1.SearchCategory_ALERTS.String()))

	// close and open index and check mapping
	assert.NoError(t, index.Close())
	index, err = initializeIndices(tmpDir, EphemeralIndex, "")
	require.NoError(t, err)
	assert.True(t, compareMappings(indexMapping, index.Mapping(), ""))
	assert.True(t, compareMappings(indexMapping, index.Mapping(), v1.SearchCategory_ALERTS.String()))

	// Now change the indexMapping that is being compared against
	// We need to marshal and unmarshal it as getIndexMapping() uses the same underlying pointer
	bytes, err := json.Marshal(indexMapping)
	require.NoError(t, err)
	var newIndexMapping bleveMapping.IndexMappingImpl
	require.NoError(t, json.Unmarshal(bytes, &newIndexMapping))

	newIndexMapping.TypeMapping[v1.SearchCategory_ALERTS.String()].Properties["list_alert"].Properties["state"].Fields[0].Store = false

	assert.False(t, compareMappings(&newIndexMapping, index.Mapping(), ""))
	assert.False(t, compareMappings(&newIndexMapping, index.Mapping(), v1.SearchCategory_ALERTS.String()))
}
