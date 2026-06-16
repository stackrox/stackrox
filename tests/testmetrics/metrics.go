package testmetrics

import (
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Metrics is a parsed set of Prometheus metric families that supports
// high-level value lookups by name and labels.
type Metrics struct {
	families map[string]*dto.MetricFamily
}

// Parse parses raw Prometheus exposition text into a Metrics set.
func Parse(text string) Metrics {
	parser := expfmt.NewTextParser(model.LegacyValidation)
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	families, _ := parser.TextToMetricFamilies(strings.NewReader(text))
	if families == nil {
		families = make(map[string]*dto.MetricFamily)
	}
	return Metrics{families: families}
}

// GetValue looks up a metric value by name and optional label matchers.
// Labels are specified as alternating key, value pairs:
//
//	m.GetValue("http_total", "method", "GET", "code", "200")
//
// When multiple series match (e.g. metrics scraped from several pods),
// their values are summed.
// Returns (value, true) on match, or (0, false) if not found.
func (m Metrics) GetValue(name string, labels ...string) (float64, bool) {
	fam, ok := m.families[name]
	if !ok {
		return 0, false
	}
	want := pairLabels(labels)
	total := 0.0
	found := false
	for _, met := range fam.GetMetric() {
		if !labelsMatch(met.GetLabel(), want) {
			continue
		}
		if val, ok := metricValue(fam.GetType(), met); ok {
			total += val
			found = true
		}
	}
	return total, found
}

// pairLabels converts alternating key, value strings into a label map.
func pairLabels(kv []string) map[string]string {
	if len(kv) == 0 {
		return nil
	}
	out := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}

// labelsMatch returns true if the metric's labels contain all wanted pairs.
func labelsMatch(have []*dto.LabelPair, want map[string]string) bool {
	if len(want) == 0 {
		return true
	}
	h := make(map[string]string, len(have))
	for _, lp := range have {
		h[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if h[k] != v {
			return false
		}
	}
	return true
}

// metricValue extracts the numeric value from a metric based on its family type.
func metricValue(typ dto.MetricType, m *dto.Metric) (float64, bool) {
	switch typ {
	case dto.MetricType_COUNTER:
		if c := m.GetCounter(); c != nil {
			return c.GetValue(), true
		}
	case dto.MetricType_GAUGE:
		if g := m.GetGauge(); g != nil {
			return g.GetValue(), true
		}
	case dto.MetricType_UNTYPED:
		if u := m.GetUntyped(); u != nil {
			return u.GetValue(), true
		}
	default:
	}
	return 0, false
}
