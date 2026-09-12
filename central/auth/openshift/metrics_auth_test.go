package openshift

import (
	"context"
	"testing"

	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	providerMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/authproviders/userpki"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"go.uber.org/mock/gomock"
)

const testCAPEM = "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----\n"

func setupMocks(t *testing.T) (*providerMocks.MockRegistry, *roleMocks.MockDataStore, *groupMocks.MockDataStore) {
	ctrl := gomock.NewController(t)
	return providerMocks.NewMockRegistry(ctrl),
		roleMocks.NewMockDataStore(ctrl),
		groupMocks.NewMockDataStore(ctrl)
}

func TestSeed_FirstBoot_CreatesAllObjects(t *testing.T) {
	registry, roleDS, groupDS := setupMocks(t)
	ctx := context.Background()

	roleDS.EXPECT().GetPermissionSet(gomock.Any(), permissionSetID).Return(nil, false, nil)
	roleDS.EXPECT().AddPermissionSet(gomock.Any(), gomock.Any()).Return(nil)
	roleDS.EXPECT().GetRole(gomock.Any(), roleName).Return(nil, false, nil)
	roleDS.EXPECT().AddRole(gomock.Any(), gomock.Any()).Return(nil)

	providerCreated := false
	registry.EXPECT().GetProvider(authProviderID).Return(nil)
	registry.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ ...interface{}) (interface{}, error) {
			providerCreated = true
			return nil, nil
		})
	groupDS.EXPECT().GetFiltered(gomock.Any(), gomock.Any()).Return(nil, nil)
	groupDS.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *storage.Group) error {
			if !providerCreated {
				t.Error("group Add called before provider creation")
			}
			return nil
		})

	ensurePermissionSet(ctx, roleDS)
	ensureRole(ctx, roleDS)
	ensureAuthProvider(ctx, registry, testCAPEM)
	ensureGroup(ctx, groupDS)
}

func TestSeed_SubsequentBoot_CAUnchanged_NoUpdate(t *testing.T) {
	registry, _, _ := setupMocks(t)
	ctx := context.Background()

	existing := providerMocks.NewMockProvider(gomock.NewController(t))
	existing.EXPECT().StorageView().Return(&storage.AuthProvider{
		Config: map[string]string{userpki.ConfigKeys: testCAPEM},
	})
	registry.EXPECT().GetProvider(authProviderID).Return(existing)

	ensureAuthProvider(ctx, registry, testCAPEM)
}

func TestSeed_CARotation_UpdatesConfig(t *testing.T) {
	registry, _, _ := setupMocks(t)
	ctx := context.Background()

	existing := providerMocks.NewMockProvider(gomock.NewController(t))
	existing.EXPECT().StorageView().Return(&storage.AuthProvider{
		Config: map[string]string{userpki.ConfigKeys: "old-ca"},
	})
	registry.EXPECT().GetProvider(authProviderID).Return(existing)
	registry.EXPECT().UpdateProvider(gomock.Any(), authProviderID, gomock.Any()).Return(nil, nil)

	ensureAuthProvider(ctx, registry, testCAPEM)
}

func TestSeed_PartialRecovery_PermissionSetExists_RoleMissing(t *testing.T) {
	registry, roleDS, groupDS := setupMocks(t)
	ctx := context.Background()

	roleDS.EXPECT().GetPermissionSet(gomock.Any(), permissionSetID).Return(&storage.PermissionSet{}, true, nil)
	roleDS.EXPECT().GetRole(gomock.Any(), roleName).Return(nil, false, nil)
	roleDS.EXPECT().AddRole(gomock.Any(), gomock.Any()).Return(nil)
	registry.EXPECT().GetProvider(authProviderID).Return(nil)
	registry.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).Return(nil, nil)
	groupDS.EXPECT().GetFiltered(gomock.Any(), gomock.Any()).Return(nil, nil)
	groupDS.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)

	ensurePermissionSet(ctx, roleDS)
	ensureRole(ctx, roleDS)
	ensureAuthProvider(ctx, registry, testCAPEM)
	ensureGroup(ctx, groupDS)
}

func TestSeed_Group_IsDefaultOriginAndImmutable(t *testing.T) {
	_, _, groupDS := setupMocks(t)
	ctx := context.Background()

	groupDS.EXPECT().GetFiltered(gomock.Any(), gomock.Any()).Return(nil, nil)
	groupDS.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(
		func(addCtx context.Context, group *storage.Group) error {
			if group.GetProps().GetTraits().GetOrigin() != storage.Traits_DEFAULT {
				t.Errorf("expected group to use DEFAULT origin, got %v", group.GetProps().GetTraits().GetOrigin())
			}
			if !declarativeconfig.CanModifyResource(addCtx, group.GetProps()) {
				t.Error("expected ensureGroup to pass a context that can modify DEFAULT-origin resources")
			}
			return nil
		})

	ensureGroup(ctx, groupDS)
}

func TestSeed_SubsequentBoot_GroupExists_NotOverwritten(t *testing.T) {
	_, _, groupDS := setupMocks(t)
	ctx := context.Background()

	groupDS.EXPECT().GetFiltered(gomock.Any(), gomock.Any()).Return([]*storage.Group{{
		Props:    &storage.GroupProperties{AuthProviderId: authProviderID, Key: "name", Value: "system:serviceaccount:openshift-monitoring:prometheus-k8s"},
		RoleName: "User Modified Role",
	}}, nil)

	ensureGroup(ctx, groupDS)
}
