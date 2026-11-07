// GOCACHEPROG: cloud-backed Go build cache.
//
// Implements the Go 1.24+ GOCACHEPROG protocol (cmd/go/internal/cacheprog):
// the go command starts this program as a subprocess and communicates via
// newline-delimited JSON over stdin/stdout.
//
// Architecture:
//   - Local disk is the hot cache (fast reads, write-through)
//   - Cloud object storage is the persistent, shared store
//   - On "get": local disk first, then cloud (download to local on hit)
//   - On "put": write to local disk immediately, upload to cloud async
//   - On "close": flush pending uploads, exit
//
// Usage: GOCACHEPROG=/path/to/gocache-serve
//
// Environment:
//   GOCACHE_SERVE_DIR   - local cache directory (default: /tmp/gocache-serve)
//   GOCACHE_SERVE_BACKEND - "azure" | "gcs" | "s3" (default: "azure")
//   AZURE_STORAGE_ACCOUNT, AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET
//   GCS_BUCKET (for gcs), S3_BUCKET (for s3)
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Protocol types (mirrors cmd/go/internal/cacheprog)

type Cmd string

const (
	CmdPut   = Cmd("put")
	CmdGet   = Cmd("get")
	CmdClose = Cmd("close")
)

type Request struct {
	ID       int64  `json:",omitempty"`
	Command  Cmd    `json:",omitempty"`
	ActionID []byte `json:",omitempty"`
	OutputID []byte `json:",omitempty"`
	BodySize int64  `json:",omitempty"`
}

type Response struct {
	ID            int64      `json:",omitempty"`
	Err           string     `json:",omitempty"`
	KnownCommands []Cmd      `json:",omitempty"`
	Miss          bool       `json:",omitempty"`
	OutputID      []byte     `json:",omitempty"`
	Size          int64      `json:",omitempty"`
	Time          *time.Time `json:",omitempty"`
	DiskPath      string     `json:",omitempty"`
}

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
	// Use first 2 chars as subdirectory (like Go's default GOCACHE)
	if len(actionID) < 2 {
		return filepath.Join(lc.dir, actionID)
	}
	return filepath.Join(lc.dir, actionID[:2], actionID[2:])
}

