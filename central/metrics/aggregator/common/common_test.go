package common

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

var testData = []map[Label]string{
	{
		"Severity":  "CRITICAL",
		"Cluster":   "cluster 1",
		"Namespace": "ns 1",
	}, {
		"Severity":  "HIGH",
		"Cluster":   "cluster 2",
		"Namespace": "ns 2",
	},
	{
		"Severity":  "LOW",
		"Cluster":   "cluster 3",
		"Namespace": "ns 3",
	},
	{
		"Severity":  "CRITICAL",
		"Cluster":   "cluster 1",
		"Namespace": "ns 3",
	},
	{
		"Severity":  "LOW",
		"Cluster":   "cluster 5",
		"Namespace": "ns 3",
	},
}

func makeTestMetricLabels(t *testing.T) map[string]*storage.PrometheusMetricsConfig_Labels {
	pfx := strings.ReplaceAll(t.Name(), "/", "_")
	return map[string]*storage.PrometheusMetricsConfig_Labels{
		pfx + "_metric1": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"Severity": {
					Expression: []*storage.PrometheusMetricsConfig_Labels_Expression_Condition{
						{
							Operator: "=",
							Argument: "CRITICAL*",
						}, {
							Operator: "OR",
						}, {
							Operator: "=",
							Argument: "HIGH*",
						},
					},
				},
				"Cluster": nil,
			},
			Exposure:     storage.PrometheusMetricsConfig_Labels_BOTH,
			RegistryName: "registry1",
		},
		pfx + "_metric2": {
			Labels: map[string]*storage.PrometheusMetricsConfig_Labels_Expression{
				"Namespace": {},
			},
		},
	}
}

func makeTestMetricLabelExpression(t *testing.T) MetricsConfiguration {
	pfx := MetricName(strings.ReplaceAll(t.Name(), "/", "_"))
	return MetricsConfiguration{
		pfx + "_metric1": {
			"Severity": {
				MustMakeCondition("=", "CRITICAL*"),
				MustMakeCondition("OR", ""),
				MustMakeCondition("=", "HIGH*"),
			},
			"Cluster": nil,
		},
		pfx + "_metric2": {
			"Namespace": nil,
		},
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

func TestHasAnyLabelOf(t *testing.T) {
	mcfg := MetricsConfiguration{
		"metric1": map[Label]Expression{
			"label1": nil,
			"label2": nil,
		},
		"metric2": map[Label]Expression{
			"label3": nil,
			"label4": nil,
		},
	}
	assert.False(t, mcfg.HasAnyLabelOf([]Label{}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label1"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label3"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label0", "label1"}))
	assert.True(t, mcfg.HasAnyLabelOf([]Label{"label0", "label4"}))
	assert.False(t, mcfg.HasAnyLabelOf([]Label{"label0", "label5"}))
}

func TestOneOrMore(t *testing.T) {
	assert.Equal(t, 1, OneOrMore(-2).Count())
	assert.Equal(t, 1, OneOrMore(-1).Count())
	assert.Equal(t, 1, OneOrMore(0).Count())
	assert.Equal(t, 1, OneOrMore(1).Count())
	assert.Equal(t, 2, OneOrMore(2).Count())
}

/*
func TestEquals(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		assert.False(t, one.Equals(MetricsConfiguration{}))
		assert.False(t, MetricsConfiguration{}.Equals(one))
	})

	t.Run("equals", func(t *testing.T) {
		var m MetricsConfiguration
		assert.True(t, m.Equals(m))
		assert.True(t, MetricsConfiguration{}.Equals(MetricsConfiguration{}))
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)
		assert.True(t, one.Equals(one))
		assert.True(t, one.Equals(another))
		assert.True(t, another.Equals(one))
	})

	t.Run("changed condition", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

	loop:
		for _, labels := range another {
			for _, expr := range labels {
				for _, cond := range expr {
					cond.arg = "changed"
					break loop
				}
			}
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra condition", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

	loop:
		for _, labels := range another {
			for label, expr := range labels {
				labels[label] = append(expr, &Condition{"=", "extra"})
				break loop
			}
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra label", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

		for _, labels := range another {
			labels["extra"] = Expression{}
			break
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra metric", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)
		another["extra"] = map[Label]Expression{}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

}

func TestEqualLabels(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		assert.False(t, one.Equals(MetricsConfiguration{}))
		assert.False(t, MetricsConfiguration{}.Equals(one))
	})

	t.Run("equals", func(t *testing.T) {
		var m MetricsConfiguration
		assert.True(t, m.Equals(m))
		assert.True(t, MetricsConfiguration{}.Equals(MetricsConfiguration{}))
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)
		assert.True(t, one.Equals(one))
		assert.True(t, one.Equals(another))
		assert.True(t, another.Equals(one))
	})

	t.Run("changed condition", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

	loop:
		for _, labels := range another {
			for _, expr := range labels {
				for _, cond := range expr {
					cond.arg = "changed"
					break loop
				}
			}
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra condition", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

	loop:
		for _, labels := range another {
			for label, expr := range labels {
				labels[label] = append(expr, &Condition{"=", "extra"})
				break loop
			}
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra label", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)

		for _, labels := range another {
			labels["extra"] = Expression{}
			break
		}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

	t.Run("extra metric", func(t *testing.T) {
		one := makeTestMetricLabelExpression(t)
		another := makeTestMetricLabelExpression(t)
		another["extra"] = map[Label]Expression{}
		assert.False(t, one.Equals(another))
		assert.False(t, another.Equals(one))
	})

}
*/

func TestDiffLabels(t *testing.T) {
	tests := []struct {
		name         string
		a, b         MetricsConfiguration
		wantToAdd    []MetricName
		wantToDelete []MetricName
		wantChanged  []MetricName
	}{
		{
			name:         "both nil",
			a:            nil,
			b:            nil,
			wantToAdd:    nil,
			wantToDelete: nil,
		},
		{
			name:         "a empty, b has one",
			a:            MetricsConfiguration{},
			b:            MetricsConfiguration{"metric1": {"label1": nil}},
			wantToAdd:    []MetricName{"metric1"},
			wantToDelete: nil,
		},
		{
			name:         "a has one, b empty",
			a:            MetricsConfiguration{"metric1": {"label1": nil}},
			b:            MetricsConfiguration{},
			wantToAdd:    nil,
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name:         "a has one, b has another",
			a:            MetricsConfiguration{"metric1": {"label1": nil}},
			b:            MetricsConfiguration{"metric2": {"label2": nil}},
			wantToAdd:    []MetricName{"metric2"},
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name: "a and b have overlap",
			a: MetricsConfiguration{
				"metric1": {"label1": nil},
				"metric2": {"label2": nil},
			},
			b: MetricsConfiguration{
				"metric2": {"label2": nil},
				"metric3": {"label3": nil},
			},
			wantToAdd:    []MetricName{"metric3"},
			wantToDelete: []MetricName{"metric1"},
		},
		{
			name: "identical",
			a: MetricsConfiguration{
				"metric1": {"label1": nil},
				"metric2": {"label2": nil},
			},
			b: MetricsConfiguration{
				"metric1": {"label1": nil},
				"metric2": {"label2": nil},
			},
			wantToAdd:    nil,
			wantToDelete: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToAdd, gotToDelete, gotChanged := tt.a.DiffLabels(tt.b)
			assert.ElementsMatch(t, tt.wantToAdd, gotToAdd)
			assert.ElementsMatch(t, tt.wantToDelete, gotToDelete)
			assert.ElementsMatch(t, tt.wantChanged, gotChanged)
		})
	}
}
