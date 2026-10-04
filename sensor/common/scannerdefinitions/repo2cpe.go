package scannerdefinitions

import (
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
)

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
	_ common.SensorComponent = (*Repo2CPE)(nil)

	errStartMoreThanOnce = errors.New("repo-to-CPE refresher already started")
)

// Repo2CPE caches Central's repo-to-CPE mapping and refreshes it in the
// background. Sensor starts and stops it like any other component.
type Repo2CPE struct {
	unimplemented.Receiver

	centralClient    *http.Client
	centralReachable atomic.Bool
	started          atomic.Bool
	cancel           context.CancelFunc

	timerMu sync.Mutex
	timer   *time.Timer

	cacheMu sync.Mutex
	cache   repo2CPECache
}

// repo2CPECache is Sensor's in-process copy of Central's repo-to-CPE mapping,
// independent of Handler's per-request proxying, plus enough state to issue
// a conditional GET and to tell "never fetched" apart from "stale".
type repo2CPECache struct {
	mapping      []byte
	hash         string
	etag         string
	lastModified string
	lastSuccess  time.Time
}

// NewRepo2CPE creates a refresher that fetches Central's repo-to-CPE mapping.
func NewRepo2CPE(centralEndpoint string, centralCertificates []*x509.Certificate) (*Repo2CPE, error) {
	client, err := centralclient.AuthenticatedCentralHTTPClient(centralEndpoint, centralCertificates)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating central HTTP transport")
	}
	return newRepo2CPE(client), nil
}

func newRepo2CPE(client *http.Client) *Repo2CPE {
	return &Repo2CPE{centralClient: client}
}

func (r *Repo2CPE) Name() string { return "scannerdefinitions.Repo2CPE" }

func (r *Repo2CPE) Capabilities() []centralsensor.SensorCapability { return nil }

func (r *Repo2CPE) ResponsesC() <-chan *message.ExpiringMessage { return nil }

func (r *Repo2CPE) Start() error {
	if r.started.Swap(true) {
		return errStartMoreThanOnce
	}
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.NewTimer(0)
	concurrency.WithLock(&r.timerMu, func() {
		r.cancel = cancel
		r.timer = timer
	})
	go r.run(ctx, timer.C)
	return nil
}

func (r *Repo2CPE) Stop() {
	concurrency.WithLock(&r.timerMu, func() {
		if r.cancel != nil {
			r.cancel()
		}
		if r.timer != nil {
			r.timer.Stop()
		}
	})
}

// Notify kicks an immediate fetch when Central becomes reachable, so a cold
// cache does not wait out the current timer.
func (r *Repo2CPE) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e, r.Name()))
	switch e {
	case common.SensorComponentEventCentralReachable:
		r.centralReachable.Store(true)
		r.resetTimer(0)
	case common.SensorComponentEventOfflineMode:
		r.centralReachable.Store(false)
	}
}

func (r *Repo2CPE) resetTimer(d time.Duration) {
	concurrency.WithLock(&r.timerMu, func() {
		if r.timer != nil {
			r.timer.Reset(d)
		}
	})
}

// run fetches on each receive from ticks, then rearms the component timer.
func (r *Repo2CPE) run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			if ctx.Err() != nil {
				return
			}
			if !r.centralReachable.Load() {
				r.resetTimer(repo2CPERetryInterval)
				continue
			}
			fetchCtx, cancel := context.WithTimeout(ctx, repo2CPEFetchTimeout)
			ok := r.attemptRepo2CPERefresh(fetchCtx)
			cancel()
			if ctx.Err() != nil {
				return
			}
			delay := repo2CPERetryInterval
			if ok {
				delay = repo2CPERefreshInterval
			}
			r.resetTimer(delay)
		}
	}
}

// FetchRepo2CPE returns a copy of the cached mapping. ok is false only
// when Central has never been fetched successfully; unreachable Central
// still yields the last-good copy.
func (r *Repo2CPE) FetchRepo2CPE(_ context.Context) (mapping []byte, hash string, ok bool) {
	return concurrency.WithLock3(&r.cacheMu, func() ([]byte, string, bool) {
		if r.cache.lastSuccess.IsZero() {
			return nil, "", false
		}
		return slices.Clone(r.cache.mapping), r.cache.hash, true
	})
}

