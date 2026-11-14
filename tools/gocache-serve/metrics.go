package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MetricsRecord is one gocache-serve process's summary. Appended as a
// JSON line to a per-job local file; the file is uploaded whole as one
// blob per job (metrics/<yyyymmdd>/<run_id>) so the per-process network
// cost is a single small Put, and the blob doubles as the raw archive.
type MetricsRecord struct {
	Ts             int64  `json:"ts"`
	RunID          string `json:"run_id"`
	Job            string `json:"job"`
	Region         string `json:"region"`
	Account        string `json:"account"`
	Gets           int64  `json:"gets"`
	Hits           int64  `json:"hits"`
	Puts           int64  `json:"puts"`
	GetBytes       int64  `json:"get_bytes"`
	GetMs          int64  `json:"get_ms"`
	ElapsedMs      int64  `json:"elapsed_ms"`
	ConcurrentHits int64  `json:"concurrent_hits"`
}

// metricsLocalPath is the per-job metrics file, keyed by run ID. All
// processes of a job share the runner machine, so appends here aggregate
// the whole job before the single upload.
func metricsLocalPath(runID string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, runID)
	return filepath.Join(os.TempDir(), "gocache-serve-metrics-"+safe)
}

// appendMetric records one process summary locally (free, no network).
func appendMetric(rec MetricsRecord) {
	f, err := os.OpenFile(metricsLocalPath(rec.RunID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return // metrics must never break the build
	}
	defer f.Close()
	json.NewEncoder(f).Encode(rec)
}

// metricsBlobKey is the per-job archive blob: date-bucketed by UTC day.
func metricsBlobKey(runID string) string {
	return fmt.Sprintf("metrics/%s/%s", time.Now().UTC().Format("20060102"), runID)
}

// uploadJobMetrics pushes the job's accumulated records as one blob.
// Centralized on the hub account: one Put per job (~100ms even
// cross-region), and readers (-metrics-report, -metrics-push) have
// exactly one place to look. Last-write-wins across the job's processes
// is correct: the file only grows via local appends.
func uploadJobMetrics(runID string, rec MetricsRecord) {
	path := metricsLocalPath(runID)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return
	}
	var b Backend
	if os.Getenv("GOCACHE_SERVE_BACKEND") == "dir" {
		b, err = newBackend()
		if err != nil {
			return
		}
	} else {
		b = hubBackend()
	}
	if err := b.Put(metricsBlobKey(runID), data); err != nil {
		log.Printf("metrics upload failed: %v", err)
	}
	_ = rec // record already appended locally; upload is file-granular
}

// readMetricsBlobs downloads and parses all metric records under a key
// prefix (a date or run ID). Returns records plus the source keys.
func readMetricsBlobs(b Backend, prefix string) ([]MetricsRecord, []string, error) {
	keys, err := b.ListPrefix("metrics/")
	if err != nil {
		return nil, nil, err
	}
	var matched []string
	// DirBackend sanitizes '/' to '_' in stored keys; Azure keeps slashes.
	// Match both forms so the report works against either backend.
	for _, k := range keys {
		if strings.HasPrefix(k, "metrics/") && strings.Contains(k, prefix) {
			matched = append(matched, k)
		} else if strings.HasPrefix(k, "metrics_") && strings.Contains(k, prefix) {
			matched = append(matched, k)
		}
	}
	sort.Strings(matched)
	var recs []MetricsRecord
	for _, k := range matched {
		body, err := b.Get(k)
		if err != nil || body == nil {
			continue
		}
		sc := bufio.NewScanner(strings.NewReader(string(body)))
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var rec MetricsRecord
			if json.Unmarshal(sc.Bytes(), &rec) == nil {
				recs = append(recs, rec)
			}
		}
	}
	return recs, matched, nil
}

