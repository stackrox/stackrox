package resolvers

import (
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	cveMocks "github.com/stackrox/rox/central/cve/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	componentMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	npsMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	k8sroleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	k8srolebindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	search2 "github.com/stackrox/rox/central/search"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestSearchCategories(t *testing.T) {
	ctrl := gomock.NewController(t)
	cluster := clusterMocks.NewMockDataStore(ctrl)
	deployment := deploymentMocks.NewMockDataStore(ctrl)
	namespace := namespaceMocks.NewMockDataStore(ctrl)
	secret := secretMocks.NewMockDataStore(ctrl)
	nps := npsMocks.NewMockDataStore(ctrl)
	violations := alertMocks.NewMockDataStore(ctrl)
	images := imageMocks.NewMockDataStore(ctrl)
	policies := policyMocks.NewMockDataStore(ctrl)
	nodes := nodeMocks.NewMockGlobalDataStore(ctrl)
	serviceAccounts := serviceAccountMocks.NewMockDataStore(ctrl)
	roles := k8sroleMocks.NewMockDataStore(ctrl)
	rolebindings := k8srolebindingMocks.NewMockDataStore(ctrl)
	cves := cveMocks.NewMockDataStore(ctrl)
	components := componentMocks.NewMockDataStore(ctrl)

	resolver := &Resolver{
		ClusterDataStore:         cluster,
		DeploymentDataStore:      deployment,
		PolicyDataStore:          policies,
		NamespaceDataStore:       namespace,
		SecretsDataStore:         secret,
		NetworkPoliciesStore:     nps,
		ViolationsDataStore:      violations,
		ImageDataStore:           images,
		ServiceAccountsDataStore: serviceAccounts,
		NodeGlobalDataStore:      nodes,
		K8sRoleBindingStore:      rolebindings,
		K8sRoleStore:             roles,
		ImageComponentDataStore:  components,
		CVEDataStore:             cves,
	}

	searchCategories := resolver.getAutoCompleteSearchers()
	searchFuncs := resolver.getSearchFuncs()

	for globalCategory := range search2.GetGlobalSearchCategories() {
		assert.True(t, searchCategories[globalCategory] != nil, "global search category %s does not exist in auto complete", globalCategory)
	}
	for category := range searchCategories {
		switch category {
		case v1.SearchCategory_COMPLIANCE:
			continue
		default:
			assert.True(t, searchFuncs[category] != nil, "search category %s does not have a search func", category.String())
		}
	}
}
