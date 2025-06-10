package aggregator

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/stackrox/rox/central/metrics/aggregator/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func getNonEmptyStorageCfg() map[string]*storage.PrometheusMetricsConfig_Labels {
	return map[string]*storage.PrometheusMetricsConfig_Labels{
		"metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"label1": {},
			},
		},
	}
}

type something int

func (something) Count() int { return 1 }

var testGetters = []common.LabelGetter[something]{
	{Label: "label1", Getter: func(f something) string { return "value1" }},
}

func Test_run(t *testing.T) {

	t.Run("stop on start", func(t *testing.T) {
		i := 0
		runner := &aggregatorRunner{}
		runner.ctx, runner.cancel = context.WithCancel(context.Background())

		tracker := common.MakeTrackerConfig("test", "test",
			testGetters,
			func(context.Context, *v1.Query, common.MetricsConfiguration) iter.Seq[something] {
				return func(yield func(something) bool) {
					i++
				}
			},
			nil,
		)
		assert.NoError(t, tracker.Reconfigure(runner.ctx, "", getNonEmptyStorageCfg(), 10*time.Minute))
		runner.Stop()
		tracker.Run(runner.ctx)
		assert.Equal(t, 2, i)
	})

	t.Run("stop after new period", func(t *testing.T) {
		i := false
		runner := &aggregatorRunner{}
		runner.ctx, runner.cancel = context.WithCancel(context.Background())

		tracker := common.MakeTrackerConfig("test", "test",
			testGetters,
			func(context.Context, *v1.Query, common.MetricsConfiguration) iter.Seq[something] {
				return func(yield func(something) bool) {
					i = true
					runner.Stop()
				}
			},
			nil,
		)
		assert.NoError(t, tracker.Reconfigure(runner.ctx, "", getNonEmptyStorageCfg(), time.Minute))
		assert.NoError(t, tracker.Reconfigure(runner.ctx, "", getNonEmptyStorageCfg(), time.Minute))
		tracker.Run(runner.ctx)
		assert.True(t, i)
	})

	t.Run("run a few ticks", func(t *testing.T) {
		i := 0
		runner := &aggregatorRunner{}
		runner.ctx, runner.cancel = context.WithCancel(context.Background())

		tracker := common.MakeTrackerConfig("test", "test",
			testGetters,
			func(context.Context, *v1.Query, common.MetricsConfiguration) iter.Seq[something] {
				return func(yield func(something) bool) {
					i++
					if i > 2 {
						runner.Stop()
					}
				}
			},
			nil,
		)

		assert.NoError(t, tracker.Reconfigure(runner.ctx, "", getNonEmptyStorageCfg(), 100*time.Microsecond))
		tracker.Run(runner.ctx)
		assert.Greater(t, i, 2)
	})
}