// reportMetrics aggregates records for ad-hoc querying: totals, per-job
// rollups, and per-region/account breakdowns.
func reportMetrics(prefix string) error {
	var b Backend
	var err error
	if os.Getenv("GOCACHE_SERVE_BACKEND") == "dir" {
		b, err = newBackend()
	} else {
		b = hubBackend()
	}
	if err != nil {
		return err
	}
	recs, keys, err := readMetricsBlobs(b, prefix)
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		fmt.Printf("no metrics found for prefix %q (%d blobs matched)\n", prefix, len(keys))
		return nil
	}

	var tot MetricsRecord
	jobs := map[string]*MetricsRecord{}
	regions := map[string]*MetricsRecord{}
	accounts := map[string]*MetricsRecord{}
	for _, r := range recs {
		tot.Gets += r.Gets
		tot.Hits += r.Hits
		tot.Puts += r.Puts
		tot.GetBytes += r.GetBytes
		tot.GetMs += r.GetMs
		tot.ElapsedMs += r.ElapsedMs
		tot.ConcurrentHits += r.ConcurrentHits
		for _, m := range []struct {
			key string
			agg map[string]*MetricsRecord
		}{{r.RunID, jobs}, {r.Region, regions}, {r.Account, accounts}} {
			a, ok := m.agg[m.key]
			if !ok {
				a = &MetricsRecord{RunID: m.key, Region: m.key, Account: m.key}
				m.agg[m.key] = a
			}
			a.Gets += r.Gets
			a.Hits += r.Hits
			a.Puts += r.Puts
			a.GetBytes += r.GetBytes
			a.GetMs += r.GetMs
			a.ElapsedMs += r.ElapsedMs
		}
	}
	hr := func(g, h int64) float64 { return float64(h) / float64(g) * 100 }

	fmt.Printf("=== metrics for %q: %d records, %d job blobs ===\n", prefix, len(recs), len(keys))
	fmt.Printf("TOTAL: %d gets (%d hits, %.1f%%), %d puts, %.1f MB transferred\n",
		tot.Gets, tot.Hits, hr(tot.Gets, tot.Hits), tot.Puts, float64(tot.GetBytes)/1048576)
	if tot.GetMs > 0 {
		fmt.Printf("       network: %.1f MB/s average GET throughput\n",
			float64(tot.GetBytes)/(float64(tot.GetMs)/1e3)/1048576)
	}
	fmt.Printf("\n-- per job --\n")
	for _, run := range sortedKeys(jobs) {
		j := jobs[run]
		fmt.Printf("  %s: %d gets (%.1f%% hits), %d puts, %d ms\n", run, j.Gets, hr(j.Gets, j.Hits), j.Puts, j.ElapsedMs)
	}
	fmt.Printf("\n-- per region --\n")
	for _, rg := range sortedKeys(regions) {
		r := regions[rg]
		fmt.Printf("  %-16s %d gets (%.1f%%), %d puts\n", rg, r.Gets, hr(r.Gets, r.Hits), r.Puts)
	}
	fmt.Printf("\n-- per account --\n")
	for _, ac := range sortedKeys(accounts) {
		a := accounts[ac]
		fmt.Printf("  %-20s %d gets (%.1f%%), %d puts, %.1f MB\n", ac, a.Gets, hr(a.Gets, a.Hits), a.Puts, float64(a.GetBytes)/1048576)
	}
	return nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- Log Analytics ingestion (-metrics-push) ---
//
// Batch path: metric blobs -> one POST per blob to the Logs Ingestion
// API (DCE + DCR, Monitoring Metrics Publisher role on the SPN). Keeps
// the ingestion dependency out of the build path entirely; run on a
// schedule (or manually) for dashboards/alerts via KQL over the
// CIGoCacheMetrics_CL table.

func monitorToken(clientID, clientSecret, tenantID string) (string, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {"https://monitor.azure.com/.default"},
	}.Encode()
	resp, err := http.Post(
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		"application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tr struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		return "", fmt.Errorf("monitor token: %s", strings.TrimSpace(string(body)))
	}
	return tr.AccessToken, nil
}

// pushMetrics reads metric blobs matching a prefix and ingests them into
// Log Analytics. Idempotent-ish: re-pushing a day duplicates rows — the
// scheduled caller should track what it pushed, or query dedupes by
// run_id (blob granularity = one row-set per job).
func pushMetrics(prefix string) error {
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	dce := os.Getenv("GOCACHE_METRICS_DCE")
	dcr := os.Getenv("GOCACHE_METRICS_DCR")
	if dce == "" {
		dce = "https://ci-gocache-dce-hm6v.eastus-1.ingest.monitor.azure.com"
	}
	if dcr == "" {
		dcr = "dcr-c06658333bd54903af2f5910a9a07f99"
	}
	token, err := monitorToken(clientID, clientSecret, tenantID)
	if err != nil {
		return err
	}
	b := hubBackend()
	recs, keys, err := readMetricsBlobs(b, prefix)
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		fmt.Printf("no metrics to push for prefix %q\n", prefix)
		return nil
	}
	// Build the ingestion body: array of records with TimeGenerated from ts.
	type row struct {
		TimeGenerated    string `json:"TimeGenerated"`
		RunID            string `json:"run_id"`
		Job              string `json:"job"`
		Region           string `json:"region"`
		Account          string `json:"account"`
		Gets             int64  `json:"gets"`
		Hits             int64  `json:"hits"`
		Puts             int64  `json:"puts"`
		GetBytes         int64  `json:"get_bytes"`
		GetMs            int64  `json:"get_ms"`
		ElapsedMs        int64  `json:"elapsed_ms"`
		ConcurrentHits   int64  `json:"concurrent_hits"`
	}
	rows := make([]row, 0, len(recs))
	for _, r := range recs {
		rows = append(rows, row{
			TimeGenerated:  time.Unix(r.Ts, 0).UTC().Format(time.RFC3339Nano),
			RunID:          r.RunID,
			Job:            r.Job,
			Region:         r.Region,
			Account:        r.Account,
			Gets:           r.Gets,
			Hits:           r.Hits,
			Puts:           r.Puts,
			GetBytes:       r.GetBytes,
			GetMs:          r.GetMs,
			ElapsedMs:      r.ElapsedMs,
			ConcurrentHits: r.ConcurrentHits,
		})
	}
	body, _ := json.Marshal(rows)
	url := fmt.Sprintf("%s/dataCollectionRules/%s/streams/Custom-CIGoCacheMetrics_CL?api-version=2023-01-01", dce, dcr)
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("ingestion HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	fmt.Printf("pushed %d records (%d blobs) to Log Analytics\n", len(recs), len(keys))
	return nil
}
