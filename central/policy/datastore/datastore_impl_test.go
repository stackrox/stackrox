package datastore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/stackrox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/stackrox/central/notifier/datastore/mocks"
	indexMocks "github.com/stackrox/stackrox/central/policy/index/mocks"
	"github.com/stackrox/stackrox/central/policy/store/boltdb"
	storeMocks "github.com/stackrox/stackrox/central/policy/store/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/policies"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestPolicyDatastore(t *testing.T) {
	suite.Run(t, new(PolicyDatastoreTestSuite))
}

type PolicyDatastoreTestSuite struct {
	suite.Suite

	mockCtrl          *gomock.Controller
	store             *storeMocks.MockStore
	indexer           *indexMocks.MockIndexer
	datastore         DataStore
	clusterDatastore  *clusterMocks.MockDataStore
	notifierDatastore *notifierMocks.MockDataStore

	ctx context.Context
}

func (s *PolicyDatastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierMocks.NewMockDataStore(s.mockCtrl)

	s.datastore = newWithoutDefaults(s.store, s.indexer, nil, s.clusterDatastore, s.notifierDatastore)

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *PolicyDatastoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
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

func (s *PolicyDatastoreTestSuite) TestImportPolicySucceeds() {
	policy := &storage.Policy{
		Name:     "policy-to-import",
		Id:       "import-1",
		SORTName: "policy-to-import",
	}

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().Get(s.ctx, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().GetAll(s.ctx).Return(nil, nil)
	s.store.EXPECT().Upsert(s.ctx, policy).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy.Clone()}, false)
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

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().Get(s.ctx, policy.GetId()).Return(policy, true, nil)
	s.store.EXPECT().GetAll(s.ctx).Return([]*storage.Policy{
		policy,
	}, nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy.Clone()}, false)
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

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().Get(s.ctx, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().GetAll(s.ctx).Return([]*storage.Policy{
		{
			Name:     name,
			Id:       "some-other-id",
			SORTName: name,
		},
	}, nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy.Clone()}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{name}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyMixedSuccessAndFailure() {
	succeedName := "success"
	policySucceed := &storage.Policy{
		Name:     succeedName,
		Id:       "Succeed ID",
		SORTName: succeedName,
	}
	fail1Name := "fail 1 name"
	policyFail1 := &storage.Policy{
		Name:     fail1Name,
		Id:       "Fail 1 ID",
		SORTName: fail1Name,
	}
	policyFail2 := &storage.Policy{
		Name:     "import failure name",
		Id:       "Fail 2 ID",
		SORTName: "import failure name",
	}

	errString := "some error string"
	errorFail1 := &boltdb.PolicyStoreErrorList{
		Errors: []error{
			&boltdb.NameConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail1Name,
			},
		},
	}
	fail2Name := "fail 2 name"
	errorFail2 := &boltdb.PolicyStoreErrorList{
		Errors: []error{
			&boltdb.IDConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail2Name,
			},
		},
	}

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)

	s.store.EXPECT().GetAll(s.ctx).Return(nil, nil)

	s.store.EXPECT().Upsert(s.ctx, policySucceed).Return(nil)
	s.indexer.EXPECT().AddPolicy(policySucceed).Return(nil)
	s.store.EXPECT().Get(s.ctx, gomock.Any()).Return(nil, false, nil).AnyTimes()

	s.store.EXPECT().Upsert(s.ctx, policyFail1).Return(errorFail1)
	s.store.EXPECT().Upsert(s.ctx, policyFail2).Return(errorFail2)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policySucceed.Clone(), policyFail1.Clone(), policyFail2.Clone()}, false)
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
		Name:     name,
		Id:       "unknown-error-id",
		SORTName: name,
	}

	errString := "this is not a structured error type"
	storeError := errors.New(errString)

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().Get(s.ctx, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().GetAll(s.ctx).Return(nil, nil)
	s.store.EXPECT().Upsert(s.ctx, policy).Return(storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy.Clone()}, false)
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

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)

	s.store.EXPECT().GetAll(s.ctx).Return([]*storage.Policy{existingPolicy1, existingPolicy2}, nil)

	s.store.EXPECT().Get(s.ctx, existingPolicy1.GetId()).Return(nil, true, nil)
	s.store.EXPECT().Delete(s.ctx, existingPolicy1.GetId()).Return(nil)
	s.indexer.EXPECT().DeletePolicy(existingPolicy1.GetId()).Return(nil)
	s.store.EXPECT().Upsert(s.ctx, policy1).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy1).Return(nil)

	s.store.EXPECT().Get(s.ctx, policy2.GetId()).Return(nil, false, nil)
	s.store.EXPECT().Delete(s.ctx, existingPolicy2.GetId()).Return(nil)
	s.indexer.EXPECT().DeletePolicy(existingPolicy2.GetId()).Return(nil)
	s.store.EXPECT().Upsert(s.ctx, policy2).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy2).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy1.Clone(), policy2.Clone()}, true)
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

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.ctx, notifierName).Return(nil, false, nil)
	s.store.EXPECT().Get(s.ctx, policy.GetId()).Return(nil, false, nil)
	s.store.EXPECT().GetAll(s.ctx).Return(nil, nil)
	s.store.EXPECT().Upsert(s.ctx, policy).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
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
		Notifiers: []string{notifierName},
	}

	mockClusters := []*storage.Cluster{
		{
			Id: clusterID,
		},
	}
	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(mockClusters, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.ctx, notifierName).Return(nil, true, nil)
	s.store.EXPECT().GetAll(s.ctx).Return(nil, nil)
	s.store.EXPECT().Get(s.ctx, policy.Id).Return(nil, false, nil)
	s.store.EXPECT().Upsert(s.ctx, policy).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy.Clone()}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	s.Equal(resp.GetPolicy(), policy)
	s.Empty(resp.GetErrors())
}
