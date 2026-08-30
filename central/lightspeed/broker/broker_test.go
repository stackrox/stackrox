package broker

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBroker_NotifyResponseReceived(t *testing.T) {
	b := New()

	t.Run("notify with valid response", func(t *testing.T) {
		sig := &querySignal{
			requestTime: time.Now(),
			arrived:     concurrency.NewSignal(),
		}
		b.active["test-id"] = sig

		resp := &central.LightspeedQueryResponse{
			Id:      "test-id",
			Summary: "test summary",
		}

		go b.NotifyResponseReceived(resp)

		select {
		case <-sig.arrived.Done():
			assert.Equal(t, resp, sig.resp)
			_, exists := b.active["test-id"]
			assert.False(t, exists, "active request should be removed")
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("notify with unknown ID logs warning", func(t *testing.T) {
		resp := &central.LightspeedQueryResponse{
			Id:      "unknown-id",
			Summary: "test summary",
		}

		// Should not panic
		b.NotifyResponseReceived(resp)
	})

	t.Run("notify with nil response logs warning", func(t *testing.T) {
		// Should not panic
		b.NotifyResponseReceived(nil)
	})
}

func TestBroker_SendAndWaitForSummary(t *testing.T) {
	ctx := context.Background()

	t.Run("successful query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := mocks.NewMockSensorConnection(ctrl)
		conn.EXPECT().ClusterID().Return("cluster1").AnyTimes()
		conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, msg *central.MsgToSensor) error {
				req := msg.GetLightspeedQueryRequest()
				require.NotNil(t, req)
				assert.Equal(t, "test query", req.GetQuery())
				assert.Equal(t, "{}", req.GetContextJson())
				return nil
			},
		)

		b := New()

		done := make(chan struct{})
		go func() {
			defer close(done)
			// Simulate response after short delay
			time.Sleep(50 * time.Millisecond)

			// Extract the request ID from active requests
			b.mu.Lock()
			var reqID string
			for id := range b.active {
				reqID = id
				break
			}
			b.mu.Unlock()

			if reqID != "" {
				b.NotifyResponseReceived(&central.LightspeedQueryResponse{
					Id:      reqID,
					Summary: "test summary",
				})
			}
		}()

		summary, err := b.SendAndWaitForSummary(ctx, conn, "test query", "{}", 5*time.Second)
		<-done

		require.NoError(t, err)
		assert.Equal(t, "test summary", summary)
	})

	t.Run("query with error response", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := mocks.NewMockSensorConnection(ctrl)
		conn.EXPECT().ClusterID().Return("cluster1").AnyTimes()
		conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Return(nil)

		b := New()

		done := make(chan struct{})
		go func() {
			defer close(done)
			time.Sleep(50 * time.Millisecond)

			b.mu.Lock()
			var reqID string
			for id := range b.active {
				reqID = id
				break
			}
			b.mu.Unlock()

			if reqID != "" {
				b.NotifyResponseReceived(&central.LightspeedQueryResponse{
					Id:    reqID,
					Error: "query failed",
				})
			}
		}()

		_, err := b.SendAndWaitForSummary(ctx, conn, "test query", "{}", 5*time.Second)
		<-done

		require.Error(t, err)
		assert.Contains(t, err.Error(), "query failed")
	})

	t.Run("timeout waiting for response", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := mocks.NewMockSensorConnection(ctrl)
		conn.EXPECT().ClusterID().Return("cluster1").AnyTimes()
		conn.EXPECT().InjectMessage(gomock.Any(), gomock.Any()).Return(nil)

		b := New()

		_, err := b.SendAndWaitForSummary(ctx, conn, "test query", "{}", 100*time.Millisecond)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")

		// Verify active request was cleaned up
		assert.Empty(t, b.active)
	})
}

func TestSingleton(t *testing.T) {
	b1 := Singleton()
	b2 := Singleton()

	assert.Same(t, b1, b2, "Singleton should return the same instance")
}
