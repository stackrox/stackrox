package benchmark

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// ParseMetricFamilies parses Prometheus text exposition into a map keyed by
// metric_name{label="value",...} with label names sorted lexicographically.
func ParseMetricFamilies(body []byte) (map[string]float64, error) {
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse prometheus text: %w", err)
	}

	out := make(map[string]float64, len(families))
	for name, family := range families {
		for _, metric := range family.GetMetric() {
			key := formatMetricKey(name, metric.GetLabel())
			value, ok, err := metricSampleValue(metric)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			out[key] = value
		}
	}
	return out, nil
}

// SumCounterDelta returns the sum of counter deltas for all series whose name
// matches metricName (with or without labels).
func SumCounterDelta(metricName string, before, after map[string]float64) float64 {
	return sumCounterDelta(metricName, before, after, nil)
}

// SumCounterDeltaFiltered returns the sum of counter deltas for series matching
// metricName and all label key=value pairs in labels.
func SumCounterDeltaFiltered(metricName string, before, after map[string]float64, labels map[string]string) float64 {
	return sumCounterDelta(metricName, before, after, labels)
}

// RatePerSec converts a counter delta over seconds into a per-second rate.
func RatePerSec(delta, seconds float64) float64 {
	if seconds <= 0 {
		return 0
	}
	return delta / seconds
}

var k8sSensorEventEgressResources = map[string]struct{}{
	"Deployment":     {},
	"Pod":            {},
	"Namespace":      {},
	"Node":           {},
	"ServiceAccount": {},
	"Role":           {},
	"RoleBinding":    {},
	"NetworkPolicy":  {},
	"Secret":         {},
	"Image":          {},
}

func sumCounterDelta(metricName string, before, after map[string]float64, labels map[string]string) float64 {
	keys := matchingSeriesKeys(metricName, before, after, labels, nil)
	var sum float64
	for key := range keys {
		sum += after[key] - before[key]
	}
	return sum
}

// SumCounterDeltaFilteredResourceIn sums counter deltas for series matching metricName,
// all labelFilters, and a resource label value in resources.
func SumCounterDeltaFilteredResourceIn(metricName string, before, after map[string]float64, labelFilters map[string]string, resources map[string]struct{}) float64 {
	keys := matchingSeriesKeys(metricName, before, after, labelFilters, resources)
	var sum float64
	for key := range keys {
		sum += after[key] - before[key]
	}
	return sum
}

func matchingSeriesKeys(metricName string, before, after map[string]float64, labelFilters map[string]string, resources map[string]struct{}) map[string]struct{} {
	keys := make(map[string]struct{})
	for key := range before {
		if seriesMatches(metricName, key, labelFilters, resources) {
			keys[key] = struct{}{}
		}
	}
	for key := range after {
		if seriesMatches(metricName, key, labelFilters, resources) {
			keys[key] = struct{}{}
		}
	}
	return keys
}

func seriesMatches(metricName, key string, labels map[string]string, resources map[string]struct{}) bool {
	name, seriesLabels := metricKeyLabels(key)
	if name != metricName {
		return false
	}
	if len(labels) > 0 && !labelMatches(seriesLabels, labels) {
		return false
	}
	if len(resources) > 0 {
		resource := labelValue(seriesLabels, "resource", "ResourceType")
		if _, ok := resources[resource]; !ok {
			return false
		}
	}
	return true
}

func labelMatches(seriesLabels map[string]string, filter map[string]string) bool {
	for k, v := range filter {
		if labelValue(seriesLabels, k, alternateLabelKey(k)) != v {
			return false
		}
	}
	return true
}

func labelValue(labels map[string]string, names ...string) string {
	for _, name := range names {
		if v, ok := labels[name]; ok {
			return v
		}
	}
	return ""
}

func alternateLabelKey(k string) string {
	switch k {
	case "type":
		return "Type"
	case "resource":
		return "ResourceType"
	case "action":
		return "Action"
	default:
		return ""
	}
}

func metricKeyLabels(key string) (string, map[string]string) {
	braceIdx := strings.IndexByte(key, '{')
	if braceIdx < 0 {
		return key, nil
	}
	if !strings.HasSuffix(key, "}") {
		return key[:braceIdx], nil
	}
	return key[:braceIdx], parseLabelSet(key[braceIdx+1 : len(key)-1])
}

func parseLabelSet(labelPart string) map[string]string {
	if labelPart == "" {
		return nil
	}
	labels := make(map[string]string)
	for _, pair := range splitLabelPairs(labelPart) {
		eqIdx := strings.IndexByte(pair, '=')
		if eqIdx < 0 {
			continue
		}
		name := pair[:eqIdx]
		value := pair[eqIdx+1:]
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		labels[name] = value
	}
	return labels
}

func splitLabelPairs(labelPart string) []string {
	var pairs []string
	var current strings.Builder
	inQuotes := false
	for i := 0; i < len(labelPart); i++ {
		ch := labelPart[i]
		switch {
		case ch == '"':
			inQuotes = !inQuotes
			current.WriteByte(ch)
		case ch == ',' && !inQuotes:
			pairs = append(pairs, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		pairs = append(pairs, strings.TrimSpace(current.String()))
	}
	return pairs
}

func formatMetricKey(name string, labels []*dto.LabelPair) string {
	if len(labels) == 0 {
		return name
	}
	sorted := make([]*dto.LabelPair, len(labels))
	copy(sorted, labels)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].GetName() < sorted[j].GetName()
	})

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i, lp := range sorted {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `%s=%q`, lp.GetName(), lp.GetValue())
	}
	b.WriteByte('}')
	return b.String()
}

func metricSampleValue(metric *dto.Metric) (float64, bool, error) {
	switch {
	case metric.GetCounter() != nil:
		return metric.GetCounter().GetValue(), true, nil
	case metric.GetGauge() != nil:
		return metric.GetGauge().GetValue(), true, nil
	case metric.GetUntyped() != nil:
		return metric.GetUntyped().GetValue(), true, nil
	default:
		return 0, false, nil
	}
}
