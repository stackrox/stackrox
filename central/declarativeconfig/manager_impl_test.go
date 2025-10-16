package declarativeconfig

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	declarativeConfigHealthMock "github.com/stackrox/rox/central/declarativeconfig/health/datastore/mocks"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	updaterMocks "github.com/stackrox/rox/central/declarativeconfig/updater/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	transformMocks "github.com/stackrox/rox/pkg/declarativeconfig/transform/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

var supportedTypesCount = len(types.GetSupportedProtobufTypesInProcessingOrder())

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

	permissionSet1 := &storage.PermissionSet{}
	permissionSet1.SetName("permission-set-1")
	permissionSet1.SetId("id-perm-set-1")
	permissionSet2 := &storage.PermissionSet{}
	permissionSet2.SetName("permission-set-2")
	permissionSet2.SetId("id-perm-set-2")
	accessScope := &storage.SimpleAccessScope{}
	accessScope.SetName("accessScope")
	accessScope.SetId("id-access-scope")
	role := &storage.Role{}
	role.SetName("role")
	authProvider := &storage.AuthProvider{}
	authProvider.SetName("authProvider")
	authProvider.SetId("id-auth-provider")
	gp := &storage.GroupProperties{}
	gp.SetId("group")
	gp.SetAuthProviderId("some-auth-provider")
	gp.SetKey("email")
	gp.SetValue("some@example.com")
	group := &storage.Group{}
	group.SetProps(gp)
	group.SetRoleName("Admin")
	splunk := &storage.Splunk{}
	splunk.SetHttpToken("http-token")
	notifier := &storage.Notifier{}
	notifier.SetName("notifierName")
	notifier.SetId("notifierId")
	notifier.SetSplunk(proto.ValueOrDefault(splunk))
	m2mConfig := &storage.AuthMachineToMachineConfig{}
	m2mConfig.SetId("m2m-config-id")
	m2mConfig.SetIssuer("https://kubernetes.default.svc")

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), accessScope),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet2),
		mockUpdater.EXPECT().Upsert(gomock.Any(), role),
		mockUpdater.EXPECT().Upsert(gomock.Any(), authProvider),
		mockUpdater.EXPECT().Upsert(gomock.Any(), group),
		mockUpdater.EXPECT().Upsert(gomock.Any(), notifier),
		mockUpdater.EXPECT().Upsert(gomock.Any(), m2mConfig),
	)

	dch := &storage.DeclarativeConfigHealth{}
	dch.SetId("id-access-scope")
	dch.SetName("accessScope in config map test-handler-1")
	dch.SetResourceType(storage.DeclarativeConfigHealth_ACCESS_SCOPE)
	dch.SetResourceName("accessScope")
	dch.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch.SetErrorMessage("")
	dch2 := &storage.DeclarativeConfigHealth{}
	dch2.SetId("id-perm-set-1")
	dch2.SetName("permission-set-1 in config map test-handler-1")
	dch2.SetResourceType(storage.DeclarativeConfigHealth_PERMISSION_SET)
	dch2.SetResourceName("permission-set-1")
	dch2.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch2.SetErrorMessage("")
	dch3 := &storage.DeclarativeConfigHealth{}
	dch3.SetId("id-perm-set-2")
	dch3.SetName("permission-set-2 in config map test-handler-1")
	dch3.SetResourceType(storage.DeclarativeConfigHealth_PERMISSION_SET)
	dch3.SetResourceName("permission-set-2")
	dch3.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch3.SetErrorMessage("")
	dch4 := &storage.DeclarativeConfigHealth{}
	dch4.SetId("61a68f2a-2599-5a9f-a98a-8fc83e2c06cf")
	dch4.SetName("role in config map test-handler-2")
	dch4.SetResourceType(storage.DeclarativeConfigHealth_ROLE)
	dch4.SetResourceName("role")
	dch4.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch4.SetErrorMessage("")
	dch5 := &storage.DeclarativeConfigHealth{}
	dch5.SetId("id-auth-provider")
	dch5.SetName("authProvider in config map test-handler-2")
	dch5.SetResourceType(storage.DeclarativeConfigHealth_AUTH_PROVIDER)
	dch5.SetResourceName("authProvider")
	dch5.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch5.SetErrorMessage("")
	dch6 := &storage.DeclarativeConfigHealth{}
	dch6.SetId("group")
	dch6.SetName("group email:some@example.com:Admin for auth provider ID some-auth-provider in config map test-handler-2")
	dch6.SetResourceType(storage.DeclarativeConfigHealth_GROUP)
	dch6.SetResourceName("group email:some@example.com:Admin for auth provider ID some-auth-provider")
	dch6.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch6.SetErrorMessage("")
	dch7 := &storage.DeclarativeConfigHealth{}
	dch7.SetId("notifierId")
	dch7.SetName("notifierName in config map test-handler-2")
	dch7.SetResourceType(storage.DeclarativeConfigHealth_NOTIFIER)
	dch7.SetResourceName("notifierName")
	dch7.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch7.SetErrorMessage("")
	dch8 := &storage.DeclarativeConfigHealth{}
	dch8.SetId("m2m-config-id")
	dch8.SetName("https://kubernetes.default.svc in config map test-handler-2")
	dch8.SetResourceType(storage.DeclarativeConfigHealth_AUTH_MACHINE_TO_MACHINE_CONFIG)
	dch8.SetResourceName("https://kubernetes.default.svc")
	dch8.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch8.SetErrorMessage("")
	gomock.InOrder(
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch2)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch3)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch4)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch5)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch6)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch7)),
		mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch8)),
	)

	// Delete resources should be called in order, ignoring the existing IDs from the previously upserted resources.
	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"m2m-config-id"}).Return(nil, nil),
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
			storage.DeclarativeConfigHealth_builder{
				Id:           "some-id",
				Name:         "Config Map some-config-map",
				ResourceType: storage.DeclarativeConfigHealth_CONFIG_MAP,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "notifierId",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_NOTIFIER,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "group",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_GROUP,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "id-auth-provider",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_AUTH_PROVIDER,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "role",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_ROLE,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "skipping-scope",
				Name:         "",
				ResourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
			}.Build(),
			storage.DeclarativeConfigHealth_builder{
				Id:           "some-non-existent-id",
				Name:         "I should be deleted",
				ResourceType: storage.DeclarativeConfigHealth_GROUP,
			}.Build(),
		}, nil),
		mockHealthDS.EXPECT().RemoveDeclarativeConfig(gomock.Any(), "some-non-existent-id"),
	)

	m := newTestManager(t)
	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
	m.declarativeConfigHealthDS = mockHealthDS

	m.reconcileTransformedMessages(map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []protocompat.Message{
				permissionSet1,
				permissionSet2,
			},
			types.AccessScopeType: []protocompat.Message{
				accessScope,
			},
		},
		"test-handler-2": {
			types.RoleType: []protocompat.Message{
				role,
			},
			types.AuthProviderType: []protocompat.Message{
				authProvider,
			},
			types.GroupType: []protocompat.Message{
				group,
			},
			types.NotifierType: []protocompat.Message{
				notifier,
			},
			types.AuthMachineToMachineConfigType: []protocompat.Message{
				m2mConfig,
			},
		},
	})
}

