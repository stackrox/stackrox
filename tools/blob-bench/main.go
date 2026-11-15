// blob-bench measures Azure Blob Storage scalability under concurrent reads:
// does aggregate throughput scale with client concurrency, or does the
// account throttle / latency inflate before the CI fleet gets large?
//
// Usage:
//
//	blob-bench -concurrency 1,5,10,25,50,100 -duration 45s
//
// Env: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET,
//
//	AZURE_STORAGE_ACCOUNT (default cigocache), GOCACHE_SERVE_CONTAINER
//	(default ci-gocache).
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	account   = envOr("AZURE_STORAGE_ACCOUNT", "cigocache")
	container = envOr("GOCACHE_SERVE_CONTAINER", "ci-gocache")
	tenantID  = os.Getenv("AZURE_TENANT_ID")
	clientID  = os.Getenv("AZURE_CLIENT_ID")
	secret    = os.Getenv("AZURE_CLIENT_SECRET")
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var (
	tokenMu  sync.Mutex
	token    string
	tokenExp time.Time
)

func getToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if token != "" && time.Now().Before(tokenExp.Add(-time.Minute)) {
		return token, nil
	}
	form := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s&scope=https://storage.azure.com/.default",
		clientID, secret)
	resp, err := http.Post(fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		"application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := jsonDecode(resp.Body, &tr); err != nil || tr.AccessToken == "" {
		return "", fmt.Errorf("token fetch failed")
	}
	token, tokenExp = tr.AccessToken, time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second)
	return token, nil
}

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func baseURL() string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", account, container)
}

type client struct {
	hc *http.Client
}

func newClient() *client {
	return &client{hc: &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 128,
			MaxConnsPerHost:     128,
		},
	}}
}

func (c *client) do(url string) (*http.Response, error) {
	tok, err := getToken()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("x-ms-version", "2023-11-03")
	return c.hc.Do(req)
}

// listBlobs returns all blob names (paginated REST list).
func (c *client) listBlobs() ([]string, error) {
	var names []string
	marker := ""
	for {
		url := baseURL() + "?restype=container&comp=list&maxresults=5000"
		if marker != "" {
			url += "&marker=" + marker
		}
		resp, err := c.do(url)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("list: HTTP %d", resp.StatusCode)
		}
		var lr struct {
			Blobs struct {
				Blob []struct {
					Name string `xml:"Name"`
				} `xml:"Blob"`
			} `xml:"Blobs"`
			NextMarker string `xml:"NextMarker"`
		}
		if err := xml.Unmarshal(body, &lr); err != nil {
			return nil, err
		}
		for _, b := range lr.Blobs.Blob {
			// Skip control blobs; bench real cache entries only
			if !strings.HasPrefix(b.Name, "lease-") && !strings.HasPrefix(b.Name, "manifest-") &&
				!strings.HasPrefix(b.Name, "bundle-") && b.Name != "bundle-index" {
				names = append(names, b.Name)
			}
		}
		if lr.NextMarker == "" {
			break
		}
		marker = lr.NextMarker
	}
	return names, nil
}

type result struct {
	latencies []time.Duration
	bytes     int64
	requests  int64
	throttled int64 // HTTP 503
	errors    int64
}

func pct(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	i := int(float64(len(sorted)-1) * p)
	return sorted[i]
}

func main() {
	levels := flag.String("concurrency", "1,5,10,25,50,100", "comma-separated concurrency levels")
	dur := flag.Duration("duration", 45*time.Second, "per-level duration")
	flag.Parse()

	c := newClient()
	names, err := c.listBlobs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list blobs: %v\n", err)
		os.Exit(1)
	}
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "no blobs to bench")
		os.Exit(1)
	}
	var totalBytes int64
	fmt.Printf("# blob-bench: account=%s container=%s blobs=%d level-duration=%s\n",
		account, container, len(names), dur)

	var levelList []int
	for _, s := range strings.Split(*levels, ",") {
		var n int
		fmt.Sscanf(s, "%d", &n)
		if n > 0 {
			levelList = append(levelList, n)
		}
	}

	for _, n := range levelList {
		r := &result{latencies: make([]time.Duration, 0, 100000)}
		var mu sync.Mutex
		var idx int64
		deadline := time.Now().Add(*dur)
		var wg sync.WaitGroup

		for w := 0; w < n; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for time.Now().Before(deadline) {
					i := atomic.AddInt64(&idx, 1)
					name := names[int(i)%len(names)]
					start := time.Now()
					resp, err := c.do(baseURL() + "/" + name)
					d := time.Since(start)
					mu.Lock()
					r.latencies = append(r.latencies, d)
					mu.Unlock()
					if err != nil {
						mu.Lock()
						r.errors++
						mu.Unlock()
						continue
					}
					b, _ := io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
					mu.Lock()
					r.requests++
					r.bytes += b
					if resp.StatusCode == 503 {
						r.throttled++
					}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		sort.Slice(r.latencies, func(i, j int) bool { return r.latencies[i] < r.latencies[j] })
		elapsed := dur.Seconds()
		mbps := float64(r.bytes) / elapsed / 1048576
		rps := float64(r.requests+r.errors) / elapsed
		fmt.Printf("concurrency=%d throughput=%.1fMB/s reqs/s=%.1f p50=%v p95=%v p99=%v throttled503=%d errors=%d\n",
			n, mbps, rps,
			pct(r.latencies, 0.50).Round(time.Millisecond),
			pct(r.latencies, 0.95).Round(time.Millisecond),
			pct(r.latencies, 0.99).Round(time.Millisecond),
			r.throttled, r.errors)
		totalBytes += r.bytes
	}
	_ = totalBytes
}
