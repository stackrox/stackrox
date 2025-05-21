package aggregator

import (
	"context"
	"iter"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stretchr/testify/assert"
)

func Test_run(t *testing.T) {

	t.Run("stop on start", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[any] {
				return func(yield func(any) bool) {
					i = true
				}
			},
			nil,
		)
		runner.Stop()
		runner.run(tracker)
		assert.False(t, i)
	})

	t.Run("stop after new period", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[any] {
				return func(yield func(any) bool) {
					i = true
					runner.Stop()
				}
			},
			nil,
		)
		tracker.GetPeriodCh() <- time.Minute
		runner.run(tracker)
		assert.True(t, i)
	})

	t.Run("run a few ticks", func(t *testing.T) {
		i := 0
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[any] {
				return func(yield func(any) bool) {
					i++
					if i > 2 {
						runner.Stop()
					}
				}
			},
			nil,
		)
		tracker.GetPeriodCh() <- 100 * time.Microsecond
		runner.run(tracker)
		assert.Greater(t, i, 2)
	})

	t.Run("stop in runtime", func(t *testing.T) {
		var i atomic.Int32
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[any] {
				return func(yield func(any) bool) {
					i.Add(1)
				}
			},
			nil,
		)
		const period = 50 * time.Millisecond
		tracker.GetPeriodCh() <- period
		start := time.Now()
		go runner.run(tracker)
		tracker.GetPeriodCh() <- 0
		passed := time.Since(start).Round(time.Millisecond)
		time.Sleep(3 * period) // there should be no ticks during the sleep.
		assert.Greater(t, i.Load(), int32(0))
		assert.LessOrEqual(t, i.Load(), int32(1+passed/period))
	})
}
