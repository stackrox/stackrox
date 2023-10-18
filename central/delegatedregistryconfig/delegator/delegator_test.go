package delegator

import (
	"context"
	"errors"
	"fmt"
	"testing"

	deleDSMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	sacHelperMocks "github.com/stackrox/rox/central/role/sachelper/mocks"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	waiterMocks "github.com/stackrox/rox/pkg/waiter/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	ctxBG         = context.Background()
	errBroken     = errors.New("broken")
	fakeClusterID = "fake-cluster-id"
	fakeWaiterID  = "fake-waiter-id"

	fakeRegistry      = "fake-reg"
	fakeImageFullName = fmt.Sprintf("%v/repo/something", fakeRegistry)
	fakeImgName       = &storage.ImageName{
		Remote:   "repo/something",
		FullName: fakeImageFullName,
	}
)

func TestGetDelegateClusterID(t *testing.T) {
	var deleClusterDS *deleDSMocks.MockDataStore
	var connMgr *connMocks.MockManager
	var waiterMgr *waiterMocks.MockManager[*storage.Image]
	var d *delegatorImpl

	fakeConnWithCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithCap.EXPECT().HasCapability(gomock.Any()).Return(true).AnyTimes()

	fakeConnWithoutCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithoutCap.EXPECT().HasCapability(gomock.Any()).Return(false).AnyTimes()

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connMgr = connMocks.NewMockManager(ctrl)
		waiterMgr = waiterMocks.NewMockManager[*storage.Image](ctrl)
		deleClusterDS = deleDSMocks.NewMockDataStore(ctrl)
		d = New(deleClusterDS, connMgr, waiterMgr, nil)
	}

	t.Run("error get config", func(t *testing.T) {
		setup(t)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, errBroken)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.ErrorContains(t, err, errBroken.Error())
	})

	t.Run("nil config", func(t *testing.T) {
		setup(t)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		setup(t)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(&storage.DelegatedRegistryConfig{}, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("none", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_NONE}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all no default cluster id", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_ALL}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no ad-hoc cluster")
	})

	t.Run("all def cluster id nil conn", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no connection")
	})

	t.Run("all def cluster id conn no cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all def cluster id conn with cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all reg cluster id conn no cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_ALL,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all reg cluster id conn with cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_ALL,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg no match", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "fake-path", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg match", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: fakeRegistry, ClusterId: fakeClusterID},
			},
		}

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, fakeImgName)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg match no reg cluster id", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_SPECIFIC,
			DefaultClusterId: fakeClusterID,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: fakeRegistry},
			},
		}

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, fakeImgName)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific multiple regs first match", func(t *testing.T) {
		setup(t)
		tImgName := &storage.ImageName{FullName: "reg/specific"}

		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "reg/specific", ClusterId: "id1"},
				{Path: "reg", ClusterId: "id2"},
			},
		}

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, tImgName)
		assert.Equal(t, "id1", clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})
}

func TestDelegateEnrichImage(t *testing.T) {
	var deleClusterDS *deleDSMocks.MockDataStore
	var namespaceSACHelper *sacHelperMocks.MockClusterNamespaceSacHelper
	var connMgr *connMocks.MockManager
	var waiterMgr *waiterMocks.MockManager[*storage.Image]
	var waiter *waiterMocks.MockWaiter[*storage.Image]

	var d *delegatorImpl

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connMgr = connMocks.NewMockManager(ctrl)
		waiterMgr = waiterMocks.NewMockManager[*storage.Image](ctrl)
		waiter = waiterMocks.NewMockWaiter[*storage.Image](ctrl)
		deleClusterDS = deleDSMocks.NewMockDataStore(ctrl)
		namespaceSACHelper = sacHelperMocks.NewMockClusterNamespaceSacHelper(ctrl)

		waiter.EXPECT().ID().Return(fakeWaiterID).AnyTimes()
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		d = New(deleClusterDS, connMgr, waiterMgr, namespaceSACHelper)
	}

	t.Run("empty cluster id", func(t *testing.T) {
		setup(t)
		image, err := d.DelegateScanImage(ctxBG, nil, "", false)
		assert.ErrorContains(t, err, "cluster id")
		assert.Nil(t, image)
	})

	t.Run("waiter create error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(nil, errBroken)

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("send msg error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any()).Return(errBroken)

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("wait error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(nil, errBroken)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("success round trip", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.Image{
			Name: &storage.ImageName{
				FullName: fakeImageFullName,
			},
		}

		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(fakeImage, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImage(ctxBG, fakeImgName, fakeClusterID, false)
		assert.NoError(t, err)
		assert.Equal(t, fakeImage, image)
	})
}

func TestInferNamespace(t *testing.T) {
	var namespaceSACHelper *sacHelperMocks.MockClusterNamespaceSacHelper

	var d *delegatorImpl

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		namespaceSACHelper = sacHelperMocks.NewMockClusterNamespaceSacHelper(ctrl)

		d = New(nil, nil, nil, namespaceSACHelper)
	}

	t.Run("no namespace on error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID, inferNamespacePermissions).Return(nil, errBroken)

		result := d.inferNamespace(ctxBG, fakeImgName, fakeClusterID)
		assert.Zero(t, result)
	})

	t.Run("no namespace on empty search result", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID, inferNamespacePermissions).Return(nil, nil)

		result := d.inferNamespace(ctxBG, fakeImgName, fakeClusterID)
		assert.Zero(t, result)
	})

	t.Run("namespace found", func(t *testing.T) {
		setup(t)
		namespaces := []*v1.ScopeObject{{Name: "repo"}}
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID, inferNamespacePermissions).Return(namespaces, nil)

		result := d.inferNamespace(ctxBG, fakeImgName, fakeClusterID)
		assert.Equal(t, "repo", result)
	})
}
