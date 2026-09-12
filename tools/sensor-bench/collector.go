package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	pprofBaseURL = "http://127.0.0.1:6060"
	metricsURL   = "http://localhost:9090/metrics"
)

type RunMetadata struct {
	Config    *Config   `json:"config"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Duration  string    `json:"duration"`
}

func collectProfiles(outputDir string) error {
	profiles := map[string]string{
		"heap.pb.gz":      pprofBaseURL + "/debug/heap",
		"goroutine.pb.gz": pprofBaseURL + "/debug/goroutine",
	}
	for filename, url := range profiles {
		if err := downloadToFile(url, filepath.Join(outputDir, filename)); err != nil {
			return fmt.Errorf("collecting %s: %w", filename, err)
		}
	}
	return nil
}

func collectMetrics(outputDir string) error {
	resp, err := http.Get(metricsURL)
	if err != nil {
		return fmt.Errorf("fetching metrics: %w", err)
	}
	defer resp.Body.Close()

	dec := expfmt.NewDecoder(resp.Body, expfmt.NewFormat(expfmt.TypeTextPlain))
	families := make(map[string]*dto.MetricFamily)
	for {
		var mf dto.MetricFamily
		if err := dec.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decoding metrics: %w", err)
		}
		families[mf.GetName()] = &mf
	}

	flat := flattenMetrics(families)

	data, err := json.MarshalIndent(flat, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metrics: %w", err)
	}

	return os.WriteFile(filepath.Join(outputDir, "metrics.json"), data, 0644)
}

func writeRunMetadata(outputDir string, cfg *Config, start, end time.Time) error {
	meta := RunMetadata{
		Config:    cfg,
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start).String(),
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling run metadata: %w", err)
	}
	return os.WriteFile(filepath.Join(outputDir, "run.json"), data, 0644)
}

func downloadToFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

type MetricValue struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

func flattenMetrics(families map[string]*dto.MetricFamily) []MetricValue {
	var result []MetricValue
	for name, family := range families {
		for _, m := range family.GetMetric() {
			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}

			var value float64
			switch family.GetType() {
			case dto.MetricType_COUNTER:
				value = m.GetCounter().GetValue()
			case dto.MetricType_GAUGE:
				value = m.GetGauge().GetValue()
			case dto.MetricType_SUMMARY:
				value = m.GetSummary().GetSampleSum()
			case dto.MetricType_HISTOGRAM:
				value = m.GetHistogram().GetSampleSum()
			case dto.MetricType_UNTYPED:
				value = m.GetUntyped().GetValue()
			}

			result = append(result, MetricValue{
				Name:   name,
				Labels: labels,
				Value:  value,
			})
		}
	}
	return result
}
