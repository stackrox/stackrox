package scannerdefinitions

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"google.golang.org/grpc/codes"
)

const scannerDefsPath = "/api/extensions/scannerdefinitions"

const (
	// repo2CPEFileParam matches sensorMappingsFile in compliance/node/index/indexer.go.
	repo2CPEFileParam = "repo2cpe"

	// repo2CPERefreshInterval is the steady-state cadence other repo2cpe consumers use.
	repo2CPERefreshInterval = 4 * time.Hour
	// repo2CPERetryInterval is short so a cold cache reaches its first success quickly.
	repo2CPERetryInterval = time.Minute
	repo2CPEFetchTimeout  = 30 * time.Second

	etagHeader            = "ETag"
	ifNoneMatchHeader     = "If-None-Match"
	lastModifiedHeader    = "Last-Modified"
	ifModifiedSinceHeader = "If-Modified-Since"
)

var (
	headersToProxy = set.NewFrozenStringSet("If-Modified-Since", "Accept-Encoding")
	log            = logging.LoggerForModule()
)

// Handler handles requests to retrieve scanner definitions
// from Central.
type Handler struct {
	centralClient    *http.Client
	centralReachable atomic.Bool

	refreshOnce sync.Once
	cacheMu     sync.Mutex
	cache       repo2CPECache
}

// repo2CPECache is Sensor's in-process copy of Central's repo-to-CPE mapping,
// independent of ServeHTTP's per-request proxying, plus enough state to issue
// a conditional GET and to tell "never fetched" apart from "stale".
type repo2CPECache struct {
	mapping      []byte
	hash         string
	etag         string
	lastModified string
	lastAttempt  time.Time
	lastSuccess  time.Time
}

// NewDefinitionsHandler creates a new scanner definitions handler.
func NewDefinitionsHandler(centralEndpoint string, centralCertificates []*x509.Certificate) (*Handler, error) {
	client, err := centralclient.AuthenticatedCentralHTTPClient(centralEndpoint, centralCertificates)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return &Handler{
		centralClient: client,
	}, nil
}

// Notify reacts to sensor going into online/offline mode.
func (h *Handler) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, "Scanner definitions handler"))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReachable.Store(true)
		h.startRepo2CPERefresh()
	case common.SensorComponentEventOfflineMode:
		h.centralReachable.Store(false)
	}
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Validate request.
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// If central is not reachable, then the request should return an error to Scanner.
	if !h.centralReachable.Load() {
		httputil.WriteGRPCStyleErrorf(writer, codes.Unavailable, "central not reachable")
		return
	}

	// Prepare the Central's request, proxy relevant headers and all parameters.
	// No need to set Scheme nor Host, as the client will already do that for us.
	centralURL := url.URL{
		Path:     scannerDefsPath,
		RawQuery: request.URL.RawQuery,
	}
	centralRequest, err := http.NewRequestWithContext(
		request.Context(), http.MethodGet, centralURL.String(), nil)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to create request: %v", err)
		return
	}
	// Proxy relevant headers.
	for headerName := range headersToProxy.All() {
		for _, value := range request.Header.Values(headerName) {
			centralRequest.Header.Add(headerName, value)
		}
	}
	// Do request, copy all response headers, and body.
	resp, err := h.centralClient.Do(centralRequest)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed to contact central: %v", err)
		return
	}
	defer utils.IgnoreError(resp.Body.Close)
	for k, vs := range resp.Header {
		for _, v := range vs {
			writer.Header().Add(k, v)
		}
	}
	writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(writer, codes.Internal, "failed write response: %v", err)
		return
	}
}

// FetchRepo2CPE returns Sensor's cached repo-to-CPE mapping for in-process
// callers such as vmscraper, starting the background refresh loop on first
// call. ok is false only if Central has never been fetched successfully; a
// copy served while central is unreachable is still ok, just possibly stale.
func (h *Handler) FetchRepo2CPE(_ context.Context) (mapping []byte, hash string, ok bool) {
	h.startRepo2CPERefresh()

	return concurrency.WithLock3(&h.cacheMu, func() ([]byte, string, bool) {
		if h.cache.lastSuccess.IsZero() {
			return nil, "", false
		}
		if !h.centralReachable.Load() {
			log.Info("Serving stale repo-to-CPE mapping: central is unreachable")
		}
		return h.cache.mapping, h.cache.hash, true
	})
}

// startRepo2CPERefresh launches the background refresh loop exactly once,
// however many times Notify or FetchRepo2CPE end up triggering it.
func (h *Handler) startRepo2CPERefresh() {
	h.refreshOnce.Do(func() {
		go h.refreshRepo2CPELoop()
	})
}

