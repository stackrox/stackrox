// Remote tier for the gocache proxy: Azure Blob storage via plain
// net/http (zero dependencies), ported from the stackrox CA work's
// gocache-serve backend.
//
// Cloud layout (content-addressed, mirroring the local one):
//
//	d/<outputIDhex>   output data blob (dedups across actions)
//	a/<actionIDhex>   JSON {"output_id":...,"size":...} mapping
//
// GET path: local first, then cloud; cloud bodies are verified
// against their OutputID (sha256) before entering the local cache —
// a corrupt or poisoned blob degrades to a miss, never to wrong
// build output (the CA-work integrity lesson).
//
// PUT path: local synchronously (the build needs it), cloud
// asynchronously (best-effort, flushed on close). Last-write-wins is
// benign: keys are content hashes.
//
// Auth: SPN client credentials (AZURE_TENANT_ID / AZURE_CLIENT_ID /
// AZURE_CLIENT_SECRET — the CI path), or a pre-minted bearer token in
// CIMEMO_AZURE_TOKEN (the `az account get-access-token` dev path).
package gocache

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Remote is the cloud backend. A nil *Remote means "local only".
type Remote struct {
	account   string
	container string
	tenantID  string
	clientID  string
	secret    string
	staticTok string

	client *http.Client

	tokenMu   sync.Mutex
	token     string
	tokenExp  time.Time
	upMu      sync.Mutex
	uploadQ   []func()
	uploaded  int64
	uploadErr int64
	lastErr   string
}

// NewRemoteFromEnv returns a Remote when remote mode is enabled
// (CIMEMO_GOCACHE_REMOTE=1), else nil.
func NewRemoteFromEnv() *Remote {
	if os.Getenv("CIMEMO_GOCACHE_REMOTE") != "1" {
		return nil
	}
	r := &Remote{
		account:   envOr("CIMEMO_AZURE_ACCOUNT", "cigocache"),
		container: envOr("CIMEMO_GOCACHE_CONTAINER", "cimemo"),
		tenantID:  os.Getenv("AZURE_TENANT_ID"),
		clientID:  os.Getenv("AZURE_CLIENT_ID"),
		secret:    os.Getenv("AZURE_CLIENT_SECRET"),
		staticTok: os.Getenv("CIMEMO_AZURE_TOKEN"),
		client: &http.Client{
			Timeout: 30 * time.Second,
			// Default Transport keeps only 2 idle conns per host — with
			// hundreds of concurrent action GETs that serializes to ~2
			// connections and the remote tier crawls (measured 47 req/s
			// despite unbounded goroutines). Pool generously instead.
			Transport: &http.Transport{
				MaxIdleConns:        128,
				MaxIdleConnsPerHost: 128,
				MaxConnsPerHost:     0,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
	return r
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (r *Remote) containerURL() string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s", r.account, r.container)
}

func (r *Remote) getToken() (string, error) {
	if r.staticTok != "" {
		return r.staticTok, nil
	}
	r.tokenMu.Lock()
	defer r.tokenMu.Unlock()
	if r.token != "" && time.Now().Before(r.tokenExp.Add(-time.Minute)) {
		return r.token, nil
	}
	if r.tenantID == "" || r.clientID == "" || r.secret == "" {
		return "", fmt.Errorf("remote: no credentials (set AZURE_TENANT_ID/CLIENT_ID/CLIENT_SECRET or CIMEMO_AZURE_TOKEN)")
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", r.tenantID)
	// url.Values.Encode, NOT raw Sprintf: secrets contain '+&%' that
	// raw interpolation would corrupt (port note from gocache-serve).
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {r.clientID},
		"client_secret": {r.secret},
		"scope":         {"https://storage.azure.com/.default"},
	}.Encode()
	resp, err := r.client.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", fmt.Errorf("remote token: %w", err)
	}
	defer resp.Body.Close()
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		redacted := strings.ReplaceAll(string(body), r.secret, "***")
		return "", fmt.Errorf("remote token failed (AAD: %s)", redacted)
	}
	r.token, r.tokenExp = tr.AccessToken, time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second)
	return r.token, nil
}

