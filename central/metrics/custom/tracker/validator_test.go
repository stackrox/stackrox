package tracker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestTranslateConfiguration(t *testing.T) {
	config := makeTestMetricLabels(t)
	testFilters := makeTestLabelFilters(t)

	tracker := MakeTrackerBase("test", "desc", testLabelGetters, nilGatherFunc)
	md, incFilters, excFilters, err := tracker.translateStorageConfiguration(config)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricDescriptors(t), md)
	assert.Empty(t, excFilters)

	// Test that the parsed include filters are equal to the test filters.
	assert.Equal(t, len(testFilters), len(incFilters))
	for metric, filters := range incFilters {
		if !assert.NotNil(t, filters) {
			break
		}
		for label, expr := range filters {
			if !assert.NotNil(t, expr) {
				break
			}
			if !assert.NotNil(t, testFilters[metric], metric) ||
				!assert.NotNil(t, testFilters[metric][label], label) {
				break
			}
			assert.Equal(t, testFilters[metric][label].String(), expr.String())
		}
	}
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
	tracker := MakeTrackerBase("test", "desc", testLabelGetters, nilGatherFunc)

	for _, labels := range []*storage.PrometheusMetrics_Group_Labels{{Labels: []string{}}, {}, nil} {
		config := map[string]*storage.PrometheusMetrics_Group_Labels{
			"metric": labels,
		}
		md, _, _, err := tracker.translateStorageConfiguration(config)
		assert.Equal(t, `invalid configuration: no labels specified for metric "test_metric"`, err.Error())
		assert.Empty(t, md)
	}

	md, _, _, err := tracker.translateStorageConfiguration(nil)
	assert.NoError(t, err)
	assert.Empty(t, md)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetrics_Group_Labels{
		"metric1": {
			Labels: []string{"unknown"},
		},
	}
	tracker := MakeTrackerBase("test", "desc", testLabelGetters, nilGatherFunc)

	md, _, _, err := tracker.translateStorageConfiguration(config)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "test_metric1" is not in the list of known labels [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, md)

	delete(config, "metric1")
	config["met rick"] = nil
	md, _, _, err = tracker.translateStorageConfiguration(config)
	assert.Equal(t, `invalid configuration: invalid metric name "test_met rick": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
	assert.Empty(t, md)

	config = map[string]*storage.PrometheusMetrics_Group_Labels{
		"metric1": {
			Labels: []string{"Namespace"},
			IncludeFilters: map[string]string{
				"filter_unknown_label": "x.*",
			},
		},
	}
	tracker = MakeTrackerBase("test", "desc", testLabelGetters, nilGatherFunc)

	md, _, _, err = tracker.translateStorageConfiguration(config)
	assert.Equal(t, `invalid configuration: label "filter_unknown_label" for metric "test_metric1" is not in the list of known labels [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, md)

	config = map[string]*storage.PrometheusMetrics_Group_Labels{
		"metric1": {
			Labels: []string{"Namespace"},
			IncludeFilters: map[string]string{
				"Namespace": "[1-",
			},
		},
	}
	tracker = MakeTrackerBase("test", "desc", testLabelGetters, nilGatherFunc)

	md, _, _, err = tracker.translateStorageConfiguration(config)
	assert.Equal(t, "invalid configuration: bad include_filter expression for metric \"test_metric1\" label \"Namespace\": error parsing regexp: invalid character class range: `1-$`", err.Error())
	assert.Empty(t, md)

}
