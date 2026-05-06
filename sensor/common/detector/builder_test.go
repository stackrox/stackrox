package detector

import (
	"testing"

	sensorInternal "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/expiringcache/mocks"
	"github.com/stackrox/rox/pkg/scopecomp"
	enforcerMocks "github.com/stackrox/rox/sensor/common/enforcer/mocks"
	"github.com/stackrox/rox/sensor/common/image/cache"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/scan"
	storeMocks "github.com/stackrox/rox/sensor/common/store/mocks"
	updaterMocks "github.com/stackrox/rox/sensor/common/updater/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func validBuilder(ctrl *gomock.Controller) *Builder {
	return NewBuilder().
		WithClusterID(fakeClusterID{}).
		WithEnforcer(enforcerMocks.NewMockEnforcer(ctrl)).
		WithDeploymentStore(storeMocks.NewMockDeploymentStore(ctrl)).
		WithServiceAccountStore(storeMocks.NewMockServiceAccountStore(ctrl)).
		WithImageCache(mocks.NewMockCache[cache.Key, cache.Value](ctrl)).
		WithAuditLogEvents(make(chan *sensorInternal.AuditEvents)).
		WithAuditLogUpdater(updaterMocks.NewMockComponent(ctrl)).
		WithNetworkPolicyStore(storeMocks.NewMockNetworkPolicyStore(ctrl)).
		WithRegistryStore(fakeRegistryProvider{}).
		WithLocalScan(&scan.LocalScan{}).
		WithNodeStore(storeMocks.NewMockNodeStore(ctrl)).
		WithClusterLabelProvider(fakeClusterLabelProvider{}).
		WithNamespaceLabelProvider(fakeNamespaceLabelProvider{})
}

type fakeClusterID struct{}

func (fakeClusterID) Get() string       { return "id" }
func (fakeClusterID) GetNoWait() string { return "id" }

type fakeRegistryProvider struct{ registry.Provider }
type fakeClusterLabelProvider struct{ scopecomp.ClusterLabelProvider }
type fakeNamespaceLabelProvider struct {
	scopecomp.NamespaceLabelProvider
}

func TestValidateRequiredFields(t *testing.T) {
	cases := map[string]struct {
		modify func(b *Builder)
		errMsg string
	}{
		"all required fields set": {
			modify: func(_ *Builder) {},
		},
		"missing ClusterID": {
			modify: func(b *Builder) { b.clusterID = nil },
			errMsg: "ClusterID is required",
		},
		"missing Enforcer": {
			modify: func(b *Builder) { b.enforcer = nil },
			errMsg: "Enforcer is required",
		},
		"missing DeploymentStore": {
			modify: func(b *Builder) { b.deploymentStore = nil },
			errMsg: "DeploymentStore is required",
		},
		"missing ServiceAccountStore": {
			modify: func(b *Builder) { b.serviceAccountStore = nil },
			errMsg: "ServiceAccountStore is required",
		},
		"missing ImageCache": {
			modify: func(b *Builder) { b.imageCache = nil },
			errMsg: "ImageCache is required",
		},
		"missing AuditLogEvents (nil channel)": {
			modify: func(b *Builder) { b.auditLogEvents = nil },
			errMsg: "AuditLogEvents is required",
		},
		"missing AuditLogUpdater": {
			modify: func(b *Builder) { b.auditLogUpdater = nil },
			errMsg: "AuditLogUpdater is required",
		},
		"missing NetworkPolicyStore": {
			modify: func(b *Builder) { b.networkPolicyStore = nil },
			errMsg: "NetworkPolicyStore is required",
		},
		"missing RegistryStore": {
			modify: func(b *Builder) { b.registryStore = nil },
			errMsg: "RegistryStore is required",
		},
		"missing LocalScan (nil pointer)": {
			modify: func(b *Builder) { b.localScan = nil },
			errMsg: "LocalScan is required",
		},
		"missing NodeStore": {
			modify: func(b *Builder) { b.nodeStore = nil },
			errMsg: "NodeStore is required",
		},
		"missing ClusterLabelProvider": {
			modify: func(b *Builder) { b.clusterLabelProvider = nil },
			errMsg: "ClusterLabelProvider is required",
		},
		"missing NamespaceLabelProvider": {
			modify: func(b *Builder) { b.namespaceLabelProvider = nil },
			errMsg: "NamespaceLabelProvider is required",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			b := validBuilder(ctrl)
			tc.modify(b)
			err := b.validate()
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

// Verify optional fields don't cause validation errors when nil.
func TestValidateOptionalFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	b := validBuilder(ctrl)
	b.admCtrlSettingsMgr = nil
	b.factSettingsMgr = nil
	require.NoError(t, b.validate())
}
