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
	errBroken     = errors.New("broken")
	fakeClusterID = "fake-cluster-id"
	// fakeConnWithCap    = &fakeSensorConn{hasCap: true}
	// fakeConnWithoutCap = &fakeSensorConn{hasCap: false}
	fakeWaiterID = "fake-waiter-id"

	fakeRegistry      = "fake-reg"
	fakeImageFullName = fmt.Sprintf("%v/repo/something", fakeRegistry)
	fakeImage         = &storage.Image{
		Name: &storage.ImageName{
			FullName: fakeImageFullName,
		},
	}
)

func TestGetDelegateClusterID(t *testing.T) {
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMocks.NewMockManager(gomock.NewController(t))
	waiterMgr := waiterMocks.NewMockManager[*storage.Image](gomock.NewController(t))

	fakeConnWithCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithCap.EXPECT().HasCapability(gomock.Any()).Return(true).AnyTimes()

	fakeConnWithoutCap := connMocks.NewMockSensorConnection(gomock.NewController(t))
	fakeConnWithoutCap.EXPECT().HasCapability(gomock.Any()).Return(false).AnyTimes()

	d := New(deleClusterDS, connMgr, waiterMgr)

	t.Run("error get config", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, errBroken)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.ErrorContains(t, err, errBroken.Error())
	})

	t.Run("nil config", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(nil, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(&storage.DelegatedRegistryConfig{}, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("none", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_NONE}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all no default cluster id", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_ALL}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no ad-hoc cluster")
	})

	t.Run("all def cluster id nil conn", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "no connection")
	})

	t.Run("all def cluster id conn no cap", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all def cluster id conn with cap", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor:       storage.DelegatedRegistryConfig_ALL,
			DefaultClusterId: fakeClusterID,
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("all reg cluster id conn no cap", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_ALL,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithoutCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.ErrorContains(t, err, "does not support")
	})

	t.Run("all reg cluster id conn no cap", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_ALL,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg no match", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "fake-path", ClusterId: fakeClusterID},
			},
		}
		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), nil)
		assert.Empty(t, clusterID)
		assert.False(t, shouldDelegate)
		assert.Nil(t, err)
	})

	t.Run("specific reg match", func(t *testing.T) {
		config := &storage.DelegatedRegistryConfig{
			EnabledFor: storage.DelegatedRegistryConfig_SPECIFIC,
			Registries: []*storage.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: fakeRegistry, ClusterId: fakeClusterID},
			},
		}

		deleClusterDS.EXPECT().GetConfig(gomock.Any()).Return(config, false, nil)
		connMgr.EXPECT().GetConnection(gomock.Any()).Return(fakeConnWithCap)
		clusterID, shouldDelegate, err := d.GetDelegateClusterID(context.Background(), fakeImage)
		assert.Equal(t, fakeClusterID, clusterID)
		assert.True(t, shouldDelegate)
		assert.Nil(t, err)
	})
}

func TestDelegateEnrichImage(t *testing.T) {
	deleClusterDS := deleDSMocks.NewMockDataStore(gomock.NewController(t))
	connMgr := connMocks.NewMockManager(gomock.NewController(t))
	waiterMgr := waiterMocks.NewMockManager[*storage.Image](gomock.NewController(t))
	fakeWaiter := waiterMocks.NewMockWaiter[*storage.Image](gomock.NewController(t))
	fakeWaiter.EXPECT().ID().Return(fakeWaiterID).AnyTimes()

	d := New(deleClusterDS, connMgr, waiterMgr)

	t.Run("empty cluster id", func(t *testing.T) {
		err := d.DelegateEnrichImage(context.Background(), nil, "", false)
		assert.ErrorContains(t, err, "cluster id")
	})

	t.Run("waiter create error", func(t *testing.T) {
		waiterMgr.EXPECT().NewWaiter().Return(nil, errBroken)
		err := d.DelegateEnrichImage(context.Background(), nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("send msg error", func(t *testing.T) {
		fakeWaiter.EXPECT().Close()
		waiterMgr.EXPECT().NewWaiter().Return(fakeWaiter, nil)
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any()).Return(errBroken)
		err := d.DelegateEnrichImage(context.Background(), nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("wait error", func(t *testing.T) {
		fakeWaiter.EXPECT().Wait(gomock.Any()).Return(nil, errBroken)
		waiterMgr.EXPECT().NewWaiter().Return(fakeWaiter, nil)
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())
		err := d.DelegateEnrichImage(context.Background(), nil, fakeClusterID, false)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("success round trip", func(t *testing.T) {
		fakeWaiter.EXPECT().Wait(gomock.Any()).Return(fakeImage, nil)
		waiterMgr.EXPECT().NewWaiter().Return(fakeWaiter, nil)
		connMgr.EXPECT().SendMessage(fakeClusterID, gomock.Any())
		image := &storage.Image{}
		err := d.DelegateEnrichImage(context.Background(), image, fakeClusterID, false)
		assert.NoError(t, err)

		// Ensure the address of image hasn't change and does not match what the waiter returned.
		// The enrichment logic expects the values of the input image to be modified in place
		assert.NotEqual(t, fmt.Sprintf("%p", image), fmt.Sprintf("%p", fakeImage))

		// Ensure that values from the fake image returned by the waiter were set on the input image
		assert.Equal(t, fakeImage.GetName().GetFullName(), image.GetName().GetFullName())
	})
}
