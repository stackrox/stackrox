package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Backend interface for cloud storage
type Backend interface {
	Get(actionID string) ([]byte, error) // nil, nil = not found
	Put(actionID string, body []byte) error
	PutIfNotExist(actionID string, body []byte) (bool, error) // atomic; false = exists
	ListKeys() ([]string, error)                              // list all blob keys in the container
	Close() error
}

// gzipAtRest compresses blobs for transfer: cross-region, the network
// (~6MB/s, 46ms RTT) is far slower than the compute to decompress
// (~400MB/s gzip), and Go object data compresses 2-3x. Level 1 keeps
// the put path fast while capturing most of the ratio.
func gzipBytes(body []byte) []byte {
	// already gzip (e.g. pfb bundles from SerializeBundle) — don't double-wrap
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		return body
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	zw.Write(body)
	zw.Close()
	return buf.Bytes()
}

// maybeGunzip reverses gzipBytes; non-gzip payloads (legacy raw blobs)
// pass through unchanged. A false positive would require a raw blob that
// both starts with the gzip magic AND passes the full CRC32 trailer
// check — effectively impossible for object files.
func maybeGunzip(body []byte) []byte {
	if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
		return body
	}
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return body
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return body
	}
	return raw
}

// DirBackend is a filesystem-backed store for local end-to-end testing of
// the full pipeline (manifests, run-scoping, prefetch bundles) without
// Azure. Blobs are files under a root dir; keys are sanitized paths.
type DirBackend struct {
	root string
}

func newDirBackend() (*DirBackend, error) {
	root := os.Getenv("GOCACHE_SERVE_DIR_STORE")
	if root == "" {
		root = "/tmp/gocache-serve-store"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &DirBackend{root: root}, nil
}

func (d *DirBackend) path(key string) string {
	// keep key readable but flat-safe
	return filepath.Join(d.root, strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' {
			return '_'
		}
		return r
	}, key))
}

func (d *DirBackend) Get(key string) ([]byte, error) {
	b, err := os.ReadFile(d.path(key))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return maybeGunzip(b), err
}

func (d *DirBackend) Put(key string, body []byte) error {
	p := d.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, gzipBytes(body), 0o644)
}

func (d *DirBackend) PutIfNotExist(key string, body []byte) (bool, error) {
	p := d.path(key)
	if _, err := os.Stat(p); err == nil {
		return false, nil
	}
	return true, d.Put(key, body)
}

func (d *DirBackend) ListKeys() ([]string, error) {
	var keys []string
	entries, err := os.ReadDir(d.root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			keys = append(keys, e.Name())
		}
	}
	return keys, nil
}

func (d *DirBackend) Close() error { return nil }

// NoopBackend does nothing (for protocol testing)
type NoopBackend struct{}

func (n *NoopBackend) Get(string) ([]byte, error)                 { return nil, nil }
func (n *NoopBackend) Put(string, []byte) error                   { return nil }
func (n *NoopBackend) PutIfNotExist(string, []byte) (bool, error) { return true, nil }
func (n *NoopBackend) ListKeys() ([]string, error)                { return nil, nil }
func (n *NoopBackend) Close() error                               { return nil }

// AzureBackend talks directly to Azure Blob Storage via REST API.
// Zero dependencies — uses net/http with SPN OAuth for auth.
type AzureBackend struct {
	account      string
	container    string
	tenantID     string
	clientID     string
	clientSecret string
	// baseURL overrides the real endpoint for tests (served by httptest);
	// empty means https://<account>.blob.core.windows.net.
	baseURL string

	tokenMu  sync.Mutex
	token    string
	tokenExp time.Time

	client *http.Client

	tokenCache *tokenDiskCache
}

