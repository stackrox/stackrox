package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	DiskPath string `json:",omitempty"` // set when Go wrote the output file itself (no body line follows)
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
	// Control plane (manifests, prefetch bundles, indexes) always uses the
	// raw backend — bundling must never swallow these keys.
	rawBackend := backend
	// Bundle backend: OPTIONAL opt-in (GOCACHE_SERVE_BUNDLES=1). The A/B
	// data showed order-based bundles add index cost without measurable
	// prefetch benefit at 64-way parallel individual fetch; the size-gated
	// per-job prefetch bundles (pfb.go) replace this concept.
	if os.Getenv("GOCACHE_SERVE_BUNDLES") == "1" {
		maxEntries := envInt("GOCACHE_SERVE_BUNDLE_MAX_ENTRIES", 50)
		maxBytes := int64(envInt("GOCACHE_SERVE_BUNDLE_MAX_BYTES", 5<<20))
		bundled, err := NewBundledBackend(backend, maxEntries, maxBytes)
		if err != nil {
			log.Fatalf("bundle backend init: %v", err)
		}
		backend = bundled
	}
	defer backend.Close() // flushes pending bundle + uploads index

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

	// Prefetch bundles: size-gated, per-job, written by manifest-writing runs.
	pfbEnabled, pfbMaxEntry, pfbBundleBytes := pfbEnvs()

	// Negative cache: skip cloud round trips for entries already confirmed
	// absent (Go's speculative gets re-check phantoms every session).
	negatives := newNegativeCache()

	// Manifest load (synchronous: one small blob GET, and the get handler
	// needs the metas to serve correct OutputIDs on cloud hits — Go's
	// client rejects any get response whose OutputID isn't exactly 32
	// bytes, silently treating the hit as a miss and recompiling).
	var manifestEntries []manifestEntry
	metaBy := make(map[string][]byte) // actionID -> raw decoded OutputID
	if jobName != "" {
		var err error
		manifestEntries, _, err = DownloadManifest(rawBackend, jobName)
		if err != nil {
			log.Printf("manifest download failed: %v", err)
		}
		for _, e := range manifestEntries {
			if raw, err := hex.DecodeString(e.meta); err == nil && len(raw) > 0 {
				metaBy[e.id] = raw
			}
		}
	}

	// Prefetch completion gate: the first get waits for prefetch to finish.
	// Added after prefetching thousands of files concurrently with go's own
	// package loading (which mmaps the module index) reproducibly
	// SIGBUS-crashed the go process on CoW/overlay filesystems. Since then
	// the write pattern changed substantially (pfb bundles unpack in
	// ~300ms, range-GETs, negative cache) — whether the race still exists
	// is unverified. GOCACHE_SERVE_GATE=0 disables for A/B; the default
	// keeps the safety until an ungated CI run proves it unnecessary.
	gatePrefetch := os.Getenv("GOCACHE_SERVE_GATE") != "0"
	prefetchDone := make(chan struct{})
	if jobName == "" || os.Getenv("GOCACHE_SERVE_PREFETCH") == "0" || !gatePrefetch {
		close(prefetchDone)
	}

	// Prefetch from manifest (see bundle.go for BundledBackend)
	if os.Getenv("GOCACHE_SERVE_PREFETCH") != "0" && jobName != "" {
		go func() {
			defer close(prefetchDone)
			start := time.Now()
			entries := manifestEntries
			if len(entries) == 0 {
				log.Printf("no manifest for '%s', skipping prefetch (first run)", jobName)
				return
			}
			log.Printf("prefetching %d entries from manifest '%s'...", len(entries), jobName)
			// Size-gated prefetch bundles first: the small (<50KB, RTT-bound)
			// half of the working set lands as a handful of parallel bundle
			// GETs; the per-entry loop below then only pulls large entries.
			if pfbEnabled {
				metas := make(map[string]string, len(entries))
				for _, e := range entries {
					metas[e.id] = e.meta
				}
				PrefetchPFBs(rawBackend, jobName, local, metas)
			}
			// 64-way concurrency: the scalability bench measured ~6MB/s per
			// connection cross-region with flat latency to 100 connections,
			// so bulk prefetch throughput is connection-count-bound.
			sem := make(chan struct{}, 64)
			var wg sync.WaitGroup
			var downloaded int64
			for _, ent := range entries {
				key := ent.id
				if strings.HasPrefix(key, "manifest-") || strings.HasPrefix(key, "bundle-") || key == "bundle-index" {
					continue
				}
				wg.Add(1)
				go func(ent manifestEntry) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					// Stat-based existence check: reading the body here
					// cost ~21s per warm session for 20k entries (~7GB of
					// pointless disk reads).
					if local.Has(ent.id) {
						return
					}
					body, err := backend.Get(ent.id)
					if err != nil || body == nil {
						return
					}
					// Store the raw decoded OutputID sidecar: Go's client
					// rejects get responses whose OutputID isn't exactly
					// 32 bytes (hex-string-as-bytes = silent miss).
					local.Put(ent.id, metaBy[ent.id], body)
					atomic.AddInt64(&downloaded, 1)
				}(ent)
			}
			wg.Wait()
			log.Printf("prefetch complete: %d/%d entries in %v",
				atomic.LoadInt64(&downloaded), len(entries), time.Since(start).Round(time.Millisecond))
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
	// waitForConcurrent: lease coordination between concurrent runners.
	// A/B experiment (3 reps x 3 workloads, 2026-11-08) showed the lease
	// wait/recheck delivered zero concurrent hits while adding acquire-PUT
	// latency to every miss — the pack effect comes from plain
	// content-addressed GETs. Default OFF; set -1 to re-enable for
	// experimentation.
	waitForConcurrent := envInt("GOCACHE_SERVE_WAIT_FOR_CONCURRENT", 0)
	var concurrentHits int64
	var leaseMgr *LeaseManager
	// Lease TTL must cover the leader's compile + async upload for the
	// largest package; a too-short TTL risks stealing live work.
	leaseTTL := time.Duration(envInt("GOCACHE_SERVE_LEASE_TTL_MS", 120000)) * time.Millisecond
	if waitForConcurrent != 0 {
		leaseMgr = NewLeaseManager(backend, leaseTTL)
		uploader.onUploaded = leaseMgr.Release // release after entry lands in cloud
		defer leaseMgr.Stop()                  // stop the renewal sweeper at exit
	}
	// Request tracing for protocol inspection: logs every get/put with
	// truncated IDs, sizes, and DiskPath usage. stderr only (stdout is
	// the GOCACHEPROG JSON channel).
	trace := os.Getenv("GOCACHE_SERVE_TRACE") == "1"

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

		// Read PUT body in the main goroutine (stdin is sequential).
		// DiskPath puts carry no body line — Go wrote the file itself.
		if req.Command == CmdPut && req.BodySize > 0 && req.DiskPath == "" {
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
			mergedEntries, err := manifest.Upload(rawBackend)
			if err != nil {
				log.Printf("manifest upload failed: %v", err)
			}
			negatives.Persist() // phantom entries: skip cloud checks next session
			// Pack the run's small entries into prefetch bundles for the
			// next run (same gating as the manifest itself).
			if pfbEnabled && err == nil && len(mergedEntries) > 0 {
				if err := UploadPFBs(rawBackend, jobName, manifest.runID, mergedEntries, local, pfbMaxEntry, pfbBundleBytes); err != nil {
					log.Printf("pfb upload failed: %v", err)
				}
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
				// Block until prefetch completes (no-op after the first
				// process's prefetch): our bulk writes must not race go's
				// mmap'd package loading (SIGBUS on CoW filesystems).
				<-prefetchDone
				actionID := fmt.Sprintf("%x", req.ActionID)
				if trace {
					// hash the served file so put-after-get comparisons show
					// whether Go recompiled different content or re-put the same
					h := ""
					if b, err := os.ReadFile(local.path(actionID)); err == nil {
						sum := sha256.Sum256(b)
						h = fmt.Sprintf("%x", sum[:8])
					}
					log.Printf("TRACE get id=%s out=%x bodyHash=%s", actionID, req.OutputID, h)
				}

				// Local cache first (no network). Meta+stat only: the body
				// is handed to Go via DiskPath — reading it here would
				// double-copy every entry.
				storedOutputID, entrySize, hit := local.GetMeta(actionID)
				if hit {
					atomic.AddInt64(&totalHits, 1)
					resp.DiskPath = local.path(actionID)
					resp.Size = entrySize
					resp.OutputID = storedOutputID
					manifest.Record(actionID, fmt.Sprintf("%x", storedOutputID))
					break
				}

				// Cloud fetch (this is the slow path — benefits from concurrency)
				if negatives.Has(actionID) {
					resp.Miss = true
					break
				}
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
					// Confirmed absent — remember so later sessions in this
					// job skip the cloud round trip.
					negatives.Add(actionID)
					resp.Miss = true
					break
				}
				atomic.AddInt64(&getBytes, int64(len(getBody)))
				cloudMeta := metaBy[actionID]
				if cloudMeta == nil {
					cloudMeta = req.ActionID // length-correct fallback
				}
				path, putErr := local.Put(actionID, cloudMeta, getBody)
				if putErr != nil {
					resp.Err = fmt.Sprintf("local put: %v", err)
					break
				}
				atomic.AddInt64(&totalHits, 1)
				resp.DiskPath = path
				resp.Size = int64(len(getBody))
				// Serve the true OutputID from the manifest meta (Go rejects
				// non-32-byte OutputIDs); fall back to the actionID which at
				// least has the right length.
				resp.OutputID = metaBy[actionID]
				if resp.OutputID == nil {
					resp.OutputID = req.ActionID
				}
				manifest.Record(actionID, fmt.Sprintf("%x", resp.OutputID))

			case CmdPut:
				atomic.AddInt64(&totalPuts, 1)
				actionID := fmt.Sprintf("%x", req.ActionID)
				if trace {
					sum := sha256.Sum256(req.Body)
					log.Printf("TRACE put id=%s out=%x bodyHash=%x bodyLen=%d size=%d disk=%q",
						actionID, req.OutputID, sum[:8], len(req.Body), req.BodySize, req.DiskPath)
				}
				// Record produced entries too: a cold run has no hits, but
				// its puts ARE the working set (they'll be hits next run).
				manifest.Record(actionID, fmt.Sprintf("%x", req.OutputID))
				path, err := local.Put(actionID, req.OutputID, req.Body)
				if err != nil {
					resp.Err = fmt.Sprintf("local put: %v", err)
					break
				}
				resp.DiskPath = path
				resp.Size = req.BodySize
				// Always queue the upload, including zero-length bodies:
				// Go issues empty-body puts (min bodyLen=0 in the observed
				// distribution), and an empty blob is a valid hittable
				// entry — skipping it froze ~35% of gets as permanent
				// misses (manifest recorded, store never served).
				uploader.Queue(actionID, req.Body)
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