// refreshRepo2CPELoop refetches after repo2CPERefreshInterval on success, or
// after repo2CPERetryInterval on failure or while offline, so a cold cache
// doesn't wait hours for its first successful fetch.
func (h *Handler) refreshRepo2CPELoop() {
	for {
		delay := repo2CPERetryInterval
		if h.centralReachable.Load() {
			ctx, cancel := context.WithTimeout(context.Background(), repo2CPEFetchTimeout)
			ok := h.attemptRepo2CPERefresh(ctx)
			cancel()
			if ok {
				delay = repo2CPERefreshInterval
			}
		}
		time.Sleep(delay)
	}
}

// attemptRepo2CPERefresh issues one conditional GET for the repo-to-CPE
// mapping and updates the cache on success or "unchanged". It reports
// whether the attempt succeeded, so the caller can pick the next delay.
func (h *Handler) attemptRepo2CPERefresh(ctx context.Context) bool {
	etag, lastModified := concurrency.WithLock2(&h.cacheMu, func() (string, string) {
		return h.cache.etag, h.cache.lastModified
	})

	req, err := h.newRepo2CPERequest(ctx, etag, lastModified)
	if err != nil {
		log.Warnf("Failed to build repo-to-CPE mapping request: %v", err)
		h.recordRepo2CPEAttempt(false)
		return false
	}

	resp, err := h.centralClient.Do(req)
	if err != nil {
		log.Warnf("Failed to fetch repo-to-CPE mapping from central: %v", err)
		h.recordRepo2CPEAttempt(false)
		return false
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusNotModified:
		h.recordRepo2CPEUnchanged(resp)
		return true
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, cpemapping.MaxMappingBytes+1))
		if err != nil {
			log.Warnf("Failed to read repo-to-CPE mapping response: %v", err)
			h.recordRepo2CPEAttempt(false)
			return false
		}
		if len(body) > cpemapping.MaxMappingBytes {
			log.Warnf("Repo-to-CPE mapping response exceeds %d bytes, rejecting", cpemapping.MaxMappingBytes)
			h.recordRepo2CPEAttempt(false)
			return false
		}
		h.recordRepo2CPESuccess(body, resp)
		return true
	default:
		log.Warnf("Unexpected status %d fetching repo-to-CPE mapping from central", resp.StatusCode)
		h.recordRepo2CPEAttempt(false)
		return false
	}
}

func (h *Handler) newRepo2CPERequest(ctx context.Context, etag, lastModified string) (*http.Request, error) {
	centralURL := url.URL{
		Path:     scannerDefsPath,
		RawQuery: "file=" + repo2CPEFileParam,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, centralURL.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}
	if etag != "" {
		req.Header.Set(ifNoneMatchHeader, etag)
	}
	if lastModified != "" {
		req.Header.Set(ifModifiedSinceHeader, lastModified)
	}
	return req, nil
}

func (h *Handler) recordRepo2CPEAttempt(success bool) {
	concurrency.WithLock(&h.cacheMu, func() {
		h.cache.lastAttempt = time.Now()
		if success {
			h.cache.lastSuccess = h.cache.lastAttempt
		}
	})
}

func (h *Handler) recordRepo2CPEUnchanged(resp *http.Response) {
	concurrency.WithLock(&h.cacheMu, func() {
		h.cache.lastAttempt = time.Now()
		h.cache.lastSuccess = h.cache.lastAttempt
		h.updateRepo2CPEValidatorsLocked(resp)
	})
}

// recordRepo2CPESuccess keeps the existing cached bytes when the new content
// hashes the same as what's cached, treating a same-hash 200 like a 304.
func (h *Handler) recordRepo2CPESuccess(body []byte, resp *http.Response) {
	hash := cpemapping.HashMapping(body)
	concurrency.WithLock(&h.cacheMu, func() {
		h.cache.lastAttempt = time.Now()
		h.cache.lastSuccess = h.cache.lastAttempt
		if hash != h.cache.hash {
			h.cache.mapping = body
			h.cache.hash = hash
		}
		h.updateRepo2CPEValidatorsLocked(resp)
	})
}

// updateRepo2CPEValidatorsLocked refreshes the cached conditional-GET
// validators from resp. Callers must hold h.cacheMu.
func (h *Handler) updateRepo2CPEValidatorsLocked(resp *http.Response) {
	if etag := resp.Header.Get(etagHeader); etag != "" {
		h.cache.etag = etag
	}
	if lm := resp.Header.Get(lastModifiedHeader); lm != "" {
		h.cache.lastModified = lm
	}
}
