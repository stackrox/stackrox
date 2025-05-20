package aggregator

import (
	"context"
	"iter"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	"github.com/stretchr/testify/assert"
)

func Test_run(t *testing.T) {

	t.Run("stop on start", func(t *testing.T) {
		runner := &aggregatorRunner{
			stopCh: make(chan bool, 1),
			vulnerabilities: common.MakeTrackerConfig("test", "test",
				map[common.Label]int{}, nil),
		}
		runner.Stop()
		i := false
		runner.run(runner.vulnerabilities, func(_ string, _ prometheus.Labels, _ int) {
			i = true
		})
		assert.False(t, i)
	})

	t.Run("stop after new period", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		runner.vulnerabilities = common.MakeTrackerConfig("test", "test",
			map[common.Label]int{}, func(context.Context, common.MetricLabelsExpressions) iter.Seq[common.Finding] {
				return func(yield func(common.Finding) bool) {
					i = true
					runner.Stop()
				}
			})
		runner.vulnerabilities.GetPeriodCh() <- time.Minute
		runner.run(runner.vulnerabilities, nil)
		assert.True(t, i)
	})

	t.Run("run a few ticks", func(t *testing.T) {
		i := 0
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		runner.vulnerabilities = common.MakeTrackerConfig("test", "test",
			map[common.Label]int{}, func(context.Context, common.MetricLabelsExpressions) iter.Seq[common.Finding] {
				return func(yield func(common.Finding) bool) {
					i++
					if i > 2 {
						runner.Stop()
					}
				}
			})
		runner.vulnerabilities.GetPeriodCh() <- 100 * time.Microsecond
		runner.run(runner.vulnerabilities, nil)
		assert.Greater(t, i, 2)
	})

	t.Run("stop in runtime", func(t *testing.T) {
		var i atomic.Int32
		runner := &aggregatorRunner{stopCh: make(chan bool, 1)}
		runner.vulnerabilities = common.MakeTrackerConfig("test", "test",
			map[common.Label]int{}, func(context.Context, common.MetricLabelsExpressions) iter.Seq[common.Finding] {
				return func(yield func(common.Finding) bool) {
					i.Add(1)
				}
			})
		const period = 50 * time.Millisecond
		runner.vulnerabilities.GetPeriodCh() <- period
		start := time.Now()
		go runner.run(runner.vulnerabilities, nil)
		runner.vulnerabilities.GetPeriodCh() <- 0
		passed := time.Since(start).Round(time.Millisecond)
		time.Sleep(3 * period) // there should be no ticks during the sleep.
		assert.Greater(t, i.Load(), int32(0))
		assert.LessOrEqual(t, i.Load(), int32(1+passed/period))
	})
}
