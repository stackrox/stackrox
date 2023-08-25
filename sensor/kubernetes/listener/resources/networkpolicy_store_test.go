package resources

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNetworkPoliciesStoreFind(t *testing.T) {
	store := newNetworkPoliciesStore()
	store.Upsert(newNPDummy("one", defaultNS, map[string]string{"app": "sensor"}))
	store.Upsert(newNPDummy("two", defaultNS, map[string]string{"app": "sensor", "role": "backend"}))
	store.Upsert(newNPDummy("three", defaultNS, map[string]string{"app": "central"}))
	store.Upsert(newNPDummy("four", defaultNS, map[string]string{"app": "central", "role": "frontend"}))
	store.Upsert(newNPDummy("five", defaultNS, map[string]string{}))

	tests := []struct {
		name        string
		podLabels   map[string]string
		expectedIDs []string
	}{
		{
			name:        "Single kv",
			podLabels:   map[string]string{"app": "sensor"},
			expectedIDs: []string{"one", "five"},
		},
		{
			name:        "Double kv full match",
			podLabels:   map[string]string{"app": "sensor", "role": "backend"},
			expectedIDs: []string{"one", "two", "five"},
		},
		{
			name:        "Double kv no match",
			podLabels:   map[string]string{"app": "xxx", "role": "backend"},
			expectedIDs: []string{"five"},
		},
		{
			name:        "Extra label in the middle",
			podLabels:   map[string]string{"app": "sensor", "aaa": "xxx", "role": "backend"},
			expectedIDs: []string{"one", "two", "five"},
		},
		{
			name:        "Unsorted labels",
			podLabels:   map[string]string{"aaa": "xxx", "role": "backend", "app": "sensor"},
			expectedIDs: []string{"one", "two", "five"},
		},
		{
			name:        "Empty label value",
			podLabels:   map[string]string{"role": "", "app": "central"},
			expectedIDs: []string{"three", "five"},
		},
		{
			name:        "Pod with 0 labels should only match policies with 0 selectors",
			podLabels:   map[string]string{},
			expectedIDs: []string{"five"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := store.Find(defaultNS, tt.podLabels)
			assert.Equal(t, len(tt.expectedIDs), len(found), "expected to find %d IDs, but found: %v", len(tt.expectedIDs), found)

			for _, expID := range tt.expectedIDs {
				assert.Equal(t, expID, found[expID].GetId())
			}
		})
	}
}

func TestNetworkPoliciesStoreCleanup(t *testing.T) {
	policies := make([]*storage.NetworkPolicy, 0)
	policies = append(policies, newNPDummy("p1", defaultNS, map[string]string{"app": "sensor"}))
	policies = append(policies, newNPDummy("p2", defaultNS, map[string]string{"app": "sensor", "role": "backend"}))
	policies = append(policies, newNPDummy("p3", defaultNS, map[string]string{"app": "central"}))
	policies = append(policies, newNPDummy("p4", defaultNS, map[string]string{"app": "central", "role": "frontend"}))

	store := newNetworkPoliciesStore()
	for _, p := range policies {
		store.Upsert(p)
	}

	store.Cleanup()

	for _, p := range policies {
		assert.Nil(t, store.Get(p.GetId()))
		assert.Len(t, store.Find(defaultNS, p.GetLabels()), 0)
	}
}

func TestNetworkPoliciesStoreDelete(t *testing.T) {
	policies := make([]*storage.NetworkPolicy, 0)
	policies = append(policies, newNPDummy("p1", defaultNS, map[string]string{"app": "sensor"}))
	policies = append(policies, newNPDummy("p2", defaultNS, map[string]string{"app": "sensor", "role": "backend"}))
	policies = append(policies, newNPDummy("p3", defaultNS, map[string]string{"app": "central"}))
	policies = append(policies, newNPDummy("p4", defaultNS, map[string]string{"app": "central", "role": "frontend"}))

	tests := []struct {
		name            string
		deleteID        string
		getIDExpectedID map[string]string
	}{
		{
			name:            "Delete p1",
			deleteID:        "p1",
			getIDExpectedID: map[string]string{"p1": "", "p2": "p2"},
		},
		{
			name:            "Delete p2",
			deleteID:        "p2",
			getIDExpectedID: map[string]string{"p1": "p1", "p2": ""},
		},
		{
			name:            "Delete not existing",
			deleteID:        "zzz",
			getIDExpectedID: map[string]string{"p1": "p1", "p2": "p2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newNetworkPoliciesStore()
			for _, p := range policies {
				store.Upsert(p)
			}

			store.Delete(tt.deleteID, defaultNS)
			for getID, expectedID := range tt.getIDExpectedID {
				got := store.Get(getID)
				if expectedID == "" && got != nil {
					t.Errorf("Expected to not find ID but found %s", got.GetId())
				} else {
					assert.Equal(t, expectedID, got.GetId())
				}
			}
		})
	}
}

