package declarativeconfig

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	updaterMocks "github.com/stackrox/rox/central/declarativeconfig/updater/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	transformMocks "github.com/stackrox/rox/pkg/declarativeconfig/transform/mocks"
	"github.com/stackrox/rox/pkg/errox"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *managerImpl {
	m := New(100*time.Millisecond, 100*time.Millisecond, map[reflect.Type]updater.ResourceUpdater{},
		nil, types.UniversalNameExtractor(), types.UniversalIDExtractor())

	mImpl, ok := m.(*managerImpl)
	require.True(t, ok)
	return mImpl
}

// Custom gomock.Matcher for storage.IntegrationHealth that ignores the timestamp field's value, but instead only checks
// that its set.
type integrationHealthMatcher struct {
	expected *storage.IntegrationHealth
}

func (i *integrationHealthMatcher) Matches(x interface{}) bool {
	integrationHealth, ok := x.(*storage.IntegrationHealth)

	if !ok {
		return false
	}

	return i.expected.GetId() == integrationHealth.GetId() &&
		i.expected.GetName() == integrationHealth.GetName() &&
		i.expected.GetStatus() == integrationHealth.GetStatus() &&
		i.expected.GetType() == integrationHealth.GetType() &&
		i.expected.GetErrorMessage() == integrationHealth.GetErrorMessage() &&
		integrationHealth.GetLastTimestamp() != nil
}
func (i *integrationHealthMatcher) String() string {
	return fmt.Sprintf("%+v", i.expected)
}

func matchIntegrationHealth(int *storage.IntegrationHealth) gomock.Matcher {
	return &integrationHealthMatcher{
		expected: int,
	}
}

func TestReconcileTransformedMessages_Success(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	reporter := reporterMocks.NewMockReporter(controller)

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

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), accessScope),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1),
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet2),
		mockUpdater.EXPECT().Upsert(gomock.Any(), role),
		mockUpdater.EXPECT().Upsert(gomock.Any(), authProvider),
		mockUpdater.EXPECT().Upsert(gomock.Any(), group),
	)

	gomock.InOrder(
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "id-access-scope",
			Name:         "accessScope in config map test-handler-1",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "id-perm-set-1",
			Name:         "permission-set-1 in config map test-handler-1",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "id-perm-set-2",
			Name:         "permission-set-2 in config map test-handler-1",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "role",
			Name:         "role in config map test-handler-2",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "id-auth-provider",
			Name:         "authProvider in config map test-handler-2",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
		reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
			Id:           "group",
			Name:         "group email:some@example.com:Admin for auth provider ID some-auth-provider in config map test-handler-2",
			Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:       storage.IntegrationHealth_HEALTHY,
			ErrorMessage: "",
		})),
	)

	// Delete resources should be called in order, ignoring the existing IDs from the previously upserted resources.
	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"group"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"id-auth-provider"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"role"}).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.InAnyOrder([]string{"id-perm-set-1", "id-perm-set-2"})).Return(nil, nil),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), []string{"id-access-scope"}).Return([]string{"skipping-scope"}, errors.New("some-error")),
	)

	// We retrieve the integration healths on the deletion, only the non-ignored ID that does not have "Config Map"
	// in its name should be deleted.
	gomock.InOrder(
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).Return([]*storage.IntegrationHealth{
			{
				Id:   "some-id",
				Name: "Config Map some-config-map",
			},
			{
				Id:   "group",
				Name: "",
			},
			{
				Id:   "id-auth-provider",
				Name: "",
			},
			{
				Id:   "role",
				Name: "",
			},
			{
				Id:   "skipping-scope",
				Name: "",
			},
			{
				Id:   "some-non-existent-id",
				Name: "I should be deleted",
			},
		}, nil),
		reporter.EXPECT().RemoveIntegrationHealth("some-non-existent-id"),
	)

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.RoleType:          mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.GroupType:         mockUpdater,
	}
	m.declarativeConfigErrorReporter = reporter

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
		},
	})
}

