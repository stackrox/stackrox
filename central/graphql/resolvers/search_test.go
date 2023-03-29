//go:build sql_integration

package resolvers

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	clusterCVEMocks "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	cveMocks "github.com/stackrox/rox/central/cve/datastore/mocks"
	imageCVEMocks "github.com/stackrox/rox/central/cve/image/datastore/mocks"
	nodeCVEMocks "github.com/stackrox/rox/central/cve/node/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageComponentMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	npsMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	nodeComponentMocks "github.com/stackrox/rox/central/nodecomponent/datastore/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	policyCategoryMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	k8sroleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	k8sRoleBindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	k8srolebindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	globalSearch "github.com/stackrox/rox/central/search"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	nodes := nodeMocks.NewMockDataStore(ctrl)
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
		NodeDataStore:            nodes,
		K8sRoleBindingStore:      rolebindings,
		K8sRoleStore:             roles,
		ImageComponentDataStore:  components,
	}

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		resolver.CVEDataStore = cves
	} else {
		resolver.PolicyCategoryDataStore = policyCategoryMocks.NewMockDataStore(ctrl)
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

func TestSubjectAutocompleteSearch(t *testing.T) {
	t.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	testDB := pgtest.ForT(t)
	testGormDB := testDB.GetGormDB(t)
	defer pgtest.CloseGormDB(t, testGormDB)
	defer testDB.Teardown(t)

	roleBindingDatastore, err := k8sRoleBindingDataStore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	ctx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	roleBindings := fixtures.GetMultipleK8sRoleBindings(2, 3)
	for _, roleBinding := range roleBindings {
		err = roleBindingDatastore.UpsertRoleBinding(ctx, roleBinding)
		require.NoError(t, err)
	}

	resolver, _ := SetupTestResolver(t, roleBindingDatastore)
	allowAllCtx := SetAuthorizerOverride(ctx, allow.Anonymous())

	testCases := []struct {
		desc     string
		request  searchRequest
		expected []string
	}{
		{
			desc: "Subject name autocomplete",
			request: searchRequest{
				Query:      fmt.Sprintf("Subject:%s", roleBindings[0].Subjects[1].Name),
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{roleBindings[0].Subjects[1].Name},
		},
		{
			desc: "Subject Kind autocomplete",
			request: searchRequest{
				Query:      "Subject Kind:",
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{"user", "group"},
		},
		{
			desc: "Cluster name autocomplete",
			request: searchRequest{
				Query:      fmt.Sprintf("Cluster:%s", roleBindings[1].ClusterName),
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{roleBindings[1].ClusterName},
		},
		{
			desc: "Cluster role autocomplete",
			request: searchRequest{
				Query:      "Cluster Role:tr",
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{"true"},
		},
		{
			desc: "Cluster name + Subject name autocomplete",
			request: searchRequest{
				Query:      fmt.Sprintf("Cluster:%s+Subject:", roleBindings[0].ClusterName),
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{roleBindings[0].Subjects[1].Name, roleBindings[0].Subjects[2].Name},
		},
		{
			desc: "Autocomplete on unsupported option",
			request: searchRequest{
				Query:      "Deployment:d1",
				Categories: &[]string{"SUBJECTS"},
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			results, err := resolver.SearchAutocomplete(allowAllCtx, tc.request)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.expected, results)
		})
	}
}
