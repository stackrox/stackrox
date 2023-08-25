package detection

import (
	"context"
	"testing"

	mocks2 "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type compiledPolicyWrapper struct {
	detection.CompiledPolicy

	policy *storage.Policy
}

func wrapPolicy(policy *storage.Policy) compiledPolicyWrapper {
	return compiledPolicyWrapper{
		policy: policy,
	}
}

func (w compiledPolicyWrapper) Policy() *storage.Policy {
	return w.policy
}

func TestPolicySet_RemoveNotifier(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	policySetMock := mocks.NewMockPolicySet(mockCtrl)
	policyDatastoreMock := mocks2.NewMockDataStore(mockCtrl)

	policySet := &setImpl{
		PolicySet:   policySetMock,
		policyStore: policyDatastoreMock,
	}

	policySetMock.EXPECT().GetCompiledPolicies().Return(map[string]detection.CompiledPolicy{
		"policy1": wrapPolicy(&storage.Policy{
			Id:        "policy1",
			Notifiers: []string{"notifier1", "notifier2"},
		}),
		"policy2": wrapPolicy(&storage.Policy{
			Id:        "policy2",
			Notifiers: []string{"notifier2", "notifier3"},
		}),
		"policy3": wrapPolicy(&storage.Policy{
			Id:        "policy3",
			Notifiers: []string{"notifier1", "notifier2", "notifier3"},
		}),
		"policy4": wrapPolicy(&storage.Policy{
			Id:        "policy4",
			Notifiers: []string{"notifier1", "notifier3"},
		}),
	})

	var updatedPolicies []*storage.Policy
	policyDatastoreMock.EXPECT().UpdatePolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, policy *storage.Policy) error {
		updatedPolicies = append(updatedPolicies, policy)
		return nil
	})

	require.NoError(t, policySet.RemoveNotifier("notifier2"))

	expectedUpdates := []*storage.Policy{
		{
			Id:        "policy1",
			Notifiers: []string{"notifier1"},
		},
		{
			Id:        "policy2",
			Notifiers: []string{"notifier3"},
		},
		{
			Id:        "policy3",
			Notifiers: []string{"notifier1", "notifier3"},
		},
	}

	assert.ElementsMatch(t, expectedUpdates, updatedPolicies)
}