func TestReconcileTransformedMessages_ErrorPropagatedToReporter(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	mockHealthDS := declarativeConfigHealthMock.NewMockDataStore(controller)

	permissionSet1 := &storage.PermissionSet{}
	permissionSet1.SetName("permission-set-1")
	permissionSet1.SetId("some-id")

	testError := errors.New("test error")
	mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(testError).Times(3)

	dch := &storage.DeclarativeConfigHealth{}
	dch.SetId("some-id")
	dch.SetName("permission-set-1 in config map test-handler-1")
	dch.SetResourceType(storage.DeclarativeConfigHealth_PERMISSION_SET)
	dch.SetResourceName("permission-set-1")
	dch.SetStatus(storage.DeclarativeConfigHealth_UNHEALTHY)
	dch.SetErrorMessage("test error")
	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch))

	mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(supportedTypesCount)

	mockHealthDS.EXPECT().GetDeclarativeConfigs(gomock.Any()).
		Return(nil, nil).Times(1)

	m := newTestManager(t)
	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
	m.declarativeConfigHealthDS = mockHealthDS

	// We need to call this 3 times, only then the error will be propagated to the mockHealthDS.
	for i := 0; i < consecutiveReconciliationErrorThreshold; i++ {
		m.reconcileTransformedMessages(map[string]protoMessagesByType{
			"test-handler-1": {
				types.PermissionSetType: []protocompat.Message{
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

	permissionSet1 := &storage.PermissionSet{}
	permissionSet1.SetName("permission-set-1")
	permissionSet1.SetId("some-id")

	m := newTestManager(t)
	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
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
			types.PermissionSetType: []protocompat.Message{
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

	permissionSet1 := &storage.PermissionSet{}
	permissionSet1.SetName("permission-set-1")
	permissionSet1.SetId("some-id")

	m := newTestManager(t)
	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
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
			types.PermissionSetType: []protocompat.Message{
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

	permissionSet1 := &storage.PermissionSet{}
	permissionSet1.SetName("permission-set-1")
	permissionSet1.SetId("some-id")

	m := newTestManager(t)
	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
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
			types.PermissionSetType: []protocompat.Message{
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

	role := &storage.Role{}
	role.SetName("test-name")
	role.SetDescription("test-description")
	role.SetPermissionSetId("access-scope")
	role.SetAccessScopeId("permission-set")
	transformer.EXPECT().Transform(&declarativeconfig.Role{
		Name:          "test-name",
		Description:   "test-description",
		AccessScope:   "access-scope",
		PermissionSet: "permission-set",
	}).Return(map[reflect.Type][]protocompat.Message{
		types.RoleType: {
			role,
		},
	}, nil)

	mockHealthDS.EXPECT().GetDeclarativeConfig(gomock.Any(), gomock.Any()).Return(nil, false, nil).AnyTimes()

	dch := &storage.DeclarativeConfigHealth{}
	dch.SetId("04a87e34-b568-5e14-90ac-380d25c8689b")
	dch.SetName("test-name in config map my-cool-config-map")
	dch.SetResourceType(storage.DeclarativeConfigHealth_ROLE)
	dch.SetResourceName("test-name")
	dch.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch.SetErrorMessage("")
	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch))

	dch2 := &storage.DeclarativeConfigHealth{}
	dch2.SetId(declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String())
	dch2.SetName("Config Map my-cool-config-map")
	dch2.SetResourceType(storage.DeclarativeConfigHealth_CONFIG_MAP)
	dch2.SetResourceName("my-cool-config-map")
	dch2.SetStatus(storage.DeclarativeConfigHealth_HEALTHY)
	dch2.SetErrorMessage("")
	mockHealthDS.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch2))

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
	dch := &storage.DeclarativeConfigHealth{}
	dch.SetId(declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String())
	dch.SetName("Config Map my-cool-config-map")
	dch.SetResourceType(storage.DeclarativeConfigHealth_CONFIG_MAP)
	dch.SetResourceName("my-cool-config-map")
	dch.SetStatus(storage.DeclarativeConfigHealth_UNHEALTHY)
	dch.SetErrorMessage(fmt.Sprintf("could not unmarshal configuration into any of the supported types [%s]",
		declarativeconfig.SupportedConfigurationTypes()))
	reporter.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch))

	m.UpdateDeclarativeConfigContents("/some/config/dir/to/my-cool-config-map", [][]byte{
		[]byte(`
{"cool": "key", "value": "pairs"}
`),
	})

	// 2. Multiple failures in transformation.

	transformer.EXPECT().Transform(gomock.Any()).Return(nil, errors.New("some-error-happened"))

	dch2 := &storage.DeclarativeConfigHealth{}
	dch2.SetId(declarativeconfig.NewDeclarativeHandlerUUID("my-cool-config-map").String())
	dch2.SetName("Config Map my-cool-config-map")
	dch2.SetResourceType(storage.DeclarativeConfigHealth_CONFIG_MAP)
	dch2.SetResourceName("my-cool-config-map")
	dch2.SetStatus(storage.DeclarativeConfigHealth_UNHEALTHY)
	dch2.SetErrorMessage("during transforming configuration: 1 error occurred:\n\t* some-error-happened\n\n")
	reporter.EXPECT().UpsertDeclarativeConfig(gomock.Any(), matchDeclarativeConfigHealth(dch2))

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

	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)

	err := m.verifyUpdaters()
	assert.NoError(t, err)

	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: nil,
	}
	err = m.verifyUpdaters()
	assert.ErrorIs(t, err, errox.InvariantViolation)

	m.updaters = fillTypeResourceUpdaters(t, mockUpdater)
	m.updaters[types.RoleType] = nil

	err = m.verifyUpdaters()
	assert.ErrorIs(t, err, errox.InvariantViolation)
}

func TestFillResourceUpdaters(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	updaterMap := fillTypeResourceUpdaters(t, mockUpdater)
	expectedLen := len(types.GetSupportedProtobufTypesInProcessingOrder())
	assert.Len(t, updaterMap, expectedLen)
}

func fillTypeResourceUpdaters(
	_ testing.TB,
	resUpdater updater.ResourceUpdater,
) map[reflect.Type]updater.ResourceUpdater {
	return map[reflect.Type]updater.ResourceUpdater{
		types.AccessScopeType:                resUpdater,
		types.AuthMachineToMachineConfigType: resUpdater,
		types.AuthProviderType:               resUpdater,
		types.GroupType:                      resUpdater,
		types.NotifierType:                   resUpdater,
		types.PermissionSetType:              resUpdater,
		types.RoleType:                       resUpdater,
	}
}
