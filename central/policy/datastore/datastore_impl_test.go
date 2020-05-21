package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	indexMocks "github.com/stackrox/rox/central/policy/index/mocks"
	"github.com/stackrox/rox/central/policy/store"
	storeMocks "github.com/stackrox/rox/central/policy/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/sac"
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

func (s *PolicyDatastoreTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())

	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.indexer = indexMocks.NewMockIndexer(s.mockCtrl)
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.datastore = New(s.store, s.indexer, nil, s.clusterDatastore, s.notifierDatastore)

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *PolicyDatastoreTestSuite) TearDownSuite() {
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
		Name: "Some Name",
		Id:   "Some ID",
	}

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetId(), nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportSuccessResponse(policy, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateID() {
	policy := &storage.Policy{
		Name: "Some Name",
		Id:   "Some ID",
	}

	otherName := "Joseph Rules"
	errString := "some error string"
	storeError := &store.PolicyStoreErrorList{
		Errors: []error{
			&store.IDConflictError{
				ErrString:          errString,
				ExistingPolicyName: otherName,
			},
		},
	}
	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetId(), storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateID}, []string{errString}, []string{otherName}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateName() {
	name := "Some Name"
	policy := &storage.Policy{
		Name: name,
		Id:   "Some ID",
	}

	errString := "some error string"
	storeError := &store.PolicyStoreErrorList{
		Errors: []error{
			&store.NameConflictError{
				ErrString:          errString,
				ExistingPolicyName: name,
			},
		},
	}
	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetId(), storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{name}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyDuplicateNameAndDuplicateID() {
	name := "Some Name"
	policy := &storage.Policy{
		Name: name,
		Id:   "Some ID",
	}

	errString := "some error string"
	storeError := &store.PolicyStoreErrorList{
		Errors: []error{
			&store.NameConflictError{
				ErrString:          errString,
				ExistingPolicyName: name,
			},
			&store.IDConflictError{
				ErrString:          errString,
				ExistingPolicyName: name,
			},
		},
	}
	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetId(), storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportDuplicateName, policies.ErrImportDuplicateID}, []string{errString, errString}, []string{name, name}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportPolicyMixedSuccessAndFailure() {
	succeedName := "Some Name"
	policySucceed := &storage.Policy{
		Name: succeedName,
		Id:   "Succeed ID",
	}
	fail1Name := "fail 1 name"
	policyFail1 := &storage.Policy{
		Name: fail1Name,
		Id:   "Fail 1 ID",
	}
	policyFail2 := &storage.Policy{
		Name: "import failure name",
		Id:   "Fail 2 ID",
	}

	errString := "some error string"
	errorFail1 := &store.PolicyStoreErrorList{
		Errors: []error{
			&store.NameConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail1Name,
			},
		},
	}
	fail2Name := "fail 2 name"
	errorFail2 := &store.PolicyStoreErrorList{
		Errors: []error{
			&store.IDConflictError{
				ErrString:          errString,
				ExistingPolicyName: fail2Name,
			},
		},
	}

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)

	s.store.EXPECT().GetAllPolicies().Return(nil, nil)

	s.store.EXPECT().AddPolicy(policySucceed).Return(policySucceed.GetId(), nil)
	s.indexer.EXPECT().AddPolicy(policySucceed).Return(nil)

	s.store.EXPECT().AddPolicy(policyFail1).Return(policyFail1.GetId(), errorFail1)

	s.store.EXPECT().AddPolicy(policyFail2).Return(policyFail2.GetId(), errorFail2)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policySucceed, policyFail1, policyFail2}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 3)

	s.testImportSuccessResponse(policySucceed, responses[0])

	s.testImportFailResponse(policyFail1, []string{policies.ErrImportDuplicateName}, []string{errString}, []string{fail1Name}, responses[1])

	s.testImportFailResponse(policyFail2, []string{policies.ErrImportDuplicateID}, []string{errString}, []string{fail2Name}, responses[2])
}

func (s *PolicyDatastoreTestSuite) TestUnknownError() {
	name := "Some Name"
	policy := &storage.Policy{
		Name: name,
		Id:   "Some ID",
	}

	errString := "This is not a structured error type"
	storeError := errors.New(errString)

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetId(), storeError)
	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.False(allSucceeded)
	s.Require().Len(responses, 1)

	s.testImportFailResponse(policy, []string{policies.ErrImportUnknown}, []string{errString}, []string{""}, responses[0])
}

func (s *PolicyDatastoreTestSuite) TestImportOverwrite() {
	id1 := "ID1"
	policy1 := &storage.Policy{
		Name: "new name 1",
		Id:   id1,
	}
	policy2Name := "a very good name"
	policy2 := &storage.Policy{
		Name: policy2Name,
		Id:   "new ID 2",
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

	s.store.EXPECT().GetAllPolicies().Return([]*storage.Policy{existingPolicy1, existingPolicy2}, nil)

	s.store.EXPECT().UpdatePolicy(policy1).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy1).Return(nil)

	s.store.EXPECT().RemovePolicy(existingPolicy2.GetId()).Return(nil)
	s.indexer.EXPECT().DeletePolicy(existingPolicy2.GetId()).Return(nil)
	s.store.EXPECT().UpdatePolicy(policy2).Return(nil)
	s.indexer.EXPECT().AddPolicy(policy2).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy1, policy2}, true)
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
		Name: "Some Name",
		Id:   "Some ID",
		Scope: []*storage.Scope{
			{
				Cluster: clusterName,
			},
		},
		Whitelists: []*storage.Whitelist{
			{
				Deployment: &storage.Whitelist_Deployment{
					Scope: &storage.Scope{
						Cluster: clusterName,
					},
				},
			},
		},
		Notifiers: []string{notifierName},
	}

	s.clusterDatastore.EXPECT().GetClusters(s.ctx).Return(nil, nil)
	s.notifierDatastore.EXPECT().GetNotifier(s.ctx, notifierName).Return(nil, false, nil)
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetName(), nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resultPolicy := &storage.Policy{
		Name: "Some Name",
		Id:   "Some ID",
	}
	resp := responses[0]
	s.True(resp.GetSucceeded())
	s.Equal(resp.GetPolicy(), resultPolicy)
	s.Require().Len(resp.GetErrors(), 1)
	importError := resp.GetErrors()[0]
	s.Equal(importError.GetType(), policies.ErrImportClustersOrNotifiersRemoved)
	s.Equal(importError.GetMessage(), "Cluster scopes, cluster whitelists, and notification options have been removed from this policy.")
}

func (s *PolicyDatastoreTestSuite) TestDoesNotRemoveScopesAndNotifiers() {
	clusterID := "test"
	notifierName := "test"
	policy := &storage.Policy{
		Name: "Some Name",
		Id:   "Some ID",
		Scope: []*storage.Scope{
			{
				Cluster: clusterID,
			},
		},
		Whitelists: []*storage.Whitelist{
			{
				Deployment: &storage.Whitelist_Deployment{
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
	s.store.EXPECT().GetAllPolicies().Return(nil, nil)
	s.store.EXPECT().AddPolicy(policy).Return(policy.GetName(), nil)
	s.indexer.EXPECT().AddPolicy(policy).Return(nil)

	responses, allSucceeded, err := s.datastore.ImportPolicies(s.ctx, []*storage.Policy{policy}, false)
	s.NoError(err)
	s.True(allSucceeded)
	s.Require().Len(responses, 1)

	resp := responses[0]
	s.True(resp.GetSucceeded())
	s.Equal(resp.GetPolicy(), policy)
	s.Empty(resp.GetErrors())
}
