package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

// storageAccountForRegion selects the nearest storage account for a runner region.
// Three replicas, westus3 as the replication hub (west<->east bidirectional,
// west->central one-way). Coverage of observed runner distribution (1900+
// samples): eastus 33%, centralus 27%, westus2+westus+northwestus ~34%.
func storageAccountForRegion(region string) string {
	switch region {
	case "eastus", "eastus2", "northcentralus", "CanadaEast",
		"westeurope", "northeurope", "uk south", "francecentral":
		return "cigocacheeast" // eastus replica
	case "centralus", "southcentralus", "northcentralus2":
		return "cigochachecentral" // centralus replica
	case "westus3", "westus2", "westus", "northwestus",
		"eastasia", "southeastasia", "japaneast", "japanwest",
		"australiaeast", "australiasoutheast":
		return "cigocache" // westus3 primary
	default:
		return "cigocache" // fallback to primary
	}
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
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", a.account, a.container)
}

func (a *AzureBackend) blobURL(actionID string) string {
	return fmt.Sprintf("%s/%s", a.containerURL(), actionID)
}

// getToken fetches an OAuth token via client credentials, cached until expiry.
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
	form := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s&scope=https://storage.azure.com/.default",
		a.clientID, a.clientSecret,
	)
	resp, err := a.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("token decode: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty token")
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
		if int64(len(first)) >= int64(partSize) {
			first = first[:partSize]
		}
	}
	parts := make([][]byte, nParts)
	parts[0] = first
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for i := int64(1); i < nParts; i++ {
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
			b, err := io.ReadAll(rr.Body)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
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
