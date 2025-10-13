package standards

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexer(t *testing.T) {
	registry, err := NewRegistry(nil, metadata.AllStandards...)
	require.NoError(t, err)
	results, err := registry.SearchStandards(search.NewQueryBuilder().AddStrings(search.StandardID, "pci").ProtoQuery())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "PCI_DSS_3_2", results[0].ID)

	results, err = registry.SearchControls(search.NewQueryBuilder().AddExactMatches(search.Control, "1.1.1").AddStrings(search.StandardID, "pci").ProtoQuery())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "PCI_DSS_3_2:1_1_1", results[0].ID)

	results, err = registry.SearchControls(search.NewQueryBuilder().AddExactMatches(search.ControlID, "PCI_DSS_3_2:1_1_1").AddStrings(search.StandardID, "pci").ProtoQuery())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "PCI_DSS_3_2:1_1_1", results[0].ID)
}
