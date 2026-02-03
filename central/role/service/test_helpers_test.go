//go:build sql_integration

package service

import (
	"fmt"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	rolePkg "github.com/stackrox/rox/central/role"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

const (
	namespaceUUIDNamespace = "namespace"
)

var (
	nilTraits *storage.Traits = nil

	imperativeOriginTraits = &storage.Traits{Origin: storage.Traits_IMPERATIVE}

	declarativeOriginTraits = &storage.Traits{Origin: storage.Traits_DECLARATIVE}

	orphanedDeclarativeOriginTraits = &storage.Traits{Origin: storage.Traits_DECLARATIVE_ORPHANED}

	ephemeralOriginTraits = &storage.Traits{Origin: storage.Traits_EPHEMERAL}
)

type serviceImplTester struct {
	postgres *pgtest.TestPostgres
	service  Service

	roleStore      roleDataStore.DataStore
	clusterStore   clusterDataStore.DataStore
	namespaceStore namespaceDataStore.DataStore

	storedClusterIDs   []string
	storedNamespaceIDs []string
	clusterNameToIDMap map[string]string

	storedPermissionSetIDs []string
	storedAccessScopeIDs   []string
	storedRoleNames        []string
}

func (s *serviceImplTester) Setup(t *testing.T) {
	if s == nil {
		return
	}

	var err error
	s.postgres = pgtest.ForT(t)
	require.NotNil(t, s.postgres)
	s.roleStore = roleDataStore.GetTestPostgresDataStore(t, s.postgres.DB)
	clusterStore, err := clusterDataStore.GetTestPostgresDataStore(t, s.postgres.DB)
	require.NoError(t, err)
	s.clusterStore = clusterStore
	namespaceStore, err := namespaceDataStore.GetTestPostgresDataStore(t, s.postgres.DB)
	require.NoError(t, err)
	s.namespaceStore = namespaceStore

	s.service = New(s.roleStore, s.clusterStore, s.namespaceStore)
}

func (s *serviceImplTester) SetupTest(t *testing.T) {
	if s == nil {
		return
	}

	s.storedAccessScopeIDs = make([]string, 0)
	s.storedPermissionSetIDs = make([]string, 0)
	s.storedRoleNames = make([]string, 0)

	s.storedClusterIDs = make([]string, 0)
	s.storedNamespaceIDs = make([]string, 0)
	s.clusterNameToIDMap = make(map[string]string, 0)

	writeCtx := sac.WithAllAccess(t.Context())

	for _, cluster := range storageClusters {
		clusterToAdd := cluster.CloneVT()
		clusterToAdd.Id = ""
		clusterToAdd.MainImage = "quay.io/rhacs-eng/main:latest"
		id, err := s.clusterStore.AddCluster(writeCtx, clusterToAdd)
		require.NoError(t, err)
		s.clusterNameToIDMap[clusterToAdd.GetName()] = id
		s.storedClusterIDs = append(s.storedClusterIDs, id)
	}

	for _, namespace := range storageNamespaces {
		ns := namespace.CloneVT()
		ns.Id = getNamespaceID(ns.GetName())
		ns.ClusterId = s.clusterNameToIDMap[ns.GetClusterName()]
		require.NoError(t, s.namespaceStore.AddNamespace(writeCtx, ns))
		s.storedNamespaceIDs = append(s.storedNamespaceIDs, ns.GetId())
	}
}

func (s *serviceImplTester) TearDownTest(t *testing.T) {
	if s == nil {
		return
	}

	writeCtx := sac.WithAllAccess(t.Context())
	for _, clusterID := range s.storedClusterIDs {
		doneSignal := concurrency.NewSignal()
		require.NoError(t, s.clusterStore.RemoveCluster(writeCtx, clusterID, &doneSignal))
		require.Eventually(t,
			func() bool { return doneSignal.IsDone() },
			5*time.Second,
			10*time.Millisecond,
		)
	}
	s.storedClusterIDs = s.storedClusterIDs[:0]
	for _, namespaceID := range s.storedNamespaceIDs {
		require.NoError(t, s.namespaceStore.RemoveNamespace(writeCtx, namespaceID))
	}
	for _, roleName := range s.storedRoleNames {
		s.deleteRole(t, roleName)
	}
	for _, permissionSetID := range s.storedPermissionSetIDs {
		s.deletePermissionSet(t, permissionSetID)
	}
	for _, accessScopeID := range s.storedAccessScopeIDs {
		s.deleteAccessScope(t, accessScopeID)
	}
}

func (s *serviceImplTester) createRole(t *testing.T, roleName string, traits *storage.Traits) *storage.Role {
	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)

	ps := s.createPermissionSet(t, roleName, traits)
	scope := s.createAccessScope(t, roleName, traits)

	role := &storage.Role{
		Name:            roleName,
		Description:     fmt.Sprintf("Test role for %s", roleName),
		PermissionSetId: ps.GetId(),
		AccessScopeId:   scope.GetId(),
		Traits:          traits,
	}

	err := s.roleStore.AddRole(ctx, role)
	require.NoError(t, err)
	s.storedRoleNames = append(s.storedRoleNames, roleName)

	return role
}

func (s *serviceImplTester) deleteRole(t *testing.T, roleName string) {
	if s == nil {
		return
	}

	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)
	request := &v1.ResourceByID{
		Id: roleName,
	}
	_, deleteErr := s.service.DeleteRole(ctx, request)
	require.NoError(t, deleteErr)
}

func (s *serviceImplTester) createPermissionSet(t *testing.T, name string, traits *storage.Traits) *storage.PermissionSet {
	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)
	permissionSet := &storage.PermissionSet{
		Name:             name,
		Description:      fmt.Sprintf("Test permission set for %s", name),
		ResourceToAccess: nil,
		Traits:           traits,
	}
	permissionSet.Id = rolePkg.GeneratePermissionSetID()
	err := s.roleStore.UpsertPermissionSet(ctx, permissionSet)
	require.NoError(t, err)
	s.storedPermissionSetIDs = append(s.storedPermissionSetIDs, permissionSet.GetId())
	return permissionSet
}

func (s *serviceImplTester) deletePermissionSet(t *testing.T, permissionSetID string) {
	if s == nil {
		return
	}

	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)
	request := &v1.ResourceByID{
		Id: permissionSetID,
	}
	_, deleteErr := s.service.DeletePermissionSet(ctx, request)
	require.NoError(t, deleteErr)
}

func (s *serviceImplTester) createAccessScope(t *testing.T, name string, traits *storage.Traits) *storage.SimpleAccessScope {
	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)
	scope := &storage.SimpleAccessScope{
		Name:        name,
		Description: fmt.Sprintf("Test access scope for %s", name),
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"test"},
		},
		Traits: traits,
	}
	scope.Id = rolePkg.GenerateAccessScopeID()
	err := s.roleStore.UpsertAccessScope(ctx, scope)
	require.NoError(t, err)
	s.storedAccessScopeIDs = append(s.storedAccessScopeIDs, scope.GetId())
	return scope
}

func (s *serviceImplTester) deleteAccessScope(t *testing.T, accessScopeID string) {
	if s == nil {
		return
	}

	ctx := sac.WithAllAccess(t.Context())
	ctx = declarativeconfig.WithModifyDeclarativeOrImperative(ctx)
	request := &v1.ResourceByID{
		Id: accessScopeID,
	}
	_, deleteErr := s.service.DeleteSimpleAccessScope(ctx, request)
	require.NoError(t, deleteErr)
}

func getNamespaceID(namespaceName string) string {
	return uuid.NewV5FromNonUUIDs(namespaceUUIDNamespace, namespaceName).String()
}