// attemptRepo2CPERefresh issues one conditional GET for the repo-to-CPE
// mapping and updates the cache on success or "unchanged". It reports
// whether the attempt succeeded, so the caller can pick the next delay.
func (r *Repo2CPE) attemptRepo2CPERefresh(ctx context.Context) bool {
	result := repoCPEMappingFetchError
	defer func() {
		repoCPEMappingFetch.WithLabelValues(result).Inc()
	}()

	etag, lastModified := concurrency.WithLock2(&r.cacheMu, func() (string, string) {
		return r.cache.etag, r.cache.lastModified
	})

	req, err := r.newRepo2CPERequest(ctx, etag, lastModified)
	if err != nil {
		log.Warnf("Failed to build repo-to-CPE mapping request: %v", err)
		return false
	}

	resp, err := r.centralClient.Do(req)
	if err != nil {
		log.Warnf("Failed to fetch repo-to-CPE mapping from central: %v", err)
		return false
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusNotModified:
		r.recordRepo2CPEUnchanged(resp)
		result = repoCPEMappingFetchUnchanged
		return true
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, cpemapping.MaxMappingBytes+1))
		if err != nil {
			log.Warnf("Failed to read repo-to-CPE mapping response: %v", err)
			return false
		}
		if err := cpemapping.ValidateMapping(body); err != nil {
			log.Warnf("Repo-to-CPE mapping failed validation, keeping last-good: %v", err)
			return false
		}
		if r.recordRepo2CPESuccess(body, resp) {
			result = repoCPEMappingFetchSuccess
		} else {
			result = repoCPEMappingFetchUnchanged
		}
		return true
	default:
		log.Warnf("Unexpected status %d fetching repo-to-CPE mapping from central", resp.StatusCode)
		return false
	}
}

func (r *Repo2CPE) newRepo2CPERequest(ctx context.Context, etag, lastModified string) (*http.Request, error) {
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

func (r *Repo2CPE) recordRepo2CPEUnchanged(resp *http.Response) {
	concurrency.WithLock(&r.cacheMu, func() {
		r.markRepo2CPESuccessNoLock()
		r.mergeRepo2CPEValidatorsNoLock(resp)
	})
	log.Debug("Repo-to-CPE mapping unchanged (304)")
}

// recordRepo2CPESuccess keeps the existing cached bytes when the new content
// hashes the same as what's cached, treating a same-hash 200 like a 304.
func (r *Repo2CPE) recordRepo2CPESuccess(body []byte, resp *http.Response) bool {
	hash := cpemapping.HashMapping(body)
	var oldHash string
	var changed bool
	concurrency.WithLock(&r.cacheMu, func() {
		r.markRepo2CPESuccessNoLock()
		oldHash = r.cache.hash
		if hash != r.cache.hash {
			r.cache.mapping = body
			r.cache.hash = hash
			changed = true
		}
		r.replaceRepo2CPEValidatorsNoLock(resp)
	})
	if changed {
		repoCPEMappingHashChanges.Inc()
		log.Infof("Repo-to-CPE mapping content hash changed: old=%q new=%q", oldHash, hash)
	}
	log.Debugf("Fetched repo-to-CPE mapping from central, hash=%s", hash)
	return changed
}

// markRepo2CPESuccessNoLock publishes lastSuccess to the last-success gauge
// in the same critical section so scrapes cannot observe a stale timestamp.
func (r *Repo2CPE) markRepo2CPESuccessNoLock() {
	now := time.Now()
	r.cache.lastSuccess = now
	repo2CPELastSuccessUnix.Store(now.Unix())
}

// replaceRepo2CPEValidatorsNoLock copies validators from a 200, including
// empty values, so the next conditional GET cannot reuse a prior mapping's
// ETag or Last-Modified.
func (r *Repo2CPE) replaceRepo2CPEValidatorsNoLock(resp *http.Response) {
	r.cache.etag = resp.Header.Get(etagHeader)
	r.cache.lastModified = resp.Header.Get(lastModifiedHeader)
}

// mergeRepo2CPEValidatorsNoLock applies non-empty validators from a 304,
// which may omit them.
func (r *Repo2CPE) mergeRepo2CPEValidatorsNoLock(resp *http.Response) {
	if etag := resp.Header.Get(etagHeader); etag != "" {
		r.cache.etag = etag
	}
	if lm := resp.Header.Get(lastModifiedHeader); lm != "" {
		r.cache.lastModified = lm
	}
}
