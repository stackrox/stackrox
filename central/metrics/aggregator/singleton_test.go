package aggregator

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stretchr/testify/assert"
)

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
