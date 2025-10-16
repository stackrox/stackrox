package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncrementMsgToSensorNotSentCounter(t *testing.T) {
	t.Run("no panic on nil msg", func(t *testing.T) {
		assert.NotPanics(t, func() {
			IncrementMsgToSensorNotSentCounter("", nil, "")
		})
	})

	t.Run("no panic on nil inner msg", func(t *testing.T) {
		assert.NotPanics(t, func() {
			mts := &central.MsgToSensor{}
			mts.ClearMsg()
			IncrementMsgToSensorNotSentCounter("", mts, "")
		})
	})

	t.Run("inc and extract type", func(t *testing.T) {
		// Clear any prior values.
		prometheus.Unregister(msgToSensorNotSentCounter)
		require.NoError(t, prometheus.Register(msgToSensorNotSentCounter))

		// Get references to the counters.
		updImgErrCounter, err := msgToSensorNotSentCounter.GetMetricWith(
			prometheus.Labels{"ClusterID": "a", "type": "UpdatedImage", "reason": NotSentError},
		)
		require.NoError(t, err)
		updImgSkipCounter, err := msgToSensorNotSentCounter.GetMetricWith(
			prometheus.Labels{"ClusterID": "a", "type": "UpdatedImage", "reason": NotSentSkip},
		)
		require.NoError(t, err)
		reprocessDeploySignalCounter, err := msgToSensorNotSentCounter.GetMetricWith(
			prometheus.Labels{"ClusterID": "b", "type": "ReprocessDeployments", "reason": NotSentSignal},
		)
		require.NoError(t, err)

		// Sanity check.
		assert.Equal(t, 0.0, testutil.ToFloat64(updImgErrCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(updImgSkipCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(reprocessDeploySignalCounter))

		// Verify the count is incremented, type extracted, and reason recorded.
		mts := &central.MsgToSensor{}
		mts.Msg = &central.MsgToSensor_UpdatedImage{}
		IncrementMsgToSensorNotSentCounter("a", mts, NotSentError)
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgErrCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(updImgSkipCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(reprocessDeploySignalCounter))

		mts2 := &central.MsgToSensor{}
		mts2.Msg = &central.MsgToSensor_UpdatedImage{}
		IncrementMsgToSensorNotSentCounter("a", mts2, NotSentSkip)
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgErrCounter))
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgSkipCounter))
		assert.Equal(t, 0.0, testutil.ToFloat64(reprocessDeploySignalCounter))

		mts3 := &central.MsgToSensor{}
		mts3.Msg = &central.MsgToSensor_ReprocessDeployments{}
		IncrementMsgToSensorNotSentCounter("b", mts3, NotSentSignal)
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgErrCounter))
		assert.Equal(t, 1.0, testutil.ToFloat64(updImgSkipCounter))
		assert.Equal(t, 1.0, testutil.ToFloat64(reprocessDeploySignalCounter))
	})
}