// detectRegion returns the runner's Azure region via Instance Metadata
// Service (IMDS). Falls back to the default region if IMDS is unavailable.
func detectRegion() string {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("GET",
		"http://169.254.169.254/metadata/instance/compute/location?api-version=2021-02-01&format=text", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata", "true")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// probeAccounts measures RTT to each storage account endpoint and returns
// the fastest. All HEADs race concurrently, so wall cost is one RTT, not
// the sum. Any HTTP response (even 400) proves reachability and yields an
// RTT sample. Used only for regions the static map doesn't know: GitHub's
// runner pool keeps growing (e.g. chilecentral) and a wrong guess costs
// minutes of cross-region prefetch (measured: 3m54s vs 72s same-region).
func probeAccounts(accounts []string) (string, time.Duration) {
	type result struct {
		acct string
		rtt  time.Duration
	}
	client := &http.Client{Timeout: 3 * time.Second}
	ch := make(chan result, len(accounts))
	for _, acct := range accounts {
		go func(a string) {
			start := time.Now()
			resp, err := client.Head(fmt.Sprintf("https://%s.blob.core.windows.net/", a))
			if err != nil {
				ch <- result{"", 0}
				return
			}
			resp.Body.Close()
			ch <- result{a, time.Since(start)}
		}(acct)
	}
	best, bestRTT := "", time.Duration(math.MaxInt64)
	for range accounts {
		r := <-ch
		if r.acct != "" && r.rtt < bestRTT {
			best, bestRTT = r.acct, r.rtt
		}
	}
	return best, bestRTT
}


// learnedAccountPath is the per-region disk file caching the probed
// account choice: "<account> <unix-ts>". gocache-serve processes are
// per-command and short-lived; without this every process would re-probe.
func learnedAccountPath(region string) string {
	return filepath.Join(os.TempDir(), "gocache-serve-account-"+strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, region))
}

// loadLearnedAccount returns the cached account choice if fresh (< 24h).
// The disk cache is a per-machine memo of the shared registry entry —
// it amortizes the registry GET across the ~20 go invocations of one job.
func loadLearnedAccount(region string) string {
	data, err := os.ReadFile(learnedAccountPath(region))
	if err != nil {
		return ""
	}
	acct, ts, ok := parseLearnedEntry(data)
	if !ok || time.Since(time.Unix(ts, 0)) > 24*time.Hour {
		return ""
	}
	return acct
}

// parseLearnedEntry decodes the shared entry format "<account> <unix-ts>",
// used identically by the disk cache and the registry blob.
func parseLearnedEntry(data []byte) (string, int64, bool) {
	fields := strings.Fields(string(data))
	if len(fields) != 2 {
		return "", 0, false
	}
	ts, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return fields[0], ts, true
}

// saveLearnedAccount records the account choice for subsequent processes.
func saveLearnedAccount(region, acct string) {
	path := learnedAccountPath(region)
	if f, err := os.Create(path); err == nil {
		fmt.Fprintf(f, "%s %d\n", acct, time.Now().Unix())
		f.Close()
	}
}

// hubBackend returns a minimal backend bound to the westus3 hub account,
// used for registry reads/writes. Token scope is account-independent, so
// this works before the region's account is selected (no chicken-and-egg).
func hubBackend() *AzureBackend {
	return &AzureBackend{
		account:      "cigocache",
		container:    containerFromEnv(),
		tenantID:     os.Getenv("AZURE_TENANT_ID"),
		clientID:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// fetchRegistryAccount reads the shared region→account entry from the
// registry blob on the hub. Returns "" when absent, stale (> 24h), or
// unreachable — callers fall through to a live probe. One tiny GET,
// ~one RTT, once per machine per day.
func fetchRegistryAccount(region string) string {
	body, err := hubBackend().Get("region-map/" + region)
	if err != nil || body == nil {
		return ""
	}
	acct, ts, ok := parseLearnedEntry(body)
	if !ok || time.Since(time.Unix(ts, 0)) > 24*time.Hour {
		return ""
	}
	return acct
}

// writeRegistryAccount records the probed answer fleet-wide: the entry's
// timestamp is the fleet TTL, so at most ~one probe per region per day
// across ALL runners — every other machine just GETs this entry. Plain
// Put (last-write-wins): concurrent cold-start probes race but converge
// (shared topology ⇒ same winner), and container versioning keeps every
// historical value as the audit trail of topology shifts.
func writeRegistryAccount(region, acct string) {
	entry := fmt.Sprintf("%s %d\n", acct, time.Now().Unix())
	if err := hubBackend().Put("region-map/"+region, []byte(entry)); err != nil {
		log.Printf("gocache-serve: region registry write failed: %v", err)
		return
	}
	log.Printf("gocache-serve: region=%s registry updated: %s", region, acct)
}

//     written to the registry (timestamp refreshes the fleet TTL) and disk
//  4. hub constant — offline fallback: reached only when the registry is
//     unreachable AND all probes failed, i.e. the blob store itself is
//     unreachable, i.e. no cache can be served and the choice is moot.
//     Returns a valid account so selection never hangs or error-paths.
// A topology change (measured: 3m54s vs 72s on a bad guess) therefore
// heals within one day, fleet-wide, with no monitoring loop. There is
// deliberately NO compiled-in region map: it could only ever answer when
// the store is unreachable (choice moot), and maintaining it would be
// exactly the monitoring loop this design eliminates.
func storageAccountForRegion(region string) string {
	if acct := loadLearnedAccount(region); acct != "" {
		return acct
	}
	if acct := fetchRegistryAccount(region); acct != "" {
		saveLearnedAccount(region, acct)
		return acct
	}
	if acct, rtt := probeAccounts([]string{"cigocache", "cigocacheeast", "cigochachecentral"}); acct != "" {
		writeRegistryAccount(region, acct)
		saveLearnedAccount(region, acct)
		log.Printf("gocache-serve: region=%s probe picked %s (%dms)", region, acct, rtt.Milliseconds())
		return acct
	}
	return "cigocache"
}

// containerFromEnv returns the blob container name, overridable for
// A/B experiments (fresh containers = identical cold-start state).
func containerFromEnv() string {
	if c := os.Getenv("GOCACHE_SERVE_CONTAINER"); c != "" {
		return c
	}
	return "ci-gocache"
}

func newAzureBackend() (*AzureBackend, error) {
	// Use explicitly-set account, or detect from runner region
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	// Always log the runner region (observability for same-region
	// classification of pinned runs); only select by it when unpinned.
	region := detectRegion()
	if account == "" {
		account = storageAccountForRegion(region)
	}
	log.Printf("gocache-serve: region=%s account=%s", region, account)
	if account == "" {
		return nil, fmt.Errorf("AZURE_STORAGE_ACCOUNT not set and region detection failed")
	}
	b := &AzureBackend{
		account:      account,
		container:    containerFromEnv(),
		tenantID:     os.Getenv("AZURE_TENANT_ID"),
		clientID:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		// Tuned transport: the default Go transport keeps only 2 idle
		// connections per host — with 64-way prefetch, nearly every request
		// paid a fresh TLS handshake (~150ms cross-region). The scalability
		// bench measured 593MB/s at 100 connections with this setting; the
		// default throttled effective prefetch to ~30MB/s.
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          256,
				MaxIdleConnsPerHost:   128,
				MaxConnsPerHost:       256,
				IdleConnTimeout:       90 * time.Second,
				ForceAttemptHTTP2:     true,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 20 * time.Second,
			},
		},
		tokenCache: newTokenDiskCache(),
	}
	if b.tenantID == "" || b.clientID == "" || b.clientSecret == "" {
		return nil, fmt.Errorf("Azure SPN env not set (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)")
	}
	return b, nil
}

func (a *AzureBackend) containerURL() string {
	if a.baseURL != "" {
		return a.baseURL + "/" + a.container
	}
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", a.account, a.container)
}

func (a *AzureBackend) blobURL(actionID string) string {
	return fmt.Sprintf("%s/%s", a.containerURL(), actionID)
}

// getToken fetches an OAuth token via client credentials, cached until expiry.
// redactSecrets scrubs credential material from error text before it
// reaches logs. AAD/storage error bodies should never echo the secret,
// but defense-in-depth: a leaked secret in CI logs would require
// rotation of the very credential we just fixed.
func (a *AzureBackend) redactSecrets(s string) string {
	if a.clientSecret != "" {
		s = strings.ReplaceAll(s, a.clientSecret, "***")
	}
	if a.token != "" {
		s = strings.ReplaceAll(s, a.token, "***")
	}
	return s
}

func (a *AzureBackend) getToken() (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.token != "" && time.Now().Before(a.tokenExp.Add(-time.Minute)) {
		return a.token, nil
	}

	// Disk cache: every `go` invocation spawns a new gocache-serve process;
	// without this each pays a ~300ms OAuth round trip.
	if a.tokenCache != nil {
		if tok, exp, ok := a.tokenCache.Load(); ok && time.Now().Before(exp.Add(-2*time.Minute)) {
			a.token, a.tokenExp = tok, exp
			return tok, nil
		}
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.tenantID)
	// url.Values.Encode, NOT raw Sprintf: client secrets routinely contain
	// '+'/'&'/'%' — raw interpolation corrupts the form body ('+' decodes as
	// space, '&' truncates the field) and AAD rejects the secret.
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {a.clientID},
		"client_secret": {a.clientSecret},
		"scope":         {"https://storage.azure.com/.default"},
	}.Encode()
	resp, err := a.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("token decode (AAD response: %s): %w", a.redactSecrets(string(body)), err)
	}
	if tr.AccessToken == "" {
		// Surface the AAD error (AADSTS...) instead of a bare "empty token" —
		// expired/invalid secrets are indistinguishable without it.
		return "", fmt.Errorf("empty token (AAD response: %s)", a.redactSecrets(string(body)))
	}

	a.token = tr.AccessToken
	a.tokenExp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if a.tokenCache != nil {
		a.tokenCache.Save(a.token, a.tokenExp)
	}
	return a.token, nil
}

