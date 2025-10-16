package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	storeMocks "github.com/stackrox/rox/central/policy/store/mocks"
	categoriesMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	policyCategoryMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicyDatastore(t *testing.T) {
	suite.Run(t, new(PolicyDatastoreTestSuite))
}

type PolicyDatastoreTestSuite struct {
	suite.Suite

	mockCtrl            *gomock.Controller
	store               *storeMocks.MockStore
	datastore           DataStore
	clusterDatastore    *clusterMocks.MockDataStore
	notifierDatastore   *notifierMocks.MockDataStore
	categoriesDatastore *policyCategoryMocks.MockDataStore

	hasReadWriteWorkflowAdministrationAccess context.Context

	hasReadWorkflowAdministrationAccess context.Context
}

func (s *PolicyDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.categoriesDatastore = categoriesMocks.NewMockDataStore(s.mockCtrl)

	s.datastore = newWithoutDefaults(s.store, s.clusterDatastore, s.notifierDatastore, s.categoriesDatastore)

	s.hasReadWriteWorkflowAdministrationAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration),
		))
	s.hasReadWorkflowAdministrationAccess = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration),
		))
}

func (s *PolicyDatastoreTestSuite) testImportSuccessResponse(expectedPolicy *storage.Policy, resp *v1.ImportPolicyResponse) {
	s.True(resp.GetSucceeded())
	protoassert.Equal(s.T(), expectedPolicy, resp.GetPolicy())
	s.Empty(resp.GetErrors())
}

func (s *PolicyDatastoreTestSuite) testImportFailResponse(expectedPolicy *storage.Policy, expectedErrTypes, expectedErrorStrings, expectedNames []string, resp *v1.ImportPolicyResponse) {
	s.False(resp.GetSucceeded())
	protoassert.Equal(s.T(), expectedPolicy, resp.GetPolicy())
	s.Require().Len(resp.GetErrors(), len(expectedErrTypes))
	s.Require().Len(resp.GetErrors(), len(expectedErrorStrings))
	s.Require().Len(resp.GetErrors(), len(expectedNames))
	for i, err := range resp.GetErrors() {
		s.Require().Equal(expectedErrTypes[i], err.GetType())
		s.Equal(expectedErrorStrings[i], err.GetMessage())
		s.Equal(expectedNames[i], err.GetDuplicateName())
	}
}

func (s *PolicyDatastoreTestSuite) TestImportPolicySucceeds() {
	policy := &storage.Policy{}
	policy.SetName("policy-to-import")
	policy.SetId("import-1")
	policy.SetSORTName("policy-to-import")
	policy.SetCategories([]string{"DevOps Best Practices"})

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy)).Return(nil)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId(), policy.GetCategories())

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportSuccessResponse(policy, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateID() {
	policy := &storage.Policy{}
	policy.SetName("test policy")
	policy.SetId("test-policy-1")
	policy.SetSORTName("test policy")

	errString := "policy with id \"test-policy-1\" already exists, unable to import policy"

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(policy, true, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Policy) error) error {
			return fn(policy)
		})
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, nil).Times(1)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateID, policies.ErrImportDuplicateName},
		[]string{errString, errString}, []string{policy.GetName(), policy.GetName()}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateName() {
	name := "test-duplicate-policy-name"
	policy := &storage.Policy{}
	policy.SetName(name)
	policy.SetId("duplicate-1")
	policy.SetSORTName(name)

	errString := fmt.Sprintf("policy with id %q already exists, unable to import policy", policy.GetId())

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).AnyTimes().Return(nil, nil)

	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Policy) error) error {
			policy2 := &storage.Policy{}
			policy2.SetName(name)
			policy2.SetId("some-other-id")
			policy2.SetSORTName(name)
			return fn(policy2)
		})
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{name}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyMixedSuccessAndFailure() {
	succeedName := "success"
	policySucceed := &storage.Policy{}
	policySucceed.SetName(succeedName)
	policySucceed.SetId("Succeed ID")
	policySucceed.SetSORTName(succeedName)
	policySucceed.SetCategories([]string{})
	fail1Name := "fail 1 name"
	policyFail1 := &storage.Policy{}
	policyFail1.SetName(fail1Name)
	policyFail1.SetId("Fail 1 ID")
	policyFail1.SetSORTName(fail1Name)
	policyFail1.SetCategories([]string{})
	policyFail2 := &storage.Policy{}
	policyFail2.SetName("import failure name")
	policyFail2.SetId("Fail 2 ID")
	policyFail2.SetSORTName("import failure name")
	policyFail2.SetCategories([]string{})

	errString := "some error string"
	errorFail1 := &PolicyStoreErrorList{
		Errors: []error{
			&NameConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail1Name,
			},
		},
	}
	fail2Name := "fail 2 name"
	errorFail2 := &PolicyStoreErrorList{
		Errors: []error{
			&IDConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail2Name,
			},
		},
	}

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)

	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)

	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policySucceed).Return(nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil, false, nil).AnyTimes()

	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policyFail1).Return(errorFail1)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policyFail2).Return(errorFail2)

	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policySucceed.GetId(), policySucceed.GetCategories())

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policySucceed.CloneVT(), policyFail1.CloneVT(), policyFail2.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 3)

	s.testImportSuccessResponse(policySucceed, responses[0])

	s.testImportFailResponse(policyFail1, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{fail1Name}, responses[1])

	s.testImportFailResponse(policyFail2, []string{policies.ErrImportDuplicateID}, []string{errString}, []string{fail2Name}, responses[2])
}

