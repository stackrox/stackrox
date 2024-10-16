package networkgraph

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestGetQueries(t *testing.T) {
	clusterQ := search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery()
	depNameQ := search.NewQueryBuilder().AddStrings(search.DeploymentName, "dep").ProtoQuery()
	baseScopeQ := search.NewQueryBuilder().AddStrings(search.OrchestratorComponent, "false").ProtoQuery()

	for _, tc := range []struct {
		desc   string
		rawQ   string
		scope  *v1.NetworkGraphScope
		depQ   *v1.Query
		scopeQ *v1.Query
	}{
		{
			desc:   "query; no scope",
			rawQ:   "Deployment:dep",
			depQ:   search.ConjunctionQuery(clusterQ, depNameQ),
			scopeQ: clusterQ,
		},
		{
			desc: "query; non-orchestrator component scope",
			rawQ: "Deployment:dep",
			scope: &v1.NetworkGraphScope{
				Query: "Orchestrator Component:false",
			},
			depQ:   search.ConjunctionQuery(search.ConjunctionQuery(clusterQ, baseScopeQ), depNameQ),
			scopeQ: search.ConjunctionQuery(clusterQ, baseScopeQ),
		},
		{
			desc: "no query; non-orchestrator component scope",
			scope: &v1.NetworkGraphScope{
				Query: "Orchestrator Component:false",
			},
			depQ:   search.ConjunctionQuery(clusterQ, baseScopeQ),
			scopeQ: search.ConjunctionQuery(clusterQ, baseScopeQ),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			actualDepQ, actualScopeQ, err := GetFilterAndScopeQueries("c1", tc.rawQ, tc.scope)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.depQ, actualDepQ)
			protoassert.Equal(t, tc.scopeQ, actualScopeQ)
		})
	}
}

func TestIsExternalDiscovered(t *testing.T) {
	for _, tc := range []struct {
		info     *storage.NetworkEntityInfo
		expected bool
	}{
		// is external and discovered
		{
			info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Discovered: true,
					},
				},
			},
			expected: true,
		},

		// is external but not discovered
		{
			info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Discovered: false,
					},
				},
			},
			expected: false,
		},

		// neither external or discovered
		{
			info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Desc: &storage.NetworkEntityInfo_Deployment_{},
			},
			expected: false,
		},
	} {
		assert.Equal(t, tc.expected, IsExternalDiscovered(tc.info))
	}
}
