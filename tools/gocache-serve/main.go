package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	Body     []byte `json:"-"` // decoded from base64 line after JSON, for put
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

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("gocache-serve: ")

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

	jobName := os.Getenv("GOCACHE_SERVE_MANIFEST")
	manifest := NewManifest(jobName)

	// Prefetch from manifest (see bundle.go for BundledBackend)
	if os.Getenv("GOCACHE_SERVE_PREFETCH") != "0" && jobName != "" {
		go func() {
			start := time.Now()
			keys, err := DownloadManifest(backend, jobName)
			if err != nil {
				log.Printf("manifest download failed: %v", err)
				return
			}
			if len(keys) == 0 {
				log.Printf("no manifest for '%s', skipping prefetch (first run)", jobName)
				return
			}
			log.Printf("prefetching %d entries from manifest '%s'...", len(keys), jobName)
			sem := make(chan struct{}, 16)
			var wg sync.WaitGroup
			var downloaded int64
			for _, key := range keys {
				if strings.HasPrefix(key, "manifest-") || strings.HasPrefix(key, "bundle-") || key == "bundle-index" {
					continue
				}
				wg.Add(1)
				go func(key string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					if _, _, err := local.Get(key); err == nil {
						return
					}
					body, err := backend.Get(key)
					if err != nil || body == nil {
						return
					}
					local.Put(key, nil, body)
					atomic.AddInt64(&downloaded, 1)
				}(key)
			}
			wg.Wait()
			log.Printf("prefetch complete: %d/%d entries in %v",
				atomic.LoadInt64(&downloaded), len(keys), time.Since(start).Round(time.Millisecond))
		}()
	}

	// Protocol handshake
	stdout := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(stdout)
	if err := enc.Encode(Response{
		ID:            0,
		KnownCommands: []Cmd{CmdGet, CmdPut, CmdClose},
	}); err != nil {
		log.Fatalf("send capabilities: %v", err)
	}
	stdout.Flush()

	stdin := bufio.NewReader(os.Stdin)
	dec := json.NewDecoder(stdin)

	var totalGets, totalHits, totalPuts int64
	var getBytes, getNs int64
	startTime := time.Now()

	// Concurrent request processing: dispatch cloud GETs to goroutines.
	// The protocol allows responses in any order. Writes to stdout are
	// serialized via writeMu; reads from stdin happen only in the main
	// goroutine (PUT bodies are read before dispatch).
	var writeMu sync.Mutex
	var wg sync.WaitGroup

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("decode error: %v", err)
			break
		}

		// Read PUT body in the main goroutine (stdin is sequential)
		if req.Command == CmdPut && req.BodySize > 0 {
			var bodyStr string
			if err := dec.Decode(&bodyStr); err != nil {
				log.Printf("decode body: %v", err)
				continue
			}
			body, err := base64.StdEncoding.DecodeString(bodyStr)
			if err != nil {
				log.Printf("decode base64: %v", err)
				continue
			}
			req.Body = body
		}

		// Close must wait for all in-flight requests
		if req.Command == CmdClose {
			wg.Wait()
			uploader.Flush()
			if err := manifest.Upload(backend); err != nil {
				log.Printf("manifest upload failed: %v", err)
			}
			elapsed := time.Since(startTime)
			var netStats string
			if getNs > 0 {
				getMBps := float64(getBytes) / (float64(getNs) / 1e9) / 1048576
				netStats = fmt.Sprintf(" GET: %d bytes in %dms (%.1f MB/s)",
					getBytes, getNs/1e6, getMBps)
			}
			log.Printf("closing: %d gets (%d hits, %.1f%%), %d puts, elapsed %v%s",
				atomic.LoadInt64(&totalGets), atomic.LoadInt64(&totalHits),
				float64(atomic.LoadInt64(&totalHits))/float64(atomic.LoadInt64(&totalGets)+1)*100,
				atomic.LoadInt64(&totalPuts), elapsed.Round(time.Millisecond), netStats)
			// Send close response
			writeMu.Lock()
			enc.Encode(Response{ID: req.ID})
			stdout.Flush()
			writeMu.Unlock()
			return
		}

		// Dispatch to goroutine
		wg.Add(1)
		go func(req Request) {
			defer wg.Done()

			var resp Response
			resp.ID = req.ID

			switch req.Command {
			case CmdGet:
				atomic.AddInt64(&totalGets, 1)
				actionID := fmt.Sprintf("%x", req.ActionID)

				// Local cache first (no network)
				body, storedOutputID, err := local.Get(actionID)
				if err != nil {
					resp.Err = fmt.Sprintf("local get: %v", err)
					break
				}
				if body != nil {
					atomic.AddInt64(&totalHits, 1)
					resp.DiskPath = local.path(actionID)
					resp.Size = int64(len(body))
					resp.OutputID = storedOutputID
					manifest.Record(actionID)
					break
				}

				// Cloud fetch (this is the slow path — benefits from concurrency)
				getStart := time.Now()
				body, err = backend.Get(actionID)
				getDur := time.Since(getStart).Nanoseconds()
				atomic.AddInt64(&getNs, getDur)
				if err != nil {
					resp.Miss = true
					break
				}
				if body == nil {
					resp.Miss = true
					break
				}
				atomic.AddInt64(&getBytes, int64(len(body)))
				path, err := local.Put(actionID, req.ActionID, body)
				if err != nil {
					resp.Err = fmt.Sprintf("local put: %v", err)
					break
				}
				atomic.AddInt64(&totalHits, 1)
				resp.DiskPath = path
				resp.Size = int64(len(body))
				resp.OutputID = req.ActionID
				manifest.Record(actionID)

			case CmdPut:
				atomic.AddInt64(&totalPuts, 1)
				actionID := fmt.Sprintf("%x", req.ActionID)
				path, err := local.Put(actionID, req.OutputID, req.Body)
				if err != nil {
					resp.Err = fmt.Sprintf("local put: %v", err)
					break
				}
				resp.DiskPath = path
				resp.Size = req.BodySize
				if len(req.Body) > 0 {
					uploader.Queue(actionID, req.Body)
				}
			}

			// Serialize response writes
			writeMu.Lock()
			if err := enc.Encode(resp); err != nil {
				log.Printf("send response: %v", err)
			}
			stdout.Flush()
			writeMu.Unlock()
		}(req)
	}

	wg.Wait()
}
