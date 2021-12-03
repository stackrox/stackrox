package effectiveaccessscope

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	desc                 string
	cst                  ScopeTreeCompacted
	expectedScopeQueries []string
}

func TestScopeQueries(t *testing.T) {
	testCases := []testCase{
		{
			desc:                 "empty compacted scope tree -> empty scope queries",
			cst:                  map[string][]string{},
			expectedScopeQueries: []string{},
		},
		{
			desc: "single cluster scope tree with all namespaces",
			cst: map[string][]string{
				"prodCluster": {"*"},
			},
			expectedScopeQueries: []string{"Cluster: prodCluster"},
		},
		{
			desc: "single cluster scope tree with specific namespaces",
			cst: map[string][]string{
				"prodCluster": {"webserver", "db"},
			},
			expectedScopeQueries: []string{"Cluster: prodCluster, Namespace: webserver", "Cluster: prodCluster, Namespace: db"},
		},
		{
			desc: "multiple cluster scope tree with specific namespaces",
			cst: map[string][]string{
				"prodCluster": {"webserver", "db"},
				"testCluster": {"test1", "test2"},
			},
			expectedScopeQueries: []string{"Cluster: prodCluster, Namespace: webserver",
				"Cluster: prodCluster, Namespace: db",
				"Cluster: testCluster, Namespace: test1",
				"Cluster: testCluster, Namespace: test2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := tc.cst.ToScopeQueries()
			assert.EqualValues(t, tc.expectedScopeQueries, result)
		})
	}
}
