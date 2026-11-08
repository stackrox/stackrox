package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Local disk cache manager
type LocalCache struct {
	dir string
	mu  sync.RWMutex
}

func NewLocalCache(dir string) (*LocalCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &LocalCache{dir: dir}, nil
}

func (lc *LocalCache) path(actionID string) string {
	if len(actionID) < 2 {
		return filepath.Join(lc.dir, actionID)
	}
	return filepath.Join(lc.dir, actionID[:2], actionID[2:])
}

func (lc *LocalCache) metaPath(actionID string) string {
	return lc.path(actionID) + ".meta"
}

func (lc *LocalCache) Get(actionID string) ([]byte, []byte, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	body, err := os.ReadFile(lc.path(actionID))
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	outputID, _ := os.ReadFile(lc.metaPath(actionID))
	return body, outputID, nil
}

func (lc *LocalCache) Put(actionID string, outputID, body []byte) (string, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	p := lc.path(actionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, body, 0o644); err != nil {
		return "", err
	}
	if len(outputID) > 0 {
		if err := os.WriteFile(lc.metaPath(actionID), outputID, 0o644); err != nil {
			return "", err
		}
	}
	return p, nil
}

// Manifest tracks which cache entries a job uses, in request order.
type Manifest struct {
	mu      sync.Mutex
	entries []string
	jobName string
	// readOnly: if true, Upload() is a no-op. PR runs set this so they
	// benefit from the master manifest's prefetch without overwriting
	// the canonical working set with PR-specific entries (which produce
	// different ActionIDs due to different source content).
	readOnly bool
}

func NewManifest(jobName string) *Manifest {
	return &Manifest{jobName: jobName}
}

func (m *Manifest) Record(actionID string) {
	if m == nil || m.jobName == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, actionID)
}

func (m *Manifest) Upload(backend Backend) error {
	if m == nil || m.jobName == "" || len(m.entries) == 0 || m.readOnly {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	data := []byte(strings.Join(m.entries, "\n"))
	return backend.Put("manifest-"+m.jobName, data)
}

func DownloadManifest(backend Backend, jobName string) ([]string, error) {
	if jobName == "" {
		return nil, nil
	}
	data, err := backend.Get("manifest-" + jobName)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var result []string
	for _, e := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if e != "" {
			result = append(result, e)
		}
	}
	return result, nil
}

// AsyncUploader queues puts and uploads to cloud in background.
// Upload latency matters: entries under an active lease unblock followers
// on other runners, so lease-held entries are prioritized and drops are
// counted loudly (a dropped lease-held upload stalls followers).
type AsyncUploader struct {
	backend   Backend
	ch        chan uploadJob
	pri       chan uploadJob // lease-held entries: served before ch
	wg        sync.WaitGroup
	done      chan struct{}
	flushOnce sync.Once
	stats     UploadStats

	// onUploaded, if set, is invoked after each successful cloud Put
	// (used to release advisory compilation leases).
	onUploaded func(actionID string)

	dropped    int64 // jobs dropped because both queues were full
	droppedPri int64 // ...of which were lease-held (stalls followers)
}

type uploadJob struct {
	actionID string
	body     []byte
}

type UploadStats struct {
	mu        sync.Mutex
	uploaded  int64
	failed    int64
	bytesSent int64
}

func NewAsyncUploader(backend Backend, workers int) *AsyncUploader {
	if workers <= 0 {
		workers = 4
	}
	au := &AsyncUploader{
		backend: backend,
		ch:      make(chan uploadJob, 1000),
		done:    make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		au.wg.Add(1)
		go au.worker()
	}
	return au
}

func (au *AsyncUploader) worker() {
	defer au.wg.Done()
	for {
		// Lease-held (priority) uploads are served first: followers on
		// other runners are actively waiting on these entries, so their
		// latency converts directly into cross-runner hit rate.
		var job uploadJob
		var ok bool
		select {
		case job = <-au.pri:
			ok = true
		default:
			select {
			case job = <-au.pri:
				ok = true
			case job = <-au.ch:
				ok = true
			case <-au.done:
				return
			}
		}
		if !ok {
			continue
		}
		if err := au.backend.Put(job.actionID, job.body); err != nil {
			au.stats.mu.Lock()
			au.stats.failed++
			au.stats.mu.Unlock()
		} else {
			au.stats.mu.Lock()
			au.stats.uploaded++
			au.stats.bytesSent += int64(len(job.body))
			au.stats.mu.Unlock()
			if au.onUploaded != nil {
				au.onUploaded(job.actionID) // release lease, best-effort
			}
		}
	}
}

// QueuePri enqueues a lease-held upload on the priority lane.
func (au *AsyncUploader) QueuePri(actionID string, body []byte) {
	select {
	case au.pri <- uploadJob{actionID, body}:
	default:
		atomic.AddInt64(&au.droppedPri, 1)
		// A dropped priority upload stalls followers until their wait
		// budget expires — loud by design.
		log.Printf("gocache-serve: WARNING priority upload queue full, dropping %s (followers will recompile)",
			shortID(actionID))
	}
}

// QueueDepth reports pending upload jobs (both lanes).
func (au *AsyncUploader) QueueDepth() int {
	return len(au.ch) + len(au.pri)
}

// Stats returns current upload counters.
func (au *AsyncUploader) Stats() (uploaded, failed, dropped int64) {
	au.stats.mu.Lock()
	defer au.stats.mu.Unlock()
	return au.stats.uploaded, au.stats.failed, atomic.LoadInt64(&au.dropped)
}

func (au *AsyncUploader) Queue(actionID string, body []byte) {
	select {
	case au.ch <- uploadJob{actionID, body}:
	default:
		atomic.AddInt64(&au.dropped, 1)
	}
}

func (au *AsyncUploader) Flush() {
	au.flushOnce.Do(func() {
		close(au.done)
		au.wg.Wait()
		au.stats.mu.Lock()
		defer au.stats.mu.Unlock()
		log.Printf("gocache-serve: uploads: %d ok, %d failed, %d bytes",
			au.stats.uploaded, au.stats.failed, au.stats.bytesSent)
	})
}
