package aggregator

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	mockConfigDS "github.com/stackrox/rox/central/config/datastore/mocks"
	mockDeploymentDS "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/metrics/aggregator/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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

func Test_makeRunner(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Run("config DS error", func(t *testing.T) {
		cds := mockConfigDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(nil, errors.New("DB error"))
		dds := mockDeploymentDS.NewMockDataStore(ctrl)
		r := makeRunner(cds, dds)
		assert.NotNil(t, r)
		assert.Nil(t, r.image_vulnerabilities)
	})

	t.Run("valid config", func(t *testing.T) {
		privateConfig := &storage.PrivateConfig{
			PrometheusMetricsConfig: &storage.PrometheusMetricsConfig{
				ImageVulnerabilities: &storage.PrometheusMetricsConfig_Vulnerabilities{
					Filter:                 "",
					Metrics:                getNonEmptyStorageCfg(),
					GatheringPeriodMinutes: 1,
				},
			},
		}
		cds := mockConfigDS.NewMockDataStore(ctrl)
		cds.EXPECT().GetPrivateConfig(gomock.Any()).Times(1).Return(privateConfig, nil)
		dds := mockDeploymentDS.NewMockDataStore(ctrl)

		r := makeRunner(cds, dds)
		assert.NotNil(t, r, "Expected makeRunner to return a runner")
		assert.NotNil(t, r.registry)
		assert.NotNil(t, r.image_vulnerabilities)
	})
}

func Test_run(t *testing.T) {

	testRegistry := prometheus.NewRegistry()

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
		assert.NoError(t, tracker.Reconfigure(runner.ctx, testRegistry, "", getNonEmptyStorageCfg(), 10*time.Minute))
		runner.Stop()
		tracker.Run(runner.ctx)
		assert.Equal(t, 1, i)
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
		assert.NoError(t, tracker.Reconfigure(runner.ctx, testRegistry, "", getNonEmptyStorageCfg(), time.Minute))
		assert.NoError(t, tracker.Reconfigure(runner.ctx, testRegistry, "", getNonEmptyStorageCfg(), time.Minute))
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

		assert.NoError(t, tracker.Reconfigure(runner.ctx, testRegistry, "", getNonEmptyStorageCfg(), 100*time.Microsecond))
		tracker.Run(runner.ctx)
		assert.Greater(t, i, 2)
	})
}
