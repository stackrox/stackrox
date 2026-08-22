package lightspeedquery

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
)

type mockBroker struct {
	resp *central.LightspeedQueryResponse
}

func (m *mockBroker) NotifyResponseReceived(resp *central.LightspeedQueryResponse) {
	m.resp = resp
}

func TestPipeline(t *testing.T) {
	broker := &mockBroker{}
	p := NewPipeline(broker)

	t.Run("Match returns true for LightspeedQueryResponse message", func(t *testing.T) {
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_LightspeedQueryResponse{
				LightspeedQueryResponse: &central.LightspeedQueryResponse{
					Id:      "test-id",
					Summary: "test summary",
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

	t.Run("Run notifies broker of response", func(t *testing.T) {
		ctx := context.Background()
		resp := &central.LightspeedQueryResponse{
			Id:      "test-id",
			Summary: "test summary",
		}
		msg := &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_LightspeedQueryResponse{
				LightspeedQueryResponse: resp,
			},
		}

		err := p.Run(ctx, "cluster1", msg, nil)

		assert.NoError(t, err)
		assert.Equal(t, resp, broker.resp)
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
