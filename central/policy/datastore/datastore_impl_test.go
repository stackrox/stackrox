package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	indexMocks "github.com/stackrox/rox/central/policy/index/mocks"
	storeMocks "github.com/stackrox/rox/central/policy/store/mocks"
	categoriesMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	policyCategoryMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
	indexer             *indexMocks.MockIndexer
	datastore           DataStore
	clusterDatastore    *clusterMocks.MockDataStore
	notifierDatastore   *notifierMocks.MockDataStore
	categoriesDatastore *policyCategoryMocks.MockDataStore

	hasReadWriteWorkflowAdministrationAccess context.Context

	hasReadWorkflowAdministrationAccess context.Context
}

func (s *PolicyDatastoreTestSuite) SetupTest() {
	pgtest.SkipIfPostgresEnabled(s.T())
	s.mockCtrl = gomock.NewController(s.T())
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
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
	s.Equal(expectedPolicy, resp.GetPolicy())
	s.Empty(resp.Errors)
}

func (s *PolicyDatastoreTestSuite) testImportFailResponse(expectedPolicy *storage.Policy, expectedErrTypes, expectedErrorStrings, expectedNames []string, resp *v1.ImportPolicyResponse) {
	s.False(resp.Succeeded)
	s.Equal(expectedPolicy, resp.GetPolicy())
	s.Require().Len(resp.GetErrors(), len(expectedErrTypes))
	s.Require().Len(resp.GetErrors(), len(expectedErrorStrings))
	s.Require().Len(resp.GetErrors(), len(expectedNames))
	for i, err := range resp.GetErrors() {
		s.Require().Equal(expectedErrTypes[i], err.GetType())
		s.Equal(expectedErrorStrings[i], err.GetMessage())
		s.Equal(expectedNames[i], err.GetDuplicateName())
	}
}

// TODO: ROX-13888 Remove test.
func (s *PolicyDatastoreTestSuite) TestReplacingResourceAccess() {
	policy := &storage.Policy{
		Name: "policy-to-import",
		Id:   "import-1",
	}

	// Should work with READ access to WorkflowAdministration.
	s.store.EXPECT().Get(s.hasReadWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil).Times(1)
	_, _, err := s.datastore.GetPolicy(s.hasReadWorkflowAdministrationAccess, policy.GetId())
	s.NoError(err)

	// Shouldn't work with READ access to WorkflowAdministration.
	_, err = s.datastore.AddPolicy(s.hasReadWorkflowAdministrationAccess, policy)
	s.Error(err)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	err = s.datastore.RemovePolicy(s.hasReadWorkflowAdministrationAccess, policy.GetId())
	s.Error(err)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)

	// Should work with READ/WRITE access to WorkflowAdministration.
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil).Times(1)
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil).Times(1)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil).Times(1)

	_, err = s.datastore.AddPolicy(s.hasReadWriteWorkflowAdministrationAccess, policy)
	s.NoError(err)
	err = s.datastore.RemovePolicy(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId())
	s.NoError(err)
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
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.Clone()}, false)
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

	errString1 := "policy with id '\"test-policy-1\"' already exists, unable to import policy"
	errString2 := "policy with name 'test policy' already exists, unable to import policy"

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(policy, true, nil)
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return([]*storage.Policy{
		policy,
	}, nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.Clone()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateID, policies.ErrImportDuplicateName},
		[]string{errString1, errString2}, []string{policy.GetName(), policy.GetName()}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateName() {
	name := "test-duplicate-policy-name"
	policy := &storage.Policy{
		Name:     name,
		Id:       "duplicate-1",
		SORTName: name,
	}

	errString := fmt.Sprintf("policy with name '%s' already exists, unable to import policy", name)

	s.clusterDatastore.EXPECT().GetClusters(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.GetId()).Return(nil, false, nil)
	s.categoriesDatastore.EXPECT().GetPolicyCategoriesForPolicy(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).AnyTimes().Return(nil, nil)

	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return([]*storage.Policy{
		{
			Name:     name,
			Id:       "some-other-id",
			SORTName: name,
		},
	}, nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.Clone()}, false)
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

	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)

	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policySucceed).Return(nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, gomock.Any()).Return(nil, false, nil).AnyTimes()

	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policyFail1).Return(errorFail1)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policyFail2).Return(errorFail2)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policySucceed.Clone(), policyFail1.Clone(), policyFail2.Clone()}, false)
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
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.Clone()}, false)
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

	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return([]*storage.Policy{existingPolicy1, existingPolicy2}, nil)

	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.GetId()).Return(nil, true, nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy1.GetId()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy1).Return(nil)

	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy2.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Delete(s.hasReadWriteWorkflowAdministrationAccess, existingPolicy2.GetId()).Return(nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy2).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy1.Clone(), policy2.Clone()}, true)

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
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	s.Equal(resultPolicy, resp.GetPolicy())
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
	s.store.EXPECT().GetAll(s.hasReadWriteWorkflowAdministrationAccess).Return(nil, nil)
	s.store.EXPECT().Get(s.hasReadWriteWorkflowAdministrationAccess, policy.Id).Return(nil, false, nil)
	s.store.EXPECT().Upsert(s.hasReadWriteWorkflowAdministrationAccess, policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.hasReadWriteWorkflowAdministrationAccess, []*storage.Policy{policy.Clone()}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	s.Equal(resp.GetPolicy(), policy)
	s.Empty(resp.GetErrors())
}
