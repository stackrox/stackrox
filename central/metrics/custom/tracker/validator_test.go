package tracker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestTranslateConfiguration(t *testing.T) {
	config := makeTestMetricLabels(t)
	md, err := translateStorageConfiguration(config, "test_", testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricDescriptors(t), md)
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
	for _, labels := range []*storage.PrometheusMetrics_Group_Labels{{Labels: []string{}}, {}, nil} {
		config := map[string]*storage.PrometheusMetrics_Group_Labels{
			"metric": labels,
		}
		md, err := translateStorageConfiguration(config, "test_", testLabelOrder)
		assert.Equal(t, `invalid configuration: no labels specified for metric "test_metric"`, err.Error())
		assert.Empty(t, md)
	}

	md, err := translateStorageConfiguration(nil, "test_", testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, md)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetrics_Group_Labels{
		"metric1": {
			Labels: []string{"unknown"},
		},
	}
	md, err := translateStorageConfiguration(config, "test_", testLabelOrder)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "test_metric1" is not in the list of known labels [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, md)

	delete(config, "metric1")
	config["met rick"] = nil
	md, err = translateStorageConfiguration(config, "test_", testLabelOrder)
	assert.Equal(t, `invalid configuration: invalid metric name "test_met rick": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
	assert.Empty(t, md)
}
