package detection

import (
	"context"
	"testing"

	mocks2 "github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
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
		"policy1": wrapPolicy(storage.Policy_builder{
			Id:        "policy1",
			Notifiers: []string{"notifier1", "notifier2"},
		}.Build()),
		"policy2": wrapPolicy(storage.Policy_builder{
			Id:        "policy2",
			Notifiers: []string{"notifier2", "notifier3"},
		}.Build()),
		"policy3": wrapPolicy(storage.Policy_builder{
			Id:        "policy3",
			Notifiers: []string{"notifier1", "notifier2", "notifier3"},
		}.Build()),
		"policy4": wrapPolicy(storage.Policy_builder{
			Id:        "policy4",
			Notifiers: []string{"notifier1", "notifier3"},
		}.Build()),
	})

	var updatedPolicies []*storage.Policy
	policyDatastoreMock.EXPECT().UpdatePolicy(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, policy *storage.Policy) error {
		updatedPolicies = append(updatedPolicies, policy)
		return nil
	})

	require.NoError(t, policySet.RemoveNotifier("notifier2"))

	policy := &storage.Policy{}
	policy.SetId("policy1")
	policy.SetNotifiers([]string{"notifier1"})
	policy2 := &storage.Policy{}
	policy2.SetId("policy2")
	policy2.SetNotifiers([]string{"notifier3"})
	policy3 := &storage.Policy{}
	policy3.SetId("policy3")
	policy3.SetNotifiers([]string{"notifier1", "notifier3"})
	expectedUpdates := []*storage.Policy{
		policy,
		policy2,
		policy3,
	}

	protoassert.ElementsMatch(t, expectedUpdates, updatedPolicies)
}
