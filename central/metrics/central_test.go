package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncrementMsgToSensorSkipCounter(t *testing.T) {
	t.Run("no panic on nil msg", func(t *testing.T) {
		assert.NotPanics(t, func() {
			IncrementMsgToSensorSkipCounter("", nil)
		})
	})

	t.Run("no panic on nil inner msg", func(t *testing.T) {
		assert.NotPanics(t, func() {
			IncrementMsgToSensorSkipCounter("", &central.MsgToSensor{
				Msg: nil,
			})
		})
	})

	t.Run("inc and extract type", func(t *testing.T) {
		// Clear any prior values.
		prometheus.Unregister(msgToSensorSkipCounter)
		require.NoError(t, prometheus.Register(msgToSensorSkipCounter))

		// Get references to the counters.
		updImgCounter, err := msgToSensorSkipCounter.GetMetricWith(
			prometheus.Labels{"ClusterID": "a", "type": "UpdatedImage"},
		)
		require.NoError(t, err)
		reprocessDeployCounter, err := msgToSensorSkipCounter.GetMetricWith(
			prometheus.Labels{"ClusterID": "b", "type": "ReprocessDeployments"},
		)
		require.NoError(t, err)

		// Sanity check.
		assert.Equal(t, 0.0, testutil.ToFloat64(updImgCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(reprocessDeployCounter))

		// Verify the count is incremented and type extracted.
		IncrementMsgToSensorSkipCounter("a", &central.MsgToSensor{
			Msg: &central.MsgToSensor_UpdatedImage{},
		})
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(reprocessDeployCounter))

		IncrementMsgToSensorSkipCounter("b", &central.MsgToSensor{
			Msg: &central.MsgToSensor_ReprocessDeployments{},
		})
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgCounter))
		assert.Equal(t, 1.0, testutil.ToFloat64(reprocessDeployCounter))
	})
}
