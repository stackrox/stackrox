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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
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
		d = New(deleClusterDS, connMgr, waiterMgr, nil, nil)
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
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_NONE)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all no default cluster id", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no ad-hoc cluster")
	})

	t.Run("all def cluster id nil conn", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		config.SetDefaultClusterId(fakeClusterID)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no connection")
	})

	t.Run("all def cluster id conn no cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		config.SetDefaultClusterId(fakeClusterID)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all def cluster id conn with cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		config.SetDefaultClusterId(fakeClusterID)
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all reg cluster id conn no cap", func(t *testing.T) {
		setup(t)
		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath("")
		dd.SetClusterId(fakeClusterID)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
		})
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		_, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all reg cluster id conn with cap", func(t *testing.T) {
		setup(t)
		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath("")
		dd.SetClusterId(fakeClusterID)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_ALL)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
		})
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg no match", func(t *testing.T) {
		setup(t)
		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath("fake-path")
		dd.SetClusterId(fakeClusterID)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_SPECIFIC)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
		})
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg match", func(t *testing.T) {
		setup(t)
		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath(fakeRegistry)
		dd.SetClusterId(fakeClusterID)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_SPECIFIC)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
		})

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, fakeImgName)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg match no reg cluster id", func(t *testing.T) {
		setup(t)
		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath(fakeRegistry)
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_SPECIFIC)
		config.SetDefaultClusterId(fakeClusterID)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
		})

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, fakeImgName)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific multiple regs first match", func(t *testing.T) {
		setup(t)
		tImgName := &storage.ImageName{}
		tImgName.SetFullName("reg/specific")

		dd := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd.SetPath("reg/specific")
		dd.SetClusterId("id1")
		dd2 := &storage.DelegatedRegistryConfig_DelegatedRegistry{}
		dd2.SetPath("reg")
		dd2.SetClusterId("id2")
		config := &storage.DelegatedRegistryConfig{}
		config.SetEnabledFor(storage.DelegatedRegistryConfig_SPECIFIC)
		config.SetRegistries([]*storage.DelegatedRegistryConfig_DelegatedRegistry{
			dd,
			dd2,
		})

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, true, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, tImgName)
		assert.Equal(t, "id1", clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})
}

func TestDelegateEnrichImage(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, false)

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
		d = New(deleClusterDS, connMgr, waiterMgr, nil, namespaceSACHelper)
	}

	t.Run("empty cluster id", func(t *testing.T) {
		setup(t)
		image, err := d.DelegateScanImage(ctxBG, nil, "", "", false)
		assert.ErrorContains(t, err, "cluster id")
		assert.Nil(t, image)
	})

	t.Run("no access to namespace", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, "no-access-ns", false)
		assert.ErrorIs(t, err, errox.NotFound)
		assert.Nil(t, image)
	})

	t.Run("waiter create error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(nil, errBroken)

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("send msg error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any()).Return(errBroken)

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("wait error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(nil, errBroken)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImage(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("success round trip", func(t *testing.T) {
		setup(t)
		so := &v1.ScopeObject{}
		so.SetName("repo")
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID,
			inferNamespacePermissions).Return([]*v1.ScopeObject{so}, nil)
		imageName := &storage.ImageName{}
		imageName.SetFullName(fakeImageFullName)
		fakeImage := &storage.Image{}
		fakeImage.SetName(imageName)

		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(fakeImage, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImage(ctxBG, fakeImgName, fakeClusterID, "repo", false)
		assert.NoError(t, err)
		protoassert.Equal(t, fakeImage, image)
	})
}

func TestDelegateScanImageV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

	var deleClusterDS *deleDSMocks.MockDataStore
	var namespaceSACHelper *sacHelperMocks.MockClusterNamespaceSacHelper
	var connMgr *connMocks.MockManager
	var waiterMgr *waiterMocks.MockManager[*storage.ImageV2]
	var waiter *waiterMocks.MockWaiter[*storage.ImageV2]

	var d *delegatorImpl

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connMgr = connMocks.NewMockManager(ctrl)
		waiterMgr = waiterMocks.NewMockManager[*storage.ImageV2](ctrl)
		waiter = waiterMocks.NewMockWaiter[*storage.ImageV2](ctrl)
		deleClusterDS = deleDSMocks.NewMockDataStore(ctrl)
		namespaceSACHelper = sacHelperMocks.NewMockClusterNamespaceSacHelper(ctrl)

		waiter.EXPECT().ID().Return(fakeWaiterID).AnyTimes()
		d = New(deleClusterDS, connMgr, nil, waiterMgr, namespaceSACHelper)
	}

	t.Run("empty cluster id", func(t *testing.T) {
		setup(t)
		image, err := d.DelegateScanImageV2(ctxBG, nil, "", "", false)
		assert.ErrorContains(t, err, "cluster id")
		assert.Nil(t, image)
	})

	t.Run("no access to namespace", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		image, err := d.DelegateScanImageV2(ctxBG, nil, fakeClusterID, "no-access-ns", false)
		assert.ErrorIs(t, err, errox.NotFound)
		assert.Nil(t, image)
	})

	t.Run("waiter create error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(nil, errBroken)

		image, err := d.DelegateScanImageV2(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("send msg error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any()).Return(errBroken)

		image, err := d.DelegateScanImageV2(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("wait error", func(t *testing.T) {
		setup(t)
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(nil, errBroken)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImageV2(ctxBG, nil, fakeClusterID, "", false)
		assert.ErrorIs(t, err, errBroken)
		assert.Nil(t, image)
	})

	t.Run("success round trip", func(t *testing.T) {
		setup(t)
		so := &v1.ScopeObject{}
		so.SetName("repo")
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID,
			inferNamespacePermissions).Return([]*v1.ScopeObject{so}, nil)
		imageName := &storage.ImageName{}
		imageName.SetFullName(fakeImageFullName)
		fakeImage := &storage.ImageV2{}
		fakeImage.SetName(imageName)

		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(fakeImage, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image, err := d.DelegateScanImageV2(ctxBG, fakeImgName, fakeClusterID, "repo", false)
		assert.NoError(t, err)
		protoassert.Equal(t, fakeImage, image)
	})
}

func TestInferNamespace(t *testing.T) {
	var namespaceSACHelper *sacHelperMocks.MockClusterNamespaceSacHelper

	var d *delegatorImpl

	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		namespaceSACHelper = sacHelperMocks.NewMockClusterNamespaceSacHelper(ctrl)

		d = New(nil, nil, nil, nil, namespaceSACHelper)
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
		so := &v1.ScopeObject{}
		so.SetName("repo")
		namespaces := []*v1.ScopeObject{so}
		namespaceSACHelper.EXPECT().GetNamespacesForClusterAndPermissions(ctxBG, fakeClusterID, inferNamespacePermissions).Return(namespaces, nil)

		result := d.inferNamespace(ctxBG, fakeImgName, fakeClusterID)
		assert.Equal(t, "repo", result)
	})
}
