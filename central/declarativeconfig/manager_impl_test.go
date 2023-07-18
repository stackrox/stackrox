package declarativeconfig

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	declarativeConfigHealthMock "github.com/stackrox/rox/central/declarativeconfig/health/datastore/mocks"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	updaterMocks "github.com/stackrox/rox/central/declarativeconfig/updater/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	transformMocks "github.com/stackrox/rox/pkg/declarativeconfig/transform/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const supportedTypesCount = 6

func newTestManager(t *testing.T) *managerImpl {
	m := New(100*time.Millisecond, 100*time.Millisecond, map[reflect.Type]updater.ResourceUpdater{},
		nil, types.UniversalNameExtractor(), types.UniversalIDExtractor())

	mImpl, ok := m.(*managerImpl)
	require.True(t, ok)
	return mImpl
}

// Custom gomock.Matcher for storage.IntegrationHealth that ignores the timestamp field's value, but instead only checks
// that its set.
type declarativeConfigHealthMatcher struct {
	expected *storage.DeclarativeConfigHealth
}

func (i *declarativeConfigHealthMatcher) Matches(x interface{}) bool {
	integrationHealth, ok := x.(*storage.DeclarativeConfigHealth)

	if !ok {
		return false
	}

	return i.expected.GetId() == integrationHealth.GetId() &&
		i.expected.GetName() == integrationHealth.GetName() &&
		i.expected.GetStatus() == integrationHealth.GetStatus() &&
		i.expected.GetResourceType() == integrationHealth.GetResourceType() &&
		i.expected.GetResourceName() == integrationHealth.GetResourceName() &&
		i.expected.GetErrorMessage() == integrationHealth.GetErrorMessage() &&
		integrationHealth.GetLastTimestamp() != nil
}
func (i *declarativeConfigHealthMatcher) String() string {
	return fmt.Sprintf("%+v", i.expected)
}

func matchDeclarativeConfigHealth(int *storage.DeclarativeConfigHealth) gomock.Matcher {
	return &declarativeConfigHealthMatcher{
		expected: int,
	}
}

func TestReconcileTransformedMessages_Success(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "id-perm-set-1",
	}
	permissionSet2 := &storage.PermissionSet{
		Name: "permission-set-2",
		Id:   "id-perm-set-2",
	}
	accessScope := &storage.SimpleAccessScope{
		Name: "accessScope",
		Id:   "id-access-scope",
	}
	role := &storage.Role{
		Name: "role",
	}
	authProvider := &storage.AuthProvider{
		Name: "authProvider",
		Id:   "id-auth-provider",
	}
	group := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "group",
			AuthProviderId: "some-auth-provider",
			Key:            "email",
			Value:          "some@example.com",
		},
		RoleName: "Admin",
	}
	notifier := &storage.Notifier{
		Name: "notifierName",
		Id:   "notifierId",
		Config: &storage.Notifier_Splunk{
			Splunk: &storage.Splunk{
				HttpToken: "http-token",
			},
		},
	}

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), accessScope),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet2),
		mockUpdater.EXPECT().Upsert(gomock.Any(), role),
		mockUpdater.EXPECT().Upsert(gomock.Any(), authProvider),
		mockUpdater.EXPECT().Upsert(gomock.Any(), group),
		mockUpdater.EXPECT().Upsert(gomock.Any(), notifier),
	)

	gomock.InOrder(
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "id-access-scope",
			Name:         "accessScope in config map test-handler-1",
			ResourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
			ResourceName: "accessScope",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "id-perm-set-1",
			Name:         "permission-set-1 in config map test-handler-1",
			ResourceType: storage.DeclarativeConfigHealth_PERMISSION_SET,
			ResourceName: "permission-set-1",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "id-perm-set-2",
			Name:         "permission-set-2 in config map test-handler-1",
			ResourceType: storage.DeclarativeConfigHealth_PERMISSION_SET,
			ResourceName: "permission-set-2",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
			Name:         "role in config map test-handler-2",
			ResourceType: storage.DeclarativeConfigHealth_ROLE,
			ResourceName: "role",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "id-auth-provider",
			Name:         "authProvider in config map test-handler-2",
			ResourceType: storage.DeclarativeConfigHealth_AUTH_PROVIDER,
			ResourceName: "authProvider",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "group",
			Name:         "group email:some@example.com:Admin for auth provider ID some-auth-provider in config map test-handler-2",
			ResourceType: storage.DeclarativeConfigHealth_GROUP,
			ResourceName: "group email:some@example.com:Admin for auth provider ID some-auth-provider",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
			Id:           "notifierId",
			Name:         "notifierName in config map test-handler-2",
			ResourceType: storage.DeclarativeConfigHealth_NOTIFIER,
			ResourceName: "notifierName",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ErrorMessage: "",
		})),
	)

	// Delete resources should be called in order, ignoring the existing IDs from the previously upserted resources.
	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"notifierId"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"group"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"id-auth-provider"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"role"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.InAnyOrder([]string{"id-perm-set-1", "id-perm-set-2"})).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"id-access-scope"}).Return([]string{"skipping-scope"}, errors.New("some-error")),
	)

	// We retrieve the integration healths on the deletion, only the non-ignored ID that does not have "Config Map"
	// in its name should be deleted.
	gomock.InOrder(
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).Return([]*storage.DeclarativeConfigHealth{
			{
				Id:           "some-id",
				Name:         "Config Map some-config-map",
				ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
			},
			{
				Id:           "notifierId",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_NOTIFIER,
			},
			{
				Id:           "group",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_GROUP,
			},
			{
				Id:           "id-auth-provider",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_AUTH_PROVIDER,
			},
			{
				Id:           "role",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_ROLE,
			},
			{
				Id:           "skipping-scope",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
			},
			{
				Id:           "some-non-existent-id",
				Name:         "I should be deleted",
				ResourceType: storage.DeclarativeConfigHealth_GROUP,
			},
		}, nil),
		mockHealthDS.EXPECT().RemoveDeclarativeConfig(gomock.Any(), "some-non-existent-id"),
	)

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.RoleType:          mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.GroupType:         mockUpdater,
		types.NotifierType:      mockUpdater,
	}
	m.declarativeConfigHealthDS = mockHealthDS

	m.reconcileTransformedMessages(map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
				permissionSet2,
			},
			types.AccessScopeType: []proto.Message{
				accessScope,
			},
		},
		"test-handler-2": {
			types.RoleType: []proto.Message{
				role,
			},
			types.AuthProviderType: []proto.Message{
				authProvider,
			},
			types.GroupType: []proto.Message{
				group,
			},
			types.NotifierType: []proto.Message{
				notifier,
			},
		},
	})
}

