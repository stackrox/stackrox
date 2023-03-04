package declarativeconfig

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/mocks"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	updaterMocks "github.com/stackrox/rox/central/declarativeconfig/updater/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stretchr/testify/assert"
)

func TestReconcileTransformedMessages_Success(t *testing.T) {
	controller := gomock.NewController(t)
	mockUpdater := updaterMocks.NewMockResourceUpdater(controller)
	reporter := mocks.NewMockReconciliationErrorReporter(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
	}
	permissionSet2 := &storage.PermissionSet{
		Name: "permission-set-2",
	}
	accessScope := &storage.SimpleAccessScope{
		Name: "accessScope",
	}
	role := &storage.Role{
		Name: "role",
	}
	authProvider := &storage.AuthProvider{
		Name: "authProvider",
	}
	group := &storage.Group{
		Props: &storage.GroupProperties{
			Id: "group",
		},
	}

	mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1)
	mockUpdater.EXPECT().Upsert(gomock.Any(), permissionSet2)
	mockUpdater.EXPECT().Upsert(gomock.Any(), accessScope)
	mockUpdater.EXPECT().Upsert(gomock.Any(), role)
	mockUpdater.EXPECT().Upsert(gomock.Any(), authProvider)
	mockUpdater.EXPECT().Upsert(gomock.Any(), group)

	m := managerImpl{
		updaters: map[reflect.Type]updater.ResourceUpdater{
			types.PermissionSetType: mockUpdater,
			types.AccessScopeType:   mockUpdater,
			types.RoleType:          mockUpdater,
			types.AuthProviderType:  mockUpdater,
			types.GroupType:         mockUpdater,
		},
		reconciliationErrorReporter: reporter,
	}
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
	permissionSetUpdater := updaterMocks.NewMockResourceUpdater(controller)
	reporter := mocks.NewMockReconciliationErrorReporter(controller)

	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
	}

	testError := errors.New("test error")
	permissionSetUpdater.EXPECT().Upsert(gomock.Any(), permissionSet1).Return(testError)
	reporter.EXPECT().ProcessError(permissionSet1, testError)

	m := managerImpl{
		updaters: map[reflect.Type]updater.ResourceUpdater{
			types.PermissionSetType: permissionSetUpdater,
		},
		reconciliationErrorReporter: reporter,
	}
	m.reconcileTransformedMessages(map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
			},
		},
	})
}

func TestReconcileTransformedMessages_MissingUpdaterCausesPanic_DevBuild(t *testing.T) {
	if buildinfo.ReleaseBuild {
		t.SkipNow()
	}

	controller := gomock.NewController(t)
	reporter := mocks.NewMockReconciliationErrorReporter(controller)
	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
	}
	reporter.EXPECT().ProcessError(permissionSet1, gomock.Any())

	m := managerImpl{
		updaters:                    map[reflect.Type]updater.ResourceUpdater{},
		reconciliationErrorReporter: reporter,
	}
	assert.Panics(t, func() {
		m.reconcileTransformedMessages(map[string]protoMessagesByType{
			"test-handler-1": {
				types.PermissionSetType: []proto.Message{
					permissionSet1,
				},
			},
		})
	})
}

func TestReconcileTransformedMessages_MissingUpdaterStopsManager_ReleaseBuild(t *testing.T) {
	if !buildinfo.ReleaseBuild {
		t.SkipNow()
	}

	controller := gomock.NewController(t)
	reporter := mocks.NewMockReconciliationErrorReporter(controller)
	permissionSet1 := &storage.PermissionSet{
		Name: "permission-set-1",
	}
	reporter.EXPECT().ProcessError(permissionSet1, gomock.Any())

	m := managerImpl{
		updaters:                    map[reflect.Type]updater.ResourceUpdater{},
		reconciliationErrorReporter: reporter,
	}
	m.reconcileTransformedMessages(map[string]protoMessagesByType{
		"test-handler-1": {
			types.PermissionSetType: []proto.Message{
				permissionSet1,
			},
		},
	})
	assert.True(t, m.stopSignal.IsDone())
}
