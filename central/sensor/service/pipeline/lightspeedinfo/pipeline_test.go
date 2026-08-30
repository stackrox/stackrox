package lightspeedinfo

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	clusterID string
	info      *central.LightspeedInfo
}

func (m *mockStore) UpdateInfo(clusterID string, info *central.LightspeedInfo) {
	m.clusterID = clusterID
	m.info = info
}

func TestPipeline(t *testing.T) {
	store := &mockStore{}
	p := NewPipeline(store)

	t.Run("Match returns true for LightspeedInfo message", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_LightspeedInfo{
				LightspeedInfo: &central.LightspeedInfo{
					Host:    "https://lightspeed.example.com",
					IsReady: true,
				},
			},
		}
		assert.True(t, p.Match(msg))
	})

	t.Run("Match returns false for other messages", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{},
		}
		assert.False(t, p.Match(msg))
	})

	t.Run("Run stores LightspeedInfo", func(t *testing.T) {
		ctx := context.Background()
		info := &central.LightspeedInfo{
			Host:           "https://lightspeed.example.com",
			IsReady:        true,
			HasQueryAccess: true,
		}
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_LightspeedInfo{
				LightspeedInfo: info,
			},
		}

		err := p.Run(ctx, "cluster1", msg, nil)

		assert.NoError(t, err)
		assert.Equal(t, "cluster1", store.clusterID)
		assert.Equal(t, info, store.info)
	})

	t.Run("OnFinish is no-op", func(t *testing.T) {
		assert.NotPanics(t, func() {
			p.OnFinish("cluster1")
		})
	})

	t.Run("Capabilities returns nil", func(t *testing.T) {
		assert.Nil(t, p.Capabilities())
	})

	t.Run("Reconcile returns nil", func(t *testing.T) {
		err := p.Reconcile(context.Background(), "cluster1", nil)
		assert.NoError(t, err)
	})
}