func TestNetworkPoliciesStoreAll(t *testing.T) {
	p0 := newNPDummy("p0", defaultNS, map[string]string{})
	p1 := newNPDummy("p1", defaultNS, map[string]string{"app": "sensor"})
	p2 := newNPDummy("p2", defaultNS, map[string]string{"app": "sensor", "role": "backend"})
	p3 := newNPDummy("p3", "otherNS", map[string]string{"role": "other"})

	tests := []struct {
		name          string
		policiesToAdd []*storage.NetworkPolicy
		expectedIDs   []string
	}{
		{
			name:          "Empty store",
			policiesToAdd: []*storage.NetworkPolicy{},
			expectedIDs:   []string{},
		},
		{
			name:          "One policy",
			policiesToAdd: []*storage.NetworkPolicy{p1},
			expectedIDs:   []string{"p1"},
		},
		{
			name:          "One policy with 0 selectors",
			policiesToAdd: []*storage.NetworkPolicy{p0},
			expectedIDs:   []string{"p0"},
		},
		{
			name:          "Two policies",
			policiesToAdd: []*storage.NetworkPolicy{p1, p2},
			expectedIDs:   []string{"p1", "p2"},
		},
		{
			name:          "Two policies different namespaces",
			policiesToAdd: []*storage.NetworkPolicy{p2, p3},
			expectedIDs:   []string{"p2", "p3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newNetworkPoliciesStore()
			for _, p := range tt.policiesToAdd {
				store.Upsert(p)
			}
			got := store.All()
			for _, expectedID := range tt.expectedIDs {
				assert.Contains(t, got, expectedID)
			}
			assert.Equal(t, len(tt.expectedIDs), store.Size())
			assert.Equal(t, len(tt.expectedIDs), len(got))
		})
	}
}

func TestNetworkPoliciesStoreOnNamespaceDeleted(t *testing.T) {
	ns0 := "random-namespace"
	ns1 := "other-namespace"
	p0 := newNPDummy("p0", defaultNS, map[string]string{})
	p1 := newNPDummy("p1", ns0, map[string]string{"app": "sensor"})
	p2 := newNPDummy("p2", ns0, map[string]string{"app": "sensor", "role": "backend"})
	p3 := newNPDummy("p3", ns1, map[string]string{"role": "other"})

	tests := []struct {
		name          string
		policiesToAdd []*storage.NetworkPolicy
		nsToDelete    string
		expectedIDs   []string
	}{
		{
			name:          "Empty store",
			policiesToAdd: []*storage.NetworkPolicy{},
			nsToDelete:    ns0,
			expectedIDs:   []string{},
		},
		{
			name:          "Policy in a different namespace",
			policiesToAdd: []*storage.NetworkPolicy{p0},
			nsToDelete:    ns0,
			expectedIDs:   []string{"p0"},
		},
		{
			name:          "Policy in same namespace",
			policiesToAdd: []*storage.NetworkPolicy{p1},
			nsToDelete:    ns0,
			expectedIDs:   []string{},
		},
		{
			name:          "Multiple namespaces",
			policiesToAdd: []*storage.NetworkPolicy{p1, p2, p3},
			nsToDelete:    ns0,
			expectedIDs:   []string{"p3"},
		},
		{
			name:          "Multiple policies, no policies in namespace",
			policiesToAdd: []*storage.NetworkPolicy{p0, p1, p2},
			nsToDelete:    ns1,
			expectedIDs:   []string{"p0", "p1", "p2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newNetworkPoliciesStore()
			for _, p := range tt.policiesToAdd {
				store.Upsert(p)
			}
			store.OnNamespaceDeleted(tt.nsToDelete)
			got := store.All()
			for _, expectedID := range tt.expectedIDs {
				assert.Contains(t, got, expectedID)
			}
			assert.Equal(t, len(tt.expectedIDs), store.Size())
			assert.Equal(t, len(tt.expectedIDs), len(got))
		})
	}
}
