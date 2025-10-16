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

	ngs := &v1.NetworkGraphScope{}
	ngs.SetQuery("Orchestrator Component:false")
	ngs2 := &v1.NetworkGraphScope{}
	ngs2.SetQuery("Orchestrator Component:false")
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
			desc:   "query; non-orchestrator component scope",
			rawQ:   "Deployment:dep",
			scope:  ngs,
			depQ:   search.ConjunctionQuery(search.ConjunctionQuery(clusterQ, baseScopeQ), depNameQ),
			scopeQ: search.ConjunctionQuery(clusterQ, baseScopeQ),
		},
		{
			desc:   "no query; non-orchestrator component scope",
			scope:  ngs2,
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
			info: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Discovered: true,
				}.Build(),
			}.Build(),
			expected: true,
		},

		// is external but not discovered
		{
			info: storage.NetworkEntityInfo_builder{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				ExternalSource: storage.NetworkEntityInfo_ExternalSource_builder{
					Discovered: false,
				}.Build(),
			}.Build(),
			expected: false,
		},

		// neither external or discovered
		{
			info: storage.NetworkEntityInfo_builder{
				Type:       storage.NetworkEntityInfo_DEPLOYMENT,
				Deployment: &storage.NetworkEntityInfo_Deployment{},
			}.Build(),
			expected: false,
		},
	} {
		assert.Equal(t, tc.expected, IsExternalDiscovered(tc.info))
	}
}
