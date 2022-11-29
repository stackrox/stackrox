package resolvers

import (
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	clusterCVEMocks "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	cveMocks "github.com/stackrox/rox/central/cve/datastore/mocks"
	imageCVEMocks "github.com/stackrox/rox/central/cve/image/datastore/mocks"
	nodeCVEMocks "github.com/stackrox/rox/central/cve/node/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageComponentMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	npsMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/globaldatastore/mocks"
	nodeComponentMocks "github.com/stackrox/rox/central/nodecomponent/datastore/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	k8sroleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	k8srolebindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	globalSearch "github.com/stackrox/rox/central/search"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
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
	components := imageComponentMocks.NewMockDataStore(ctrl)

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
	}

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		resolver.CVEDataStore = cves
	} else {
		resolver.ImageCVEDataStore = imageCVEMocks.NewMockDataStore(ctrl)
		resolver.NodeCVEDataStore = nodeCVEMocks.NewMockDataStore(ctrl)
		resolver.ClusterCVEDataStore = clusterCVEMocks.NewMockDataStore(ctrl)
		resolver.NodeComponentDataStore = nodeComponentMocks.NewMockDataStore(ctrl)
	}

	searchCategories := resolver.getAutoCompleteSearchers()
	searchFuncs := resolver.getSearchFuncs()

	for globalCategory := range globalSearch.GetGlobalSearchCategories() {
		if globalCategory == v1.SearchCategory_IMAGE_INTEGRATIONS {
			continue
		}
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