func TestReconcileTransformedMessages_ErrorPropagatedToReporter(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "some-id",
	}

	testError := errors.New("test error")
	mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(testError).Times(3)

	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
		Id:           "some-id",
		Name:         "permission-set-1 in config map test-handler-1",
		ResourceType: storage.DeclarativeConfigHealth_PERMISSION_SET,
		ResourceName: "permission-set-1",
		Status:       storage.DeclarativeConfigHealth_UNHEALTHY,
		ErrorMessage: "test error",
	}))

	mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(supportedTypesCount)

	mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
		Return(nil, nil).Times(1)

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
		types.NotifierType:      mockUpdater,
	}
	m.declarativeConfigHealthDS = mockHealthDS

	// We need to call this 3 times, only then the error will be propagated to the mockHealthDS.
	for i := 0; i < consecutiveReconciliationErrorThreshold; i++ {
		m.reconcileTransformedMessages(map[string]protoMessagesByType{
			"test-handler-1": {
				types.PermissionSetType: []proto.Message{
					permissionSet1,
				},
			},
		})
	}
}

func TestReconcileTransformedMessages_SkipReconciliationWithNoChanges(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "some-id",
	}

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
		types.NotifierType:      mockUpdater,
	}
	m.declarativeConfigHealthDS = mockHealthDS

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(nil).Times(1),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), gomock.Any()).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(supportedTypesCount),
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
			Return(nil, nil).Times(1),
	)

	messages := map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
			},
		},
	}

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())

	// 2. Run the reconciliation again which should be a no-op. Nothing should be called.

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())
}

func TestReconcileTransformedMessages_SkipDeletion(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "some-id",
	}

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
		types.NotifierType:      mockUpdater,
	}
	m.declarativeConfigHealthDS = mockHealthDS

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(errors.New("some error")).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(supportedTypesCount),
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
			Return(nil, nil).Times(1),
	)

	messages := map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
			},
		},
	}

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.True(t, m.lastUpsertFailed.Get())

	// 2. Run the reconciliation again. Only upsert should be done.

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(errors.New("some error")).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.True(t, m.lastUpsertFailed.Get())

	// 3. Run the reconciliation again. Only upsert should be done, and if successful no upsert error should be indicated.

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(nil).Times(1),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), gomock.Any()).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())
}

