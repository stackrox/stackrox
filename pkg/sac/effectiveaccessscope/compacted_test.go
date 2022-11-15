package effectiveaccessscope

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeQueries(t *testing.T) {
	type testCase struct {
		desc                 string
		stc                  ScopeTreeCompacted
		expectedScopeQueries []string
	}
	testCases := []testCase{
		{
			desc:                 "empty compacted scope tree -> empty scope queries",
			stc:                  map[string][]string{},
			expectedScopeQueries: []string{},
		},
		{
			desc: "single cluster scope tree with all namespaces",
			stc: map[string][]string{
				"prodCluster": {"*"},
			},
			expectedScopeQueries: []string{`Cluster:"prodCluster"`},
		},
		{
			desc: "single cluster scope tree with specific namespaces",
			stc: map[string][]string{
				"prodCluster": {"webserver", "db"},
			},
			expectedScopeQueries: []string{`Cluster:"prodCluster"+Namespace:"webserver"`, `Cluster:"prodCluster"+Namespace:"db"`},
		},
		{
			desc: "multiple cluster scope tree with specific namespaces",
			stc: map[string][]string{
				"prodCluster": {"webserver", "db"},
				"testCluster": {"test1", "test2"},
			},
			expectedScopeQueries: []string{`Cluster:"prodCluster"+Namespace:"webserver"`,
				`Cluster:"prodCluster"+Namespace:"db"`,
				`Cluster:"testCluster"+Namespace:"test1"`,
				`Cluster:"testCluster"+Namespace:"test2"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := tc.stc.ToScopeQueries()
			assert.ElementsMatch(t, tc.expectedScopeQueries, result)
		})
	}
}
