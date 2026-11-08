package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
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

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
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
	// Read-only for PR runs (don't pollute the master working set);
	// testable via GOCACHE_SERVE_MANIFEST_WRITE=0.
	if os.Getenv("GOCACHE_SERVE_MANIFEST_WRITE") == "0" {
		manifest.readOnly = true
	}

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
	//
	// Timeout tracker: adaptive by default (P95-based), configurable via env
	timeoutTracker := NewTimeoutTracker()

	// Concurrency limit: adaptive by default (self-tunes via AIMD),
	// or fixed via GOCACHE_SERVE_CONCURRENCY=N (for testing).
	// waitForConcurrent: max total ms a follower waits for the leader's
	// upload. Default -1 = wait until the lease goes stale (TTL-bounded),
	// which is the correct shape: after giving up, the follower pays the
	// full compile time anyway, so waiting while the leader is alive is
	// nearly always cheaper than duplicating its work. A dead leader is
	// detected by lease staleness, not by this deadline.
	// 0 disables lease coordination entirely.
	waitForConcurrent := envInt("GOCACHE_SERVE_WAIT_FOR_CONCURRENT", -1)
	var concurrentHits int64
	var leaseMgr *LeaseManager
	// Lease TTL must cover the leader's compile + async upload for the
	// largest package; a too-short TTL risks stealing live work.
	leaseTTL := time.Duration(envInt("GOCACHE_SERVE_LEASE_TTL_MS", 120000)) * time.Millisecond
	if waitForConcurrent != 0 {
		leaseMgr = NewLeaseManager(backend, leaseTTL)
		uploader.onUploaded = leaseMgr.Release // release after entry lands in cloud
	}
	var writeMu sync.Mutex
	var wg sync.WaitGroup

	// Periodic health ticker: correlates upload backlog with get activity.
	// If concurrent-hit rates are low on same-region runs, this log shows
	// whether uploads were saturating the pipe at that moment (queue depth
	// high while gets slow) or whether something else was the cause.
	if os.Getenv("GOCACHE_SERVE_HEALTH_TICKER") != "0" {
		go func() {
			var lastUploaded, lastGetBytes int64
			for range time.Tick(10 * time.Second) {
				up, _, dropped := uploader.Stats()
				uploadedDelta := up - lastUploaded
				lastUploaded = up
				gb := atomic.LoadInt64(&getBytes)
				getDelta := (gb - lastGetBytes) / 1048576
				lastGetBytes = gb
				log.Printf("gocache-serve: health: uploadQ=%d uploads=%d (+%d) dropped=%d gets=%d hits=%d concurrentHits=%d getMB+=%d",
					uploader.QueueDepth(), up, uploadedDelta, dropped,
					atomic.LoadInt64(&totalGets), atomic.LoadInt64(&totalHits),
					atomic.LoadInt64(&concurrentHits), getDelta)
			}
		}()
	}

	var cloudSem *AdaptiveSem
	switch n := os.Getenv("GOCACHE_SERVE_CONCURRENCY"); {
	case n == "" || n == "auto":
		// Adaptive: start at 2, max 64, self-tunes based on latency
		cloudSem = NewAdaptiveSem(2, 64)
		log.Printf("gocache-serve: adaptive cloud GET concurrency (2-64)")
	case n == "0":
		// Unlimited
		cloudSem = nil
	default:
		// Fixed limit
		var limit int
		if _, err := fmt.Sscanf(n, "%d", &limit); err == nil && limit > 0 {
			cloudSem = NewAdaptiveSem(limit, limit)
			log.Printf("gocache-serve: fixed cloud GET concurrency: %d", limit)
		}
	}

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
			var concurrentStats string
			if n := atomic.LoadInt64(&concurrentHits); n > 0 {
				concurrentStats = fmt.Sprintf(", %d concurrent hits (leader-shared)", n)
			}
			log.Printf("closing: %d gets (%d hits, %.1f%%), %d puts, elapsed %v%s%s",
				atomic.LoadInt64(&totalGets), atomic.LoadInt64(&totalHits),
				float64(atomic.LoadInt64(&totalHits))/float64(atomic.LoadInt64(&totalGets)+1)*100,
				atomic.LoadInt64(&totalPuts), elapsed.Round(time.Millisecond), netStats, concurrentStats)
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
				if cloudSem != nil {
					cloudSem.Acquire()
				}
				getStart := time.Now()

				// Timeout: if the fetch takes too long, return Miss so Go
				// recompiles rather than blocking its critical path. The
				// recompiled result gets `put` back to the cache (self-healing).
				timeout := timeoutTracker.Timeout()
				type getResult struct {
					body []byte
					err  error
				}
				ch := make(chan getResult, 1)
				go func() {
					body, err := backend.Get(actionID)
					ch <- getResult{body, err}
				}()

				var getBody []byte
				var getErr error
				timedOut := false
				select {
				case r := <-ch:
					getBody, getErr = r.body, r.err
				case <-time.After(timeout):
					timedOut = true
					getBody, getErr = nil, nil
					log.Printf("gocache-serve: get %s timed out after %v (returning miss)", shortID(actionID), timeout)
				}

				getDur := time.Since(getStart)
				if cloudSem != nil {
					cloudSem.Release(getDur)
				}
				timeoutTracker.Record(getDur, timedOut)
				atomic.AddInt64(&getNs, getDur.Nanoseconds())
				if timedOut {
					resp.Miss = true
					break
				}
				if getErr != nil {
					resp.Miss = true
					break
				}
				if getBody == nil {
					if leaseMgr != nil {
						// Advisory lease: divide compilation among concurrent
						// runners. The leader compiles immediately (no wait);
						// followers wait while the leader's lease is fresh.
						if !leaseMgr.Acquire(actionID) {
							// Follower budget: TTL-bounded by default (wait
							// until the lease goes stale), or the explicit
							// GOCACHE_SERVE_WAIT_FOR_CONCURRENT override.
							budget := leaseTTL
							if waitForConcurrent > 0 {
								budget = time.Duration(waitForConcurrent) * time.Millisecond
							}
							waitDeadline := time.Now().Add(budget)
							for time.Now().Before(waitDeadline) {
								time.Sleep(1500 * time.Millisecond)
								if cloudSem != nil {
									cloudSem.Acquire()
								}
								retryStart := time.Now()
								body, retryErr := backend.Get(actionID)
								retryDur := time.Since(retryStart)
								if cloudSem != nil {
									cloudSem.Release(retryDur)
								}
								timeoutTracker.Record(retryDur, false)
								if retryErr == nil && body != nil {
									// Leader delivered while we waited
									atomic.AddInt64(&concurrentHits, 1)
									log.Printf("gocache-serve: concurrent hit for %s (leader delivered)",
										shortID(actionID))
									getBody = body
									break
								}
								// Entry still absent: if the leader's lease went
								// stale (crash/hang), take over compilation.
								if !leaseMgr.HolderIsFresh(actionID) && leaseMgr.Acquire(actionID) {
									log.Printf("gocache-serve: lease for %s expired, taking over",
										shortID(actionID))
									break // fall through to Miss — we compile
								}
							}
						}
					}
				}
				if getBody == nil {
					resp.Miss = true
					break
				}
				atomic.AddInt64(&getBytes, int64(len(getBody)))
				path, putErr := local.Put(actionID, req.ActionID, getBody)
				if putErr != nil {
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
					if leaseMgr != nil && leaseMgr.Holds(actionID) {
						// Lease-held: followers are waiting for this entry —
						// priority lane so it reaches the cloud fast.
						uploader.QueuePri(actionID, req.Body)
					} else {
						uploader.Queue(actionID, req.Body)
					}
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