func TestReconcileTransformedMessages_SkipUpsert(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "some-id",
	}

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
		types.NotifierType:      mockUpdater,
	}
	m.declarativeConfigHealthDS = mockHealthDS

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(nil).Times(1),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), gomock.Any()).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error")).Times(supportedTypesCount),
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
			Return(nil, nil).Times(1),
	)

	messages := map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
			},
		},
	}

	m.reconcileTransformedMessages(messages)
	assert.True(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())

	// 2. Run the reconciliation again. Only deletion should be done.

	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error")).Times(supportedTypesCount),
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
			Return(nil, nil).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.True(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())

	// 3. Run the reconciliation again. Only deletion should be done, and if successful no deletion error should be indicated.

	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(supportedTypesCount),
		mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
			Return(nil, nil).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())
}

func TestUpdateDeclarativeConfigContents_RegisterHealthStatus(t *testing.T) {
	controller := gomock.NewController(t)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)
	transformer := transformMocks.NewMockTransformer(controller)

	m := newTestManager(t)
	m.universalTransformer = transformer
	m.declarativeConfigHealthDS = mockHealthDS

	transformer.EXPECT().Transform(&declarativeconfig.Role{
		Name:          "test-name",
		Description:   "test-description",
		AccessScope:   "access-scope",
		PermissionSet: "permission-set",
	}).Return(map[reflect.Type][]proto.Message{
		types.RoleType: {
			&storage.Role{
				Name:            "test-name",
				Description:     "test-description",
				PermissionSetId: "access-scope",
				AccessScopeId:   "permission-set",
			},
		},
	}, nil)

	mockHealthDS.EXPECT().GetDeclarativeConfig(gomock.Any(), gomock.Any()).Return(nil, false, nil).AnyTimes()

	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
		Id:           "04a87e34-b568-5e14-90ac-380d25c8689b",
		Name:         "test-name in config map my-cool-config-map",
		ResourceType: storage.DeclarativeConfigHealth_ROLE,
		ResourceName: "test-name",
		Status:       storage.DeclarativeConfigHealth_HEALTHY,
		ErrorMessage: "",
	}))

	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
		Id:           declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String(),
		Name:         "Config Map my-cool-config-map",
		ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
		ResourceName: "my-cool-config-map",
		Status:       storage.DeclarativeConfigHealth_HEALTHY,
		ErrorMessage: "",
	}))

	m.UpdateDeclarativeConfigContents("/some/config/dir/to/my-cool-config-map", [][]byte{
		[]byte(`
name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`),
	})

}

func TestUpdateDeclarativeConfigContents_Errors(t *testing.T) {
	controller := gomock.NewController(t)
	reporter := declarativeConfigHealthMock.NewMockDataStore(controller)
	transformer := transformMocks.NewMockTransformer(controller)

	m := newTestManager(t)
	m.universalTransformer = transformer
	m.declarativeConfigHealthDS = reporter

	// 1. Failure in unmarshalling the file.
	reporter.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
		Id:           declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String(),
		Name:         "Config Map my-cool-config-map",
		ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
		ResourceName: "my-cool-config-map",
		Status:       storage.DeclarativeConfigHealth_UNHEALTHY,
		ErrorMessage: "could not unmarshal configuration into any of the supported types [auth-provider,access-scope,permission-set,role,notifier]",
	}))

	m.UpdateDeclarativeConfigContents("/some/config/dir/to/my-cool-config-map", [][]byte{
		[]byte(`
{"cool": "key", "value": "pairs"}
`),
	})

	// 2. Multiple failures in transformation.

	transformer.EXPECT().Transform(gomock.Any()).Return(nil, errors.New("some-error-happened"))

	reporter.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(&storage.DeclarativeConfigHealth{
		Id:           declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String(),
		Name:         "Config Map my-cool-config-map",
		ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
		ResourceName: "my-cool-config-map",
		Status:       storage.DeclarativeConfigHealth_UNHEALTHY,
		ErrorMessage: "during transforming configuration: 1 error occurred:\n\t* some-error-happened\n\n",
	}))

	m.UpdateDeclarativeConfigContents("/some/config/dir/to/my-cool-config-map", [][]byte{
		[]byte(`
name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`),
	})
}

func TestVerifyUpdaters(t *testing.T) {
	m := newTestManager(t)
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)

	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
		types.NotifierType:      mockUpdater,
	}

	err := m.verifyUpdaters()
	assert.NoError(t, err)

	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: nil,
	}
	err = m.verifyUpdaters()
	assert.ErrorIs(t, err, errox.InvariantViolation)

	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          nil,
		types.NotifierType:      mockUpdater,
	}

	err = m.verifyUpdaters()
	assert.ErrorIs(t, err, errox.InvariantViolation)
}