func TestReconcileTransformedMessages_ErrorPropagatedToReporter(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	reporter := reporterMocks.NewMockReporter(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
		Id:   "some-id",
	}

	testError := errors.New("test error")
	mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(testError).Times(3)

	reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
		Id:           "some-id",
		Name:         "permission-set-1 in config map test-handler-1",
		Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:       storage.IntegrationHealth_UNHEALTHY,
		ErrorMessage: "test error",
	}))

	mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(5)

	reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
		Return(nil, nil).Times(1)

	m := newTestManager(t)
	m.updaters = map[reflect.Type]updater.ResourceUpdater{
		types.PermissionSetType: mockUpdater,
		types.AccessScopeType:   mockUpdater,
		types.GroupType:         mockUpdater,
		types.AuthProviderType:  mockUpdater,
		types.RoleType:          mockUpdater,
	}
	m.declarativeConfigErrorReporter = reporter

	// We need to call this 3 times, only then the error will be propagated to the reporter.
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
	reporter := reporterMocks.NewMockReporter(controller)

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
	}
	m.declarativeConfigErrorReporter = reporter

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(nil).Times(1),
		reporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(5),
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
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
	reporter := reporterMocks.NewMockReporter(controller)

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
	}
	m.declarativeConfigErrorReporter = reporter

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(errors.New("some error")).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(5),
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
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
		reporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())
}

func TestReconcileTransformedMessages_SkipUpsert(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	reporter := reporterMocks.NewMockReporter(controller)

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
	}
	m.declarativeConfigErrorReporter = reporter

	// 1. Run the first reconciliation where the hash is not yet set. Everything should be run (upsert, delete).

	gomock.InOrder(
		mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(nil).Times(1),
		reporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).Times(1),
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error")).Times(5),
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
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
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, errors.New("some error")).Times(5),
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
			Return(nil, nil).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.True(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())

	// 3. Run the reconciliation again. Only deletion should be done, and if successful no deletion error should be indicated.

	gomock.InOrder(
		mockUpdater.EXPECT().DeleteResources(gomock.Any(), gomock.Any()).Return(nil, nil).Times(5),
		reporter.EXPECT().RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG).
			Return(nil, nil).Times(1),
	)

	m.reconcileTransformedMessages(messages)
	assert.False(t, m.lastDeletionFailed.Get())
	assert.False(t, m.lastUpsertFailed.Get())
}

func TestUpdateDeclarativeConfigContents_RegisterHealthStatus(t *testing.T) {
	controller := gomock.NewController(t)
	reporter := reporterMocks.NewMockReporter(controller)
	transformer := transformMocks.NewMockTransformer(controller)

	m := newTestManager(t)
	m.universalTransformer = transformer
	m.declarativeConfigErrorReporter = reporter

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

	reporter.EXPECT().Register("test-name", "test-name in config map my-cool-config-map", storage.IntegrationHealth_DECLARATIVE_CONFIG)

	reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
		Id:           "/some/config/dir/to/my-cool-config-map",
		Name:         "Config Map my-cool-config-map",
		Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:       storage.IntegrationHealth_HEALTHY,
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
	reporter := reporterMocks.NewMockReporter(controller)
	transformer := transformMocks.NewMockTransformer(controller)

	m := newTestManager(t)
	m.universalTransformer = transformer
	m.declarativeConfigErrorReporter = reporter

	// 1. Failure in unmarshalling the file.
	reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
		Id:           "/some/config/dir/to/my-cool-config-map",
		Name:         "Config Map my-cool-config-map",
		Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:       storage.IntegrationHealth_UNHEALTHY,
		ErrorMessage: "unable to unmarshal the configuration: 4 errors occurred:\n\t* yaml: unmarshal errors:\n  line 1: field cool not found in type declarativeconfig.AuthProvider\n  line 2: field value not found in type declarativeconfig.AuthProvider\n\t* yaml: unmarshal errors:\n  line 1: field cool not found in type declarativeconfig.AccessScope\n  line 2: field value not found in type declarativeconfig.AccessScope\n\t* yaml: unmarshal errors:\n  line 1: field cool not found in type declarativeconfig.PermissionSet\n  line 2: field value not found in type declarativeconfig.PermissionSet\n\t* yaml: unmarshal errors:\n  line 1: field cool not found in type declarativeconfig.Role\n  line 2: field value not found in type declarativeconfig.Role\n\n",
	}))

	m.UpdateDeclarativeConfigContents("/some/config/dir/to/my-cool-config-map", [][]byte{
		[]byte(`
{"cool": "key", "value": "pairs"}
`),
	})

	// 2. Multiple failures in transformation.

	transformer.EXPECT().Transform(gomock.Any()).Return(nil, errors.New("some-error-happened"))

	reporter.EXPECT().UpdateIntegrationHealthAsync(matchIntegrationHealth(&storage.IntegrationHealth{
		Id:           "/some/config/dir/to/my-cool-config-map",
		Name:         "Config Map my-cool-config-map",
		Type:         storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:       storage.IntegrationHealth_UNHEALTHY,
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
	}

	err = m.verifyUpdaters()
	assert.ErrorIs(t, err, errox.InvariantViolation)
}
