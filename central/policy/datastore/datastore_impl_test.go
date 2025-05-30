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

	s.datastore = newWithoutDefaults(s.store, nil, s.clusterDatastore, s.notifierDatastore, s.categoriesDatastore)

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
	s.True(resp.Succeeded)
	protoassert.Equal(s.T(), expectedPolicy, resp.GetPolicy())
	s.Empty(resp.Errors)
}

func (s *PolicyDatastoreTestSuite) testImportFailResponse(expectedPolicy *storage.Policy, expectedErrTypes, expectedErrorStrings, expectedNames []string, resp *v1.ImportPolicyResponse) {
	s.False(resp.Succeeded)
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
	policy := &storage.Policy{
		Name:       "policy-to-import",
		Id:         "import-1",
		SORTName:   "policy-to-import",
		Categories: []string{"DevOps Best Practices"},
	}

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy)).Return(nil)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.Id, policy.Categories)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportSuccessResponse(policy, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateID() {
	policy := &storage.Policy{
		Name:     "test policy",
		Id:       "test-policy-1",
		SORTName: "test policy",
	}

	errString := "policy with id \"test-policy-1\" already exists, unable to import policy"

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(policy, true, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Policy) error) error {
			return fn(policy)
		})
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.Id).Return(nil, nil).Times(1)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateID, policies.ErrImportDuplicateName},
		[]string{errString, errString}, []string{policy.GetName(), policy.GetName()}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateName() {
	name := "test-duplicate-policy-name"
	policy := &storage.Policy{
		Name:     name,
		Id:       "duplicate-1",
		SORTName: name,
	}

	errString := fmt.Sprintf("policy with id %q already exists, unable to import policy", policy.GetId())

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).AnyTimes().Return(nil, nil)

	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).DoAndReturn(
		func(_ context.Context, fn func(obj *storage.Policy) error) error {
			return fn(&storage.Policy{
				Name:     name,
				Id:       "some-other-id",
				SORTName: name,
			})
		})
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.CloneVT()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{name}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyMixedSuccessAndFailure() {
	succeedName := "success"
	policySucceed := &storage.Policy{
		Name:       succeedName,
		Id:         "Succeed ID",
		SORTName:   succeedName,
		Categories: []string{},
	}
	fail1Name := "fail 1 name"
	policyFail1 := &storage.Policy{
		Name:       fail1Name,
		Id:         "Fail 1 ID",
		SORTName:   fail1Name,
		Categories: []string{},
	}
	policyFail2 := &storage.Policy{
		Name:       "import failure name",
		Id:         "Fail 2 ID",
		SORTName:   "import failure name",
		Categories: []string{},
	}

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

	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policySucceed.Id, policySucceed.Categories)

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
	policy := &storage.Policy{
		Name:       name,
		Id:         "unknown-error-id",
		SORTName:   name,
		Categories: []string{},
	}

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
	policy1 := &storage.Policy{
		Name:     "new name 1",
		Id:       id1,
		SORTName: "new name 1",
	}
	policy2Name := "a very good name"
	policy2 := &storage.Policy{
		Name:     policy2Name,
		Id:       "new ID 2",
		SORTName: policy2Name,
	}
	// Same ID as policy1, unique name
	existingPolicy1 := &storage.Policy{
		Name: "existing name 1",
		Id:   id1,
	}
	// Unique ID, same name as policy 2
	existingPolicy2 := &storage.Policy{
		Name: policy2Name,
		Id:   "existing ID 2",
	}

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
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy2.Id).Return(existingPolicy1, true, nil).Times(2)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, id1).Return(nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, policy2.Id).Return(nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy2.Id).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy1)).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy2)).Return(nil)

	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.Id).Return(nil, nil).Times(1)
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy2.Id).Return(nil, nil).Times(1)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy2.Id, nil).Return(nil).Times(1)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.Id, nil).Return(nil).Times(1)

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
	policy := &storage.Policy{
		Name:     "Boo's policy",
		Id:       "policy-boo",
		SORTName: "Boo's policy",
		Scope: []*storage.Scope{
			{
				Cluster: clusterName,
			},
		},
		Exclusions: []*storage.Exclusion{
			{
				Deployment: &storage.Exclusion_Deployment{
					Scope: &storage.Scope{
						Cluster: clusterName,
					},
				},
			},
		},
		Notifiers: []string{notifierName},
	}

	resultPolicy := &storage.Policy{
		Name:     "Boo's policy",
		Id:       "policy-boo",
		SORTName: "Boo's policy",
	}

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.hasReadWriteWorkflowAdministrationAccess, notifierName).Return(nil, false, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil)
	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.Id, policy.Categories)

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
	policy := &storage.Policy{
		Name:     "Some Name",
		Id:       "Some ID",
		SORTName: "Some Name",
		Scope: []*storage.Scope{
			{
				Cluster: clusterID,
			},
		},
		Exclusions: []*storage.Exclusion{
			{
				Deployment: &storage.Exclusion_Deployment{
					Scope: &storage.Scope{
						Cluster: clusterID,
					},
				},
			},
		},
		Notifiers:  []string{notifierName},
		Categories: []string{"DevOps Best Practices"},
	}

	mockClusters := []*storage.Cluster{
		{
			Id: clusterID,
		},
	}
	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(mockClusters, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.hasReadWriteWorkflowAdministrationAccess, notifierName).Return(nil, true, nil)
	s.store.EXPECT().Walk(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.Id).Return(nil, false, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, eq(policy)).Return(nil)

	s.categoriesDatastore.EXPECT().SetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.Id, policy.Categories)

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
		e.Categories = nil
		return e.EqualVT(actual)
	})
}
