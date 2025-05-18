package common

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func Test_validateMetricName(t *testing.T) {
	tests := map[string]string{
		"good":             "",
		"not good":         "bad characters",
		"":                 "empty",
		"abc_defAZ0145609": "",
		"not-good":         "bad characters",
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

func TestMakeTrackFunc(t *testing.T) {
	type myDS struct{}
	result := make(map[string][]*Record)
	track := MakeTrackFunc(
		myDS{},
		func() MetricLabelExpressions {
			return MetricLabelExpressions{}
		},
		func(ctx context.Context, md myDS, mle MetricLabelExpressions) *Result {
			return &Result{
				aggregated: map[MetricName]map[metricKey]*Record{
					"metric1": {
						metricKey("abc"): {
							prometheus.Labels{"label1": "value1"},
							37,
						},
						metricKey("def"): {
							prometheus.Labels{"label1": "value1"},
							73,
						},
					},
					"metric2": {
						metricKey("abc"): {
							prometheus.Labels{"label1": "value1"},
							44,
						},
					},
				},
			}
		},
		func(metricName string, labels prometheus.Labels, total int) {
			result[metricName] = append(result[metricName], &Record{labels, total})
		},
	)
	track(context.Background())
	assert.Equal(t, map[string][]*Record{
		"metric1": {
			{
				prometheus.Labels{"label1": "value1"},
				37,
			},
			{
				prometheus.Labels{"label1": "value1"},
				73,
			},
		},
		"metric2": {
			{
				prometheus.Labels{"label1": "value1"},
				44,
			},
		},
	}, result)
}