// Range-parallel downloads: every GET goes through getRanged — the first
// range request doubles as size discovery (no separate HEAD), small blobs
// return complete in one shot, and anything above rangeGetThreshold splits
// into parallel ranges. This unifies entries, bundles, and manifests under
// one large-object policy: per-request latency stops being the prefetch
// bottleneck (measured: 13,578-entry direct prefetch was request-bound at
// ~170/s / 25MB/s aggregate with single-stream mid-band entries).
const rangeGetThreshold = 512 << 10
const rangeGetMaxParallel = 16

// getRanged downloads a blob with parallel HTTP range requests.
func (a *AzureBackend) getRanged(actionID string, token string) ([]byte, error) {
	// First ranged GET also discovers the total size from Content-Range.
	partSize := rangeGetThreshold
	req, _ := http.NewRequest("GET", a.blobURL(actionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")
	req.Header.Set("x-ms-range", fmt.Sprintf("bytes=0-%d", partSize-1))
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		log.Printf("GET %s: 404 (account=%s)", shortID(actionID), a.account)
		return nil, nil
	}
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		// Range on a zero-length blob: it exists and is empty.
		resp.Body.Close()
		return []byte{}, nil
	}
	if resp.StatusCode == http.StatusOK {
		// Service returned the full body (range covered everything, or
		// ranges unsupported): read it whole.
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		return b, err
	}
	if resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: HTTP %d: %s", shortID(actionID), resp.StatusCode, body)
	}
	first, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Parse total size: "bytes 0-N/TOTAL"
	var start, last, total int64
	if _, err := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &start, &last, &total); err != nil {
		return first, nil // fall back to partial — better than nothing
	}
	if total <= int64(partSize) {
		return first, nil
	}

	// Fetch remaining ranges in parallel.
	nParts := (total-1)/int64(partSize) + 1
	if nParts > rangeGetMaxParallel {
		// fewer, bigger parts
		nParts = rangeGetMaxParallel
		partSize = int((total-1)/nParts + 1)
	}
	parts := make([][]byte, nParts)
	startIdx := int64(0)
	if int64(len(first)) == int64(partSize) {
		parts[0] = first
		startIdx = 1
	} else {
		// The initial discovery request only fetched rangeGetThreshold bytes;
		// after resizing parts it no longer covers part 0's full range.
		// Refetch part 0 like every other part below.
		first = nil
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for i := startIdx; i < nParts; i++ {
		wg.Add(1)
		go func(i int64) {
			defer wg.Done()
			lo := i * int64(partSize)
			hi := lo + int64(partSize) - 1
			if hi >= total {
				hi = total - 1
			}
			r, _ := http.NewRequest("GET", a.blobURL(actionID), nil)
			r.Header.Set("Authorization", "Bearer "+token)
			r.Header.Set("x-ms-version", "2023-11-03")
			r.Header.Set("x-ms-range", fmt.Sprintf("bytes=%d-%d", lo, hi))
			rr, err := a.client.Do(r)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			defer rr.Body.Close()
			if rr.StatusCode != http.StatusPartialContent {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("range part %d: HTTP %d", i, rr.StatusCode)
				}
				mu.Unlock()
				return
			}
			b, err := io.ReadAll(rr.Body)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			if want := hi - lo + 1; int64(len(b)) != want {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("range part %d: short read %d/%d bytes", i, len(b), want)
				}
				mu.Unlock()
				return
			}
			parts[i] = b
		}(i)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	if int64(len(out)) != total {
		// size mismatch (concurrent rewrite?) — treat as corrupt, caller misses
		log.Printf("GET %s: range assembly size mismatch: got %d bytes, expected %d (account=%s)", shortID(actionID), len(out), total, a.account)
		return nil, nil
	}
	return out, nil
}