func (lc *LocalCache) Get(actionID string) ([]byte, error) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	data, err := os.ReadFile(lc.path(actionID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (lc *LocalCache) Put(actionID string, body []byte) (string, error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	p := lc.path(actionID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, body, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// Async uploader: queues puts and uploads to cloud in background
type AsyncUploader struct {
	backend Backend
	ch      chan uploadJob
	wg      sync.WaitGroup
	done    chan struct{}
	stats   UploadStats
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
		select {
		case job := <-au.ch:
			if err := au.backend.Put(job.actionID, job.body); err != nil {
				au.stats.mu.Lock()
				au.stats.failed++
				au.stats.mu.Unlock()
				log.Printf("gocache-serve: upload %s failed: %v", job.actionID[:8], err)
			} else {
				au.stats.mu.Lock()
				au.stats.uploaded++
				au.stats.bytesSent += int64(len(job.body))
				au.stats.mu.Unlock()
			}
		case <-au.done:
			return
		}
	}
}

func (au *AsyncUploader) Queue(actionID string, body []byte) {
	select {
	case au.ch <- uploadJob{actionID, body}:
	default:
		// Queue full — skip upload (local cache still has it)
		log.Printf("gocache-serve: upload queue full, skipping %s", actionID[:8])
	}
}

func (au *AsyncUploader) Flush() {
	close(au.done)
	au.wg.Wait()
	au.stats.mu.Lock()
	defer au.stats.mu.Unlock()
	log.Printf("gocache-serve: uploads: %d ok, %d failed, %d bytes",
		au.stats.uploaded, au.stats.failed, au.stats.bytesSent)
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("gocache-serve: ")

	// Setup
	cacheDir := os.Getenv("GOCACHE_SERVE_DIR")
	if cacheDir == "" {
		cacheDir = "/tmp/gocache-serve"
	}

	backend, err := newBackend()
	if err != nil {
		log.Fatalf("backend init: %v", err)
	}

	local, err := NewLocalCache(cacheDir)
	if err != nil {
		log.Fatalf("local cache init: %v", err)
	}

	uploader := NewAsyncUploader(backend, 4)
	defer uploader.Flush()

	// Protocol: send capabilities first
	stdout := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(stdout)

	if err := enc.Encode(Response{
		ID:            0,
		KnownCommands: []Cmd{CmdGet, CmdPut, CmdClose},
	}); err != nil {
		log.Fatalf("send capabilities: %v", err)
	}
	stdout.Flush()

	// Read requests from stdin
	stdin := bufio.NewReader(os.Stdin)
	dec := json.NewDecoder(stdin)

	var totalGets, totalHits, totalPuts int64
	startTime := time.Now()

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("decode error: %v", err)
			break
		}

		var resp Response
		resp.ID = req.ID

		switch req.Command {
		case CmdGet:
			totalGets++
			actionID := fmt.Sprintf("%x", req.ActionID)

			// Try local disk first
			body, err := local.Get(actionID)
			if err != nil {
				resp.Err = fmt.Sprintf("local get: %v", err)
				break
			}

			if body != nil {
				// Local hit
				totalHits++
				resp.DiskPath = local.path(actionID)
				resp.Size = int64(len(body))
				resp.OutputID = req.ActionID // echo back what we know
				break
			}

			// Try cloud
			body, err = backend.Get(actionID)
			if err != nil {
				log.Printf("cloud get %s: %v", actionID[:8], err)
				resp.Miss = true
				break
			}
			if body == nil {
				// Cloud miss
				resp.Miss = true
				break
			}

			// Cloud hit — download to local for future access
			path, err := local.Put(actionID, body)
			if err != nil {
				resp.Err = fmt.Sprintf("local put after cloud get: %v", err)
				break
			}
			totalHits++
			resp.DiskPath = path
			resp.Size = int64(len(body))
			resp.OutputID = req.ActionID

		case CmdPut:
			totalPuts++
			actionID := fmt.Sprintf("%x", req.ActionID)

			// Read body (base64-encoded JSON string on next line)
			var body []byte
			if req.BodySize > 0 {
				var bodyStr string
				if err := dec.Decode(&bodyStr); err != nil {
					resp.Err = fmt.Sprintf("decode body: %v", err)
					break
				}
				// bodyStr is base64
			body, err = base64.StdEncoding.DecodeString(bodyStr)
				if err != nil {
					resp.Err = fmt.Sprintf("decode base64 body: %v", err)
					break
				}
			}

			// Write to local disk
			path, err := local.Put(actionID, body)
			if err != nil {
				resp.Err = fmt.Sprintf("local put: %v", err)
				break
			}
			resp.DiskPath = path
			resp.Size = req.BodySize

			// Queue for async cloud upload
			if len(body) > 0 {
				uploader.Queue(actionID, body)
			}

		case CmdClose:
			// Flush pending uploads and exit
			uploader.Flush()
			elapsed := time.Since(startTime)
			log.Printf("closing: %d gets (%d hits, %.1f%% hit rate), %d puts, elapsed %v",
				totalGets, totalHits,
				float64(totalHits)/float64(totalGets+1)*100,
				totalPuts, elapsed.Round(time.Millisecond))
			resp.ID = req.ID
			if err := enc.Encode(resp); err != nil {
				log.Printf("send close response: %v", err)
			}
			stdout.Flush()
			return

		default:
			resp.Err = fmt.Sprintf("unknown command: %s", req.Command)
		}

		if err := enc.Encode(resp); err != nil {
			log.Printf("send response: %v", err)
			break
		}
		stdout.Flush()
	}
}
