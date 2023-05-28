package delegator

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	deleDSMocks "github.com/stackrox/rox/central/delegatedregistryconfig/datastore/mocks"
	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	waiterMocks "github.com/stackrox/rox/pkg/waiter/mocks"
	"github.com/stretchr/testify/assert"
)

var (
	ctxBG         = context.Background()
	errBroken     = errors.New("broken")
	fakeClusterID = "fake-cluster-id"
	fakeWaiterID  = "fake-waiter-id"

	fakeRegistry      = "fake-reg"
	fakeImageFullName = fmt.Sprintf("%v/repo/something", fakeRegistry)
	fakeImage         = &storage.Image{
		Name: &storage.ImageName{
			FullName: fakeImageFullName,
		},
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
		d = New(deleClusterDS, connMgr, waiterMgr)
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
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all no default cluster id", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_ALL}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
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
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no connection")
	})

	t.Run("all def cluster id conn no cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all def cluster id conn with cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
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
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all reg cluster id conn no cap", func(t *testing.T) {
		setup(t)
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_ALL,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
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
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
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

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(ctxBG, fakeImage)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})
}

func TestDelegateEnrichImage(t *testing.T) {
	var deleClusterDS *deleDSMocks.MockDataStore
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

		waiter.EXPECT().ID().Return(fakeWaiterID).AnyTimes()
		d = New(deleClusterDS, connMgr, waiterMgr)
	}

	t.Run("empty cluster id", func(t *testing.T) {
		setup(t)
		err := d.DelegateEnrichImage(ctxBG, nil, "", false)
		assert.ErrorContains(t, err, "cluster id")
	})

	t.Run("waiter create error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(nil, errBroken)

		err := d.DelegateEnrichImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("send msg error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Close()
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any()).Return(errBroken)

		err := d.DelegateEnrichImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("wait error", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(nil, errBroken)
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		err := d.DelegateEnrichImage(ctxBG, nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("success round trip", func(t *testing.T) {
		setup(t)
		waiterMgr.EXPECT().NewWaiter().Return(waiter, nil)
		waiter.EXPECT().Wait(gomock.Any()).Return(fakeImage, nil)
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())

		image := &storage.Image{}
		err := d.DelegateEnrichImage(ctxBG, image, fakeClusterID, false)
		assert.NoError(t, err)

		// Ensure the address of image hasn't change and does not match what the waiter returned.
		// The enrichment logic expects the values of the input image to be modified in place.
		assert.NotEqual(t, fmt.Sprintf("%p", image), fmt.Sprintf("%p", fakeImage))

		// Ensure that value(s) from the fake image returned by the waiter were set on the input image.
		assert.Equal(t, fakeImage.GetName().GetFullName(), image.GetName().GetFullName())
	})
}
