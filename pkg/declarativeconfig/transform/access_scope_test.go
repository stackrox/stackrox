package transform

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformAccessScope(t *testing.T) {
	at := newAccessScopeTransform()
	msgs, err := at.Transform(&declarativeconfig.AuthProvider{})
	assert.Nil(t, msgs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestTransformAccessScope(t *testing.T) {
	at := newAccessScopeTransform()

	simpleAccessScopeType := reflect.TypeOf((*storage.SimpleAccessScope)(nil))

	// 1. Access scope with empty rules mimicking an unrestricted scope.
	scopeConfig := &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules:       declarativeconfig.Rules{},
	}

	msgs, err := at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg := msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok := msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	assert.Empty(t, scopeProto.GetRules().GetIncludedClusters())
	assert.Empty(t, scopeProto.GetRules().GetIncludedNamespaces())
	assert.Empty(t, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetClusterLabelSelectors())

	// 2. Access scope with cluster label selectors.
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			ClusterLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{
					{
						Key:      "a",
						Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
						Values:   []string{"a", "b", "c"}}}}}},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	compareLabelSelectors(t, scopeConfig.Rules.ClusterLabelSelectors, scopeProto.GetRules().GetClusterLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedClusters())
	assert.Empty(t, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedNamespaces())

	// 3. Access scope with namespace label selectors.
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			NamespaceLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{
					{
						Key:      "a",
						Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
						Values:   []string{"a", "b", "c"}}}}}},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	compareLabelSelectors(t, scopeConfig.Rules.NamespaceLabelSelectors, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedClusters())
	assert.Empty(t, scopeProto.GetRules().GetClusterLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedNamespaces())

	// 4. Access scope with cluster and label selectors.
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			ClusterLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{{
					Key:      "a",
					Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
					Values:   []string{"a", "b", "c"},
				}}},
			},
			NamespaceLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{
					{
						Key:      "a",
						Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
						Values:   []string{"a", "b", "c"}}}}}},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	compareLabelSelectors(t, scopeConfig.Rules.ClusterLabelSelectors, scopeProto.GetRules().GetClusterLabelSelectors())
	compareLabelSelectors(t, scopeConfig.Rules.NamespaceLabelSelectors, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedClusters())
	assert.Empty(t, scopeProto.GetRules().GetIncludedNamespaces())

	// 5. Access scope with included clusters.
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			IncludedObjects: []declarativeconfig.IncludedObject{
				{Cluster: "clusterA"},
				{Cluster: "clusterB"},
				{Cluster: "clusterC"},
			},
		},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	assert.Empty(t, scopeProto.GetRules().GetClusterLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.ElementsMatch(t, []string{"clusterA", "clusterB", "clusterC"}, scopeProto.GetRules().GetIncludedClusters())

	// 6. Access scope with included clusters and namespaces.
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			IncludedObjects: []declarativeconfig.IncludedObject{
				{Cluster: "clusterA", Namespaces: []string{"NamespaceA", "NamespaceB"}},
				{Cluster: "clusterB", Namespaces: []string{"NamespaceC", "NamespaceD"}},
			},
		},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	assert.Empty(t, scopeProto.GetRules().GetClusterLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Empty(t, scopeProto.GetRules().GetIncludedClusters())
	expectedNamespaces := []*storage.SimpleAccessScope_Rules_Namespace{
		{
			ClusterName:   "clusterA",
			NamespaceName: "NamespaceA",
		},
		{
			ClusterName:   "clusterA",
			NamespaceName: "NamespaceB",
		},
		{
			ClusterName:   "clusterB",
			NamespaceName: "NamespaceC",
		},
		{
			ClusterName:   "clusterB",
			NamespaceName: "NamespaceD",
		},
	}
	assert.ElementsMatch(t, expectedNamespaces, scopeProto.GetRules().GetIncludedNamespaces())

	// 7. Access scope with "everything-and-the-kitchen-sink", i.e. cluster/namespace label selectors, clusters, and
	//    namespaces
	scopeConfig = &declarativeconfig.AccessScope{
		Name:        "test-scope",
		Description: "test description",
		Rules: declarativeconfig.Rules{
			IncludedObjects: []declarativeconfig.IncludedObject{
				{Cluster: "clusterA", Namespaces: []string{"NamespaceA", "NamespaceB"}},
				{Cluster: "clusterB", Namespaces: []string{"NamespaceC", "NamespaceD"}},
				{Cluster: "clusterC"},
			},
			ClusterLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{{
					Key:      "a",
					Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
					Values:   []string{"a", "b", "c"},
				}}},
			},
			NamespaceLabelSelectors: []declarativeconfig.LabelSelector{
				{Requirements: []declarativeconfig.Requirement{
					{
						Key:      "a",
						Operator: declarativeconfig.Operator(storage.LabelSelector_IN),
						Values:   []string{"a", "b", "c"}}}}}},
	}
	msgs, err = at.Transform(scopeConfig)
	assert.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs, simpleAccessScopeType)
	msg = msgs[simpleAccessScopeType]
	require.Len(t, msg, 1)
	scopeProto, ok = msg[0].(*storage.SimpleAccessScope)
	require.True(t, ok)
	assert.Equal(t, scopeConfig.Name, scopeProto.GetName())
	assert.Equal(t, scopeConfig.Description, scopeProto.GetDescription())
	assert.Equal(t, storage.Traits_DECLARATIVE, scopeProto.GetTraits().GetOrigin())
	compareLabelSelectors(t, scopeConfig.Rules.ClusterLabelSelectors, scopeProto.GetRules().GetClusterLabelSelectors())
	compareLabelSelectors(t, scopeConfig.Rules.NamespaceLabelSelectors, scopeProto.GetRules().GetNamespaceLabelSelectors())
	assert.Equal(t, []string{"clusterC"}, scopeProto.GetRules().GetIncludedClusters())
	expectedNamespaces = []*storage.SimpleAccessScope_Rules_Namespace{
		{
			ClusterName:   "clusterA",
			NamespaceName: "NamespaceA",
		},
		{
			ClusterName:   "clusterA",
			NamespaceName: "NamespaceB",
		},
		{
			ClusterName:   "clusterB",
			NamespaceName: "NamespaceC",
		},
		{
			ClusterName:   "clusterB",
			NamespaceName: "NamespaceD",
		},
	}
	assert.ElementsMatch(t, expectedNamespaces, scopeProto.GetRules().GetIncludedNamespaces())
}

func compareLabelSelectors(t *testing.T, labelSelectors []declarativeconfig.LabelSelector, protoLabelSelectors []*storage.SetBasedLabelSelector) {
	for labelID, labelSelector := range labelSelectors {
		for reqID, req := range labelSelector.Requirements {
			protoReq := protoLabelSelectors[labelID].GetRequirements()[reqID]
			assert.Equal(t, req.Key, protoReq.GetKey())
			assert.Equal(t, storage.SetBasedLabelSelector_Operator(req.Operator), protoReq.GetOp())
			assert.Equal(t, req.Values, protoReq.GetValues())
		}
	}
}
