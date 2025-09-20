//go:build sql_integration

package postgres

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	testDB := pgtest.ForT(t)
	store := New(testDB)

	const virtualMachineName1 = "Test VM 1"
	const virtualMachineName2 = "Test VM 2"
	const namespace1 = "namespace1"
	const namespace2 = "namespace2"
	const namespace3 = "namespace3"

	vm1 := createVirtualMachine(
		uuid.NewTestUUID(1).String(),
		virtualMachineName1,
		fixtureconsts.Cluster1,
		namespace1,
	)
	vm2 := createVirtualMachine(
		uuid.NewTestUUID(2).String(),
		virtualMachineName2,
		fixtureconsts.Cluster1,
		namespace2,
	)
	vm3 := createVirtualMachine(
		uuid.NewTestUUID(3).String(),
		virtualMachineName1,
		fixtureconsts.Cluster2,
		namespace3,
	)
	injectVirtualMachine(t, store, vm1)
	injectVirtualMachine(t, store, vm2)
	injectVirtualMachine(t, store, vm3)

	for name, tc := range map[string]struct {
		targetSearchField search.FieldLabel
		searchFieldValue  string
		expectedResult    []*storage.VirtualMachine
	}{
		"Search by Cluster ID returns all VMs in the cluster": {
			targetSearchField: search.ClusterID,
			searchFieldValue:  fixtureconsts.Cluster1,
			expectedResult:    []*storage.VirtualMachine{vm1, vm2},
		},
		"Search by Namespace returns the VM in the namespace": {
			targetSearchField: search.Namespace,
			searchFieldValue:  namespace1,
			expectedResult:    []*storage.VirtualMachine{vm1},
		},
		"Search by Name returns the VMs with matching names": {
			targetSearchField: search.VirtualMachineName,
			searchFieldValue:  virtualMachineName1,
			expectedResult:    []*storage.VirtualMachine{vm1, vm3},
		},
		"Search by ID returns the one VM matching when any": {
			targetSearchField: search.VirtualMachineID,
			searchFieldValue:  uuid.NewTestUUID(2).String(),
			expectedResult:    []*storage.VirtualMachine{vm2},
		},
		"Search by ID returns no VM if there is no match": {
			targetSearchField: search.VirtualMachineID,
			searchFieldValue:  uuid.NewTestUUID(42).String(),
			expectedResult:    []*storage.VirtualMachine{},
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := sac.WithAllAccess(it.Context())
			query := createFieldSearchQuery(tc.targetSearchField, tc.searchFieldValue)
			results, err := store.GetByQuery(ctx, query)
			assert.NoError(it, err)
			protoassert.ElementsMatch(it, tc.expectedResult, results)
		})
	}
}

func createVirtualMachine(
	virtualMachineID string,
	virtualMachineName string,
	virtualMachineClusterID string,
	virtualMachineNamespace string,
) *storage.VirtualMachine {
	return &storage.VirtualMachine{
		Id:        virtualMachineID,
		Namespace: virtualMachineNamespace,
		Name:      virtualMachineName,
		ClusterId: virtualMachineClusterID,
	}
}

func injectVirtualMachine(
	t *testing.T,
	store Store,
	vm *storage.VirtualMachine,
) {
	ctx := sac.WithAllAccess(t.Context())
	err := store.Upsert(ctx, vm)
	require.NoError(t, err)
}

func createFieldSearchQuery(
	targetField search.FieldLabel,
	targetValue string,
) *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(targetField, targetValue).ProtoQuery()
}
