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

func getNonEmptyMLE() common.MetricLabelsExpressions {
	return common.MetricLabelsExpressions{
		"metric1": map[common.Label][]*common.Expression{
			"label1": nil,
		},
	}
}

type something int

func (something) Count() int { return 1 }

func Test_run(t *testing.T) {

	t.Run("stop on start", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[something] {
				return func(yield func(something) bool) {
					i = true
				}
			},
			nil,
		)
		tracker.SetMetricLabelExpressions(getNonEmptyMLE())
		runner.Stop()
		runner.run(tracker)
		assert.False(t, i)
	})

	t.Run("stop after new period", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[something] {
				return func(yield func(something) bool) {
					i = true
					runner.Stop()
				}
			},
			nil,
		)
		tracker.SetMetricLabelExpressions(getNonEmptyMLE())
		tracker.GetPeriodCh() <- time.Minute
		runner.run(tracker)
		assert.True(t, i)
	})

	t.Run("run a few ticks", func(t *testing.T) {
		i := 0
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[something] {
				return func(yield func(something) bool) {
					i++
					if i > 2 {
						runner.Stop()
					}
				}
			},
			nil,
		)
		tracker.SetMetricLabelExpressions(getNonEmptyMLE())
		tracker.GetPeriodCh() <- 100 * time.Microsecond
		runner.run(tracker)
		assert.Greater(t, i, 2)
	})

	t.Run("stop in runtime", func(t *testing.T) {
		var i atomic.Int32
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		tracker := common.MakeTrackerConfig("test", "test",
			nil,
			func(context.Context, common.MetricLabelsExpressions) iter.Seq[something] {
				return func(yield func(something) bool) {
					i.Add(1)
				}
			},
			nil,
		)
		tracker.SetMetricLabelExpressions(getNonEmptyMLE())
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
