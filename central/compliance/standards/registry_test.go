package standards

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/standards/index"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexer(t *testing.T) {
	globalIdx, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)
	defer globalIdx.Close()

	standardIdx := index.New(globalIdx)
	registry := NewRegistry(standardIdx, nil)
	registry.RegisterStandards(metadata.AllStandards...)
	results, err := registry.SearchStandards(search.NewQueryBuilder().AddStrings(search.StandardID, "pci").ProtoQuery())
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}