func (a *AzureBackend) Get(actionID string) ([]byte, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}
	// Universal range path: getRanged discovers size from the first range's
	// Content-Range (no HEAD round trip), returns small blobs whole, and
	// splits larger ones across parallel connections.
	b, err := a.getRanged(actionID, token)
	if err != nil {
		return nil, err
	}
	return maybeGunzip(b), nil
}

func (a *AzureBackend) Put(actionID string, body []byte) error {
	token, err := a.getToken()
	if err != nil {
		return err
	}
	body = gzipBytes(body)
	req, err := http.NewRequest("PUT", a.blobURL(actionID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("PUT %s: HTTP %d", shortID(actionID), resp.StatusCode)
	}
	return nil
}

// PutIfNotExist writes a blob only if it does not already exist (atomic).
// Returns false if another writer got there first (HTTP 409/412).
// Used for advisory compilation leases between concurrent runners.
func (a *AzureBackend) PutIfNotExist(actionID string, body []byte) (bool, error) {
	token, err := a.getToken()
	if err != nil {
		return false, err
	}
	req, err := http.NewRequest("PUT", a.blobURL(actionID), bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	req.Header.Set("If-None-Match", "*") // atomic: fail if blob exists

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return true, nil
	}
	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusPreconditionFailed {
		return false, nil // someone else holds it
	}
	return false, fmt.Errorf("PUT-if-none-match %s: HTTP %d", shortID(actionID), resp.StatusCode)
}

// DeleteKey is intentionally NOT used for lease release: deleting the lease
// right after an entry upload would let late missers (whose GET raced the
// upload) acquire and recompile redundantly instead of waiting-and-hitting.
// Lease blobs are GC'd server-side by a lifecycle rule (see lease.go).

// ListKeys returns all blob names in the container (paginated).
func (a *AzureBackend) ListKeys() ([]string, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}

	var keys []string
	marker := ""
	for {
		url := a.containerURL() + "?restype=container&comp=list&maxresults=5000"
		if marker != "" {
			url += "&marker=" + marker
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-ms-version", "2023-11-03")

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("list: HTTP %d", resp.StatusCode)
		}

		var listResp struct {
			Blobs []struct {
				Name string `xml:"Name"`
			} `xml:"Blobs>Blob"`
			NextMarker string `xml:"NextMarker"`
		}
		if err := xml.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return nil, fmt.Errorf("list decode: %w", err)
		}
		for _, b := range listResp.Blobs {
			keys = append(keys, b.Name)
		}
		if listResp.NextMarker == "" {
			break
		}
		marker = listResp.NextMarker
	}
	return keys, nil
}

func (a *AzureBackend) Close() error { return nil }

func shortID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

func newBackend() (Backend, error) {
	switch os.Getenv("GOCACHE_SERVE_BACKEND") {
	case "", "azure":
		return newAzureBackend()
	case "dir":
		return newDirBackend()
	case "noop":
		return &NoopBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown backend (azure, dir, noop)")
	}
}
