package aggregator

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stackrox/rox/central/metrics"
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

func Test_getRegistryName(t *testing.T) {
	metrics.GetExternalRegistry("r1")
	metrics.GetExternalRegistry("r2")

	runner := &aggregatorRunner{}

	for path, expected := range map[string]string{
		"/metrics/r1":     "r1",
		"/metrics/r2":     "r2",
		"/metrics/r1?a=b": "r1",
		"/metrics/r2?a=b": "r2",
		"/metrics":        "",
	} {
		u, _ := url.Parse("https://central" + path)
		name, ok := runner.getRegistryName(&http.Request{URL: u})
		assert.True(t, ok)
		assert.Equal(t, expected, name)
	}

	for _, path := range []string{
		"",
		"/r1",
		"/r1/",
		"/metrics/r1/",
		"/metricsr1",
		"/metricsr1/",
		"/metricsr1/r1",
		"/metrics/bad",
		"/kilometrics/r1",
	} {
		u, _ := url.Parse("https://central" + path)
		name, ok := runner.getRegistryName(&http.Request{URL: u})
		assert.False(t, ok)
		assert.Empty(t, name)
	}
}
