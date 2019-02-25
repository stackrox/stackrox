package matcher

import (
	"testing"

	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getDeployment(id string, labels map[string]string) *storage.Deployment {
	d := fixtures.GetDeployment()
	d.Id = id
	d.Labels = labels
	return d
}

func TestMapQueries(t *testing.T) {
	indexer, err := globalindex.TempInitializeIndices("")
	require.NoError(t, err)

	deploymentIndexer := deploymentIndex.New(indexer)

	d1 := getDeployment("d1", map[string]string{"h1": "h2", "h3": "h4"})
	d2 := getDeployment("d2", map[string]string{"not-h1": "h2", "h5": "h6", "h7": "h8"})
	d3 := getDeployment("d3", nil)
	d4 := getDeployment("d4", map[string]string{"h1": "not-h2", "h5": "h6", "h7": "h8"})

	require.NoError(t, deploymentIndexer.AddDeployment(d1))
	require.NoError(t, deploymentIndexer.AddDeployment(d2))
	require.NoError(t, deploymentIndexer.AddDeployment(d3))
	require.NoError(t, deploymentIndexer.AddDeployment(d4))

	var cases = []struct {
		key, value  string
		expectedIDs []string
	}{
		//Key and value must exist
		{
			key:         "h1",
			value:       "h2",
			expectedIDs: []string{"d1"},
		},
		// Key must exist and not equal value
		{
			key:         "h1",
			value:       "!h2",
			expectedIDs: []string{"d4"},
		},
		//Key cannot have a value that's not h1 and a value of h2
		{
			key:         "!h1",
			value:       "h2",
			expectedIDs: []string{"d2"},
		},
		//Key cannot have any values that aren't h1 -> h2, also check if value is nil
		{
			key:         "!h1",
			value:       "!h2",
			expectedIDs: []string{"d2", "d3", "d4"},
		},
		// !h1 means that key doesn't exist by itself also check against d3 which is nil
		{
			key:         "!h1",
			expectedIDs: []string{"d2", "d3"},
		},
	}

	for _, c := range cases {
		q := search.NewQueryBuilder().AddMapQuery(search.Label, c.key, c.value).ProtoQuery()
		results, err := deploymentIndexer.Search(q)
		assert.NoError(t, err)
		assert.ElementsMatch(t, c.expectedIDs, search.ResultsToIDs(results))
	}
}