// Get returns the blob at key, or nil,nil when absent.
func (r *Remote) Get(key string) ([]byte, error) {
	token, err := r.getToken()
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest("GET", r.containerURL()+"/"+key, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-version", "2023-11-03")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote GET %s: HTTP %d", key, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return maybeGunzip(body), nil
}

// put uploads body at key (gzip at rest), last-write-wins.
func (r *Remote) put(key string, body []byte) error {
	token, err := r.getToken()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write(body)
	zw.Close()
	req, err := http.NewRequest("PUT", r.containerURL()+"/"+key, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("remote PUT %s: HTTP %d", key, resp.StatusCode)
	}
	return nil
}

// putIfNotExist writes only when absent; returns false if it exists.
func (r *Remote) putIfNotExist(key string, body []byte) (bool, error) {
	token, err := r.getToken()
	if err != nil {
		return false, err
	}
	req, err := http.NewRequest("PUT", r.containerURL()+"/"+key, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-version", "2023-11-03")
	req.Header.Set("If-None-Match", "*")
	resp, err := r.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated:
		return true, nil
	case http.StatusConflict, http.StatusPreconditionFailed:
		return false, nil
	default:
		return false, fmt.Errorf("remote PUT-if-none-match %s: HTTP %d", key, resp.StatusCode)
	}
}

// remoteGetAction fetches an action from the cloud and, if valid,
// writes it into the local standard layout. Returns ok=false on any
// absence, error, or integrity failure (degrades to a local miss).
// The data blob STREAMS to disk through a hasher — constant memory
// regardless of size (the CA-work OOM lesson). Actions known absent
// (negatives) skip the cloud entirely.
func remoteGetAction(r *Remote, cacheDir, actionIDHex string, neg *Negatives, tt *TimeoutTracker) bool {
	if neg.Has(actionIDHex) {
		return false
	}
	rec, err := r.Get("a/" + actionIDHex)
	if err != nil || rec == nil {
		if rec == nil {
			neg.Add(actionIDHex) // known absent: stop re-probing
		}
		return false
	}
	var ar struct {
		OutputID string `json:"output_id"`
		Size     int64  `json:"size"`
	}
	if json.Unmarshal(rec, &ar) != nil || len(ar.OutputID) != 64 {
		return false
	}
	dest := dataPath(cacheDir, ar.OutputID)
	if _, err := os.Stat(dest); err == nil {
		// Data already local (e.g. concurrent fetch) — just need the record.
		writeActionRecord(cacheDir, actionIDHex, ar.OutputID, ar.Size)
		neg.Clear(actionIDHex)
		return true
	}
	size, ok := r.getStreamingVerify("d/"+ar.OutputID, ar.OutputID, dest, tt)
	if !ok {
		return false
	}
	if ar.Size > 0 && size != ar.Size {
		os.Remove(dest)
		return false
	}
	writeActionRecord(cacheDir, actionIDHex, ar.OutputID, size)
	neg.Clear(actionIDHex)
	return true
}

// queueCloudPut schedules an async upload of an action's data file to
// the cloud. Streams from the local cache file (constant memory —
// no body buffering for large link outputs).
func queueCloudPut(r *Remote, actionIDHex, outputIDHex, dataFilePath string, size int64) {
	r.upMu.Lock()
	r.uploadQ = append(r.uploadQ, func() {
		if dataFilePath != "" {
			if _, err := r.putFileIfNotExistGzip("d/"+outputIDHex, dataFilePath); err != nil {
				r.upMu.Lock()
				r.uploadErr++
				r.lastErr = err.Error()
				r.upMu.Unlock()
				return
			}
		}
		rec, _ := json.Marshal(struct {
			OutputID string `json:"output_id"`
			Size     int64  `json:"size"`
		}{outputIDHex, size})
		if err := r.put("a/"+actionIDHex, rec); err != nil {
			r.upMu.Lock()
			r.uploadErr++
			r.lastErr = err.Error()
			r.upMu.Unlock()
			return
		}
		r.upMu.Lock()
		r.uploaded++
		r.upMu.Unlock()
	})
	r.upMu.Unlock()
}

// flushUploadsConcurrent drains the queue with bounded parallelism.
// Sequential draining measured 476s for a 3,654-action Prometheus
// build (2 PUTs per action at ~65ms each) — the upload must not be
// the tail latency of a build that just finished.
func (r *Remote) flushUploadsConcurrent() {
	r.upMu.Lock()
	q := r.uploadQ
	r.uploadQ = nil
	r.upMu.Unlock()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 16)
	for _, f := range q {
		wg.Add(1)
		sem <- struct{}{}
		go func(f func()) {
			defer wg.Done()
			defer func() { <-sem }()
			f()
		}(f)
	}
	wg.Wait()
}

func (r *Remote) uploadCountsWithDetail() (uploaded, failed int64, lastErr string) {
	r.upMu.Lock()
	defer r.upMu.Unlock()
	return r.uploaded, r.uploadErr, r.lastErr
}

func maybeGunzip(body []byte) []byte {
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return body // not gzip (legacy/raw)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		return body
	}
	return out
}