func (s *PolicyDatastoreTestSuite) TestUnknownError() {
	name := "unknown-error"
	policy := &storage.Policy{}
	policy.SetName(name)
	policy.SetId("unknown-error-id")
	policy.SetSORTName(name)
	policy.SetCategories([]string{})

	errString := "this is not a structured error type"
	storeError := errors.New(errString)

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportUnknown}, []string{errString}, []string{""}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportOverwrite() {
	id1 := "ID1"
	policy1 := &storage.Policy{}
	policy1.SetName("new name 1")
	policy1.SetId(id1)
	policy1.SetSORTName("new name 1")
	policy2Name := "a very good name"
	policy2 := &storage.Policy{}
	policy2.SetName(policy2Name)
	policy2.SetId("new ID 2")
	policy2.SetSORTName(policy2Name)
	// Same ID as policy1, unique name
	existingPolicy1 := &storage.Policy{}
	existingPolicy1.SetName("existing name 1")
	existingPolicy1.SetId(id1)
	// Unique ID, same name as policy 2
	existingPolicy2 := &storage.Policy{}
	existingPolicy2.SetName(policy2Name)
	existingPolicy2.SetId("existing ID 2")

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)

	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Policy) error) error {
			for _, p := range []*storage.Policy{existingPolicy1, existingPolicy2} {
				if err := fn(p); err != nil {
					return err
				}
			}
			return nil
		})
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, id1).Return(existingPolicy1, true, nil).Times(2)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy2.GetId()).Return(existingPolicy1, true, nil).Times(2)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, id1).Return(nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, policy2.GetId()).Return(nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy2.GetId()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy1)).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy2)).Return(nil)

	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.GetId()).Return(nil, nil).Times(1)
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy2.GetId()).Return(nil, nil).Times(1)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy2.GetId(), nil).Return(nil).Times(1)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.GetId(), nil).Return(nil).Times(1)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy1.CloneVT(), policy2.CloneVT()}, true)

	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 2)

	s.testImportSuccessResponse(policy1, responses[0])
	s.testImportSuccessResponse(policy2, responses[1])
}

func (s *PolicyDatastoreTestSuite) TestRemoveScopesAndNotifiers() {
	clusterName := "test"
	notifierName := "test"
	policy := storage.Policy_builder{
		Name:     "Boo's policy",
		Id:       "policy-boo",
		SORTName: "Boo's policy",
		Scope: []*storage.Scope{
			storage.Scope_builder{
				Cluster: clusterName,
			}.Build(),
		},
		Exclusions: []*storage.Exclusion{
			storage.Exclusion_builder{
				Deployment: storage.Exclusion_Deployment_builder{
					Scope: storage.Scope_builder{
						Cluster: clusterName,
					}.Build(),
				}.Build(),
			}.Build(),
		},
		Notifiers: []string{notifierName},
	}.Build()

	resultPolicy := &storage.Policy{}
	resultPolicy.SetName("Boo's policy")
	resultPolicy.SetId("policy-boo")
	resultPolicy.SetSORTName("Boo's policy")

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.hasReadWriteWorkflowAdministrationAccess, notifierName).Return(nil, false, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId(), policy.GetCategories())

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	protoassert.Equal(s.T(), resultPolicy, resp.GetPolicy())
	s.Require().Len(resp.GetErrors(), 1)
	importError := resp.GetErrors()[0]
	s.Equal(importError.GetType(), policies.ErrImportClustersOrNotifiersRemoved)
	s.Equal(importError.GetMessage(), "Cluster scopes, cluster exclusions, and notification options have been removed from this policy.")
}

func (s *PolicyDatastoreTestSuite) TestDoesNotRemoveScopesAndNotifiers() {
	clusterID := "test"
	notifierName := "test"
	policy := storage.Policy_builder{
		Name:     "Some Name",
		Id:       "Some ID",
		SORTName: "Some Name",
		Scope: []*storage.Scope{
			storage.Scope_builder{
				Cluster: clusterID,
			}.Build(),
		},
		Exclusions: []*storage.Exclusion{
			storage.Exclusion_builder{
				Deployment: storage.Exclusion_Deployment_builder{
					Scope: storage.Scope_builder{
						Cluster: clusterID,
					}.Build(),
				}.Build(),
			}.Build(),
		},
		Notifiers:  []string{notifierName},
		Categories: []string{"DevOps Best Practices"},
	}.Build()

	cluster := &storage.Cluster{}
	cluster.SetId(clusterID)
	mockClusters := []*storage.Cluster{
		cluster,
	}
	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(mockClusters, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.hasReadWriteWorkflowAdministrationAccess, notifierName).Return(nil, true, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy)).Return(nil)

	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId(), policy.GetCategories())

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	protoassert.Equal(s.T(), resp.GetPolicy(), policy)
	s.Empty(resp.GetErrors())
}

func eq(expected *storage.Policy) gomock.Matcher {
	return gomock.Cond(func(actual *storage.Policy) bool {
		e := expected.CloneVT()
		e.SetCategories(nil)
		return e.EqualVT(actual)
	})
}
