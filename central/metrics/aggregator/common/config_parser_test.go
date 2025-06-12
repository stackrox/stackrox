package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func Test_parseMetricLabels(t *testing.T) {
	config := makeTestMetricLabels(t)
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricLabelExpression(t), labelExpression)
}

func Test_validateMetricName(t *testing.T) {
	tests := map[string]string{
		"good":             "",
		"not good":         `doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`,
		"":                 "empty",
		"abc_defAZ0145609": "",
		"not-good":         `doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`,
	}
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateMetricName(name); err != nil {
				assert.Equal(t, expected, err.Error())
			} else {
				assert.Empty(t, expected)
			}
		})
	}
}

func Test_noLabels(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_Labels{
		"metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{},
		},
		"metric2": {},
		"metric3": nil,
	}
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpression)

	labelExpression, err = parseMetricLabels(nil, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, labelExpression)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetricsConfig_Labels{
		"metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"unknown": nil,
			},
		},
	}
	labelExpression, err := parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "metric1" is not in the list of known labels: [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, labelExpression)

	delete(config, "metric1")
	config["met rick"] = nil
	labelExpression, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: invalid metric name "met rick": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
	assert.Empty(t, labelExpression)

	delete(config, "met rick")
	config["metric1"] = &storage.PrometheusMetricsConfig_Labels{
		Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
			"test": {
				Expression: []*storage.PrometheusMetricsConfig_Labels_Expression_Condition{
					{
						Operator: "smooth",
						Argument: "y",
					},
				},
			},
		},
	}
	labelExpression, err = parseMetricLabels(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: failed to parse a condition for metric "metric1" with label "test": operator in "smoothy" is not one of ["="]`, err.Error())
	assert.Empty(t, labelExpression)
}

func TestParseConfiguration(t *testing.T) {
	t.Run("bad metric name", func(t *testing.T) {
		cfg, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics: map[string]*storage.PrometheusMetricsConfig_Labels{
				" ": nil,
			},
		}, nil, testLabelOrder)

		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: invalid metric name " ": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
		assert.Nil(t, cfg)
	})

	t.Run("bad registry name", func(t *testing.T) {
		cfg, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics: map[string]*storage.PrometheusMetricsConfig_Labels{
				"m1": {
					RegistryName: "bad name",
				},
			},
		}, nil, testLabelOrder)

		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: registry name "bad name" for metric m1 doesn't match "^[a-zA-Z0-9-_]*$"`, err.Error())
		assert.Nil(t, cfg)
	})

	t.Run("test parse sequence", func(t *testing.T) {
		// Good:
		cfg0, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics:                makeTestMetricLabels(t),
			Filter:                 "Cluster:name",
		}, nil, testLabelOrder)

		assert.NoError(t, err)
		assert.Equal(t, "Cluster", cfg0.filter.GetBaseQuery().GetMatchFieldQuery().GetField())
		if assert.NotNil(t, cfg0.metrics) {
			assert.Equal(t, makeTestMetricLabelExpression(t), cfg0.metrics)
		}

		// Bad:
		cfg1, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics: map[string]*storage.PrometheusMetricsConfig_Labels{
				"m1": {
					Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
						"label1": nil,
					},
				},
			},
		}, nil, testLabelOrder)
		assert.ErrorIs(t, err, errInvalidConfiguration)
		assert.Equal(t, `invalid configuration: label "label1" for metric "m1" is not in the list of known labels: [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
		assert.Nil(t, cfg1)

		// Another good:
		cfg2, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics: map[string]*storage.PrometheusMetricsConfig_Labels{
				"m2": {
					Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
						"Cluster": nil,
					},
				},
			},
			Filter: "Namespace:name",
		}, cfg0.metrics, testLabelOrder)

		assert.NoError(t, err)
		assert.Equal(t, "Namespace", cfg2.filter.GetBaseQuery().GetMatchFieldQuery().GetField())
		if assert.NotNil(t, cfg2.metrics) {
			assert.NotNil(t, cfg2.metrics["m2"])
			assert.Nil(t, cfg2.metrics["m1"])
		}
	})

	t.Run("test bad query", func(t *testing.T) {
		cfg, err := ParseConfiguration(&storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics:                makeTestMetricLabels(t),
			Filter:                 "bad query?",
		}, nil, testLabelOrder)

		assert.NoError(t, err)
		if assert.NotNil(t, cfg) {
			assert.Empty(t, cfg.filter.GetBaseQuery().GetMatchFieldQuery().GetField())
		}
	})

	t.Run("change exposure", func(t *testing.T) {
		storageConfig := &storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics:                makeTestMetricLabels(t),
		}
		cfg0, err := ParseConfiguration(storageConfig, nil, testLabelOrder)
		assert.NoError(t, err)

		for _, labels := range storageConfig.Metrics {
			if labels.Exposure == storage.PrometheusMetricsConfig_Labels_BOTH {
				labels.Exposure = storage.PrometheusMetricsConfig_Labels_INTERNAL
			}
		}

		for _, metric := range cfg0.toAdd {
			regCfg := cfg0.metricRegistry[metric]
			if err := metrics.RegisterCustomAggregatedMetric(
				string(metric),
				"test",
				cfg0.period,
				getMetricLabels(cfg0.metrics[metric], testLabelOrder),
				regCfg.registry,
				metrics.Exposure(regCfg.exposure)); err != nil {
				assert.Failf(t, "Failed to register test metric", "%q: %v", metric, err)
			}
		}

		cfg1, err := ParseConfiguration(storageConfig, cfg0.metrics, testLabelOrder)

		assert.ErrorIs(t, err, errInvalidConfiguration, err)
		assert.Nil(t, cfg1)
	})

	t.Run("change labels", func(t *testing.T) {
		storageConfig := &storage.PrometheusMetricsConfig_Metrics{
			GatheringPeriodMinutes: 121,
			Metrics:                makeTestMetricLabels(t),
		}
		cfg0, err := ParseConfiguration(storageConfig, nil, testLabelOrder)
		assert.NoError(t, err)
		for _, config := range storageConfig.Metrics {
			if config.Exposure == storage.PrometheusMetricsConfig_Labels_BOTH {
				config.Labels["CVE"] = &storage.PrometheusMetricsConfig_Labels_Expression{}
			}
		}
		cfg1, err := ParseConfiguration(storageConfig, cfg0.metrics, testLabelOrder)

		assert.ErrorIs(t, err, errInvalidConfiguration, err)
		assert.True(t, strings.Contains(err.Error(), "cannot alter metrics"))
		assert.Nil(t, cfg1)
	})
}
