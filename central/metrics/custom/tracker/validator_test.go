package tracker

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestTranslateConfiguration(t *testing.T) {
	config := makeTestMetricLabels(t)
	mcfg, err := TranslateConfiguration(config, testLabelOrder)
	assert.NoError(t, err)
	assert.Equal(t, makeTestMetricConfiguration(t), mcfg)
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
		mcfg, err := TranslateConfiguration(config, testLabelOrder)
		assert.Equal(t, `invalid configuration: no labels specified for metric "metric"`, err.Error())
		assert.Empty(t, mcfg)
	}

	mcfg, err := TranslateConfiguration(nil, testLabelOrder)
	assert.NoError(t, err)
	assert.Empty(t, mcfg)
}

func Test_parseErrors(t *testing.T) {
	config := map[string]*storage.PrometheusMetrics_Group_Labels{
		"metric1": {
			Labels: []string{"unknown"},
		},
	}
	mcfg, err := TranslateConfiguration(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: label "unknown" for metric "metric1" is not in the list of known labels [CVE CVSS Cluster IsFixable Namespace Severity test]`, err.Error())
	assert.Empty(t, mcfg)

	delete(config, "metric1")
	config["met rick"] = nil
	mcfg, err = TranslateConfiguration(config, testLabelOrder)
	assert.Equal(t, `invalid configuration: invalid metric name "met rick": doesn't match "^[a-zA-Z_:][a-zA-Z0-9_:]*$"`, err.Error())
	assert.Empty(t, mcfg)
}
