// Package gocache implements a GOCACHEPROG pass-through proxy backed by
// the standard GOCACHE directory layout, plus a miss log.
//
// Why a proxy at all, locally? Three inputs to cimemo's reasoning (see
// SENSORS.md, sensor #9):
//
//  1. Exact work accounting — the miss log reports which actions a
//     build missed/put, i.e. what a change actually invalidated
//     (per-package ground truth, no co-change inference).
//  2. Build-config capture — ActionIDs fold in CGO_ENABLED, GOOS,
//     GOARCH, flags and env that file-content keys miss.
//  3. Future remote tier — the same loop backed by a blob store
//     (ported from gocache-serve) makes compile caches
//     location-independent; local mode is the same codepath.
//
// Cohabitation rule: the proxy reads and writes the EXACT standard
// layout ($GOCACHE/<xx>/<actionID>-a action records, <outputID>-d data
// files), so cimemo-wrapped builds and bare `go` builds share one
// cache. A proxy that stored elsewhere would split the cache and make
// both sides colder.
//
// Standard record format (verified empirically against go 1.27's own
// cache): "v1 <actionIDhex> <outputIDhex> <size padded to 20> <timeNanos>\n"
package gocache

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
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Request is the JSON go sends us (mirrors cmd/go/internal/cacheprog).
type Request struct {
	ID       int64  `json:",omitempty"`
	Command  string `json:",omitempty"`
	ActionID []byte `json:",omitempty"`
	OutputID []byte `json:",omitempty"`
	BodySize int64  `json:",omitempty"`
	DiskPath string `json:",omitempty"` // set when go wrote the output file itself (no body follows)
}

// Response is the JSON we send back.
type Response struct {
	ID            int64     `json:",omitempty"`
	Err           string    `json:",omitempty"`
	KnownCommands []string  `json:",omitempty"`
	Miss          bool      `json:",omitempty"`
	OutputID      []byte    `json:",omitempty"`
	Size          int64     `json:",omitempty"`
	Time          time.Time `json:",omitempty"`
	DiskPath      string    `json:",omitempty"` // go reads the body from this file itself
}

// Stats is the miss log written on close. Path comes from
// CIMEMO_GOCACHE_STATS. This is sensor #9's output: what the build
// actually did at action granularity.
type Stats struct {
	Gets          int64    `json:"gets"`
	Hits          int64    `json:"hits"`
	Misses        int64    `json:"misses"`
	Puts          int64    `json:"puts"`
	PutBytes      int64    `json:"put_bytes"`
	PutDiskPath   int64    `json:"put_diskpath"`
	MissedActions []string `json:"missed_actions"`          // actionID hex, capped
	RequestedAll  []string `json:"requested_all,omitempty"` // every requested get, capped (drift diagnosis)
	DurationMs    int64    `json:"duration_ms"`

	RemoteHits   int64 `json:"remote_hits,omitempty"`
	RemotePuts   int64 `json:"remote_puts,omitempty"`
	RemoteFailed int64 `json:"remote_failed,omitempty"`
}

const maxLoggedMisses = 512

// Serve runs the protocol loop on stdin/stdout. Intended to be re-exec'ed
// as `cimemo gocache` with GOCACHEPROG pointing here.
func Serve() error {
	cacheDir := os.Getenv("GOCACHE")
	if cacheDir == "" {
		home, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("gocache: no GOCACHE: %w", err)
		}
		cacheDir = filepath.Join(home, "go-build")
	}
	return serveLoop(os.Stdin, os.Stdout, cacheDir, os.Getenv("CIMEMO_GOCACHE_STATS"), NewRemoteFromEnv())
}

// serveLoop is the testable protocol core.
func serveLoop(in io.Reader, out io.Writer, cacheDir, statsPath string, remote *Remote) error {
	stdout := bufio.NewWriter(out)
	enc := json.NewEncoder(stdout)
	var writeMu sync.Mutex // serializes response writes across goroutines
	var inFlight sync.WaitGroup

	// Handshake: announce capabilities before any request.
	if err := enc.Encode(Response{ID: 0, KnownCommands: []string{"get", "put", "close"}}); err != nil {
		return err
	}
	if err := stdout.Flush(); err != nil {
		return err
	}

	dec := json.NewDecoder(in)

	var stats Stats
	start := time.Now()

	// Prefetch: bulk-download the expected working set (module-identity
	// manifest) so gets mostly hit locally instead of paying per-request
	// cloud latency at go's request rate. Gets wait for it before their
	// own cloud fallback (they would only duplicate in-flight work).
	prefetchDone := make(chan struct{})
	var sessionActions []string
	var sessMu sync.Mutex
	neg := loadNegatives(cacheDir)
	tt := NewTimeoutTracker()
	if remote != nil {
		go prefetch(remote, cacheDir, neg, tt, prefetchDone)
	} else {
		close(prefetchDone)
	}
	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("cimemo-gocache: decode: %v", err)
			break
		}

		// PUT bodies arrive as a base64 JSON string after the request
		// line (only when go did not write the file itself).
		var body []byte
		if req.Command == "put" && req.BodySize > 0 && req.DiskPath == "" {
			var bodyStr string
			if err := dec.Decode(&bodyStr); err != nil {
				log.Printf("cimemo-gocache: body decode: %v", err)
				continue
			}
			// go sends base64 without padding markers via JSON string;
			// encoding is standard.
			b, err := base64Decode(bodyStr)
			if err != nil {
				log.Printf("cimemo-gocache: base64: %v", err)
				continue
			}
			body = b
		}

		switch req.Command {
		case "get":
			stats.Gets++
			if os.Getenv("CIMEMO_GOCACHE_RECORD_GETS") == "1" && len(stats.RequestedAll) < 20000 {
				stats.RequestedAll = append(stats.RequestedAll, hex.EncodeToString(req.ActionID))
			}
			resp := handleGet(cacheDir, &req)
			if resp.Miss && remote != nil {
				// Local miss: dispatch the cloud fetch to a goroutine and
				// keep draining go's request stream — the protocol allows
				// out-of-order responses. Sequential fetch measured 155ms
				// per action (2 RTTs to the blob store); that made the
				// remote tier slower than recompiling.
				inFlight.Add(1)
				go func(req Request) {
					defer inFlight.Done()
					<-prefetchDone // no duplicate work while bulk prefetch runs
					actionID := hex.EncodeToString(req.ActionID)
					var r Response
					if remoteGetAction(remote, cacheDir, actionID, neg, tt) {
						r = handleGet(cacheDir, &req)
						atomic.AddInt64(&stats.RemoteHits, 1)
						sessMu.Lock()
						sessionActions = append(sessionActions, actionID)
						sessMu.Unlock()
					} else {
						r = Response{ID: req.ID, Miss: true}
						atomic.AddInt64(&stats.Misses, 1)
					}
					writeMu.Lock()
					enc.Encode(r)
					stdout.Flush()
					writeMu.Unlock()
				}(req)
				continue
			}
			if resp.Miss {
				stats.Misses++
				if len(stats.MissedActions) < maxLoggedMisses {
					stats.MissedActions = append(stats.MissedActions, hex.EncodeToString(req.ActionID))
				}
			} else {
				stats.Hits++
			}
			writeMu.Lock()
			enc.Encode(resp)
			stdout.Flush()
			writeMu.Unlock()

		case "put":
			stats.Puts++
			if req.DiskPath != "" {
				stats.PutDiskPath++
			}
			storedPath, storedSize := handlePut(cacheDir, &req, body)
			if storedPath != "" {
				// Record only actions that produced storable output. go
				// also puts empty-body markers (buildid/vet) with no data
				// — uncacheable, and manifest entries for them would
				// waste a prefetch request each (measured: 330 of 1,233).
				sessMu.Lock()
				sessionActions = append(sessionActions, hex.EncodeToString(req.ActionID))
				sessMu.Unlock()
			}
			if remote != nil && storedPath != "" {
				// Queue the cloud copy — streams from the local data
				// file at flush time (constant memory).
				aID := hex.EncodeToString(req.ActionID)
				oID := actionOutputID(cacheDir, aID)
				if oID != "" {
					queueCloudPut(remote, aID, oID, storedPath, storedSize)
					stats.RemotePuts++
				}
			}
			stats.PutBytes += storedSize
			// go 1.27 requires DiskPath in put responses: where the
			// proxy stored the output (go reads it back from there for
			// -o copies and links). BodySize may be absent on
			// DiskPath puts — report what we actually stored.
			writeMu.Lock()
			enc.Encode(Response{ID: req.ID, DiskPath: storedPath, Size: storedSize})
			stdout.Flush()
			writeMu.Unlock()

		case "close":
			inFlight.Wait() // every dispatched get must answer first
			neg.Persist()
			if remote != nil {
				remote.flushUploadsConcurrent()
				sessMu.Lock()
				actions := sessionActions
				sessMu.Unlock()
				saveSessionManifest(remote, actions)
				up, failed, lastErr := remote.uploadCountsWithDetail()
				stats.RemotePuts = up
				stats.RemoteFailed = failed
				if failed > 0 {
					logf("uploads: %d ok, %d FAILED (last: %s)", up, failed, lastErr)
				}
			}
			stats.DurationMs = time.Since(start).Milliseconds()
			writeStats(statsPath, &stats)
			writeMu.Lock()
			enc.Encode(Response{ID: req.ID})
			stdout.Flush()
			writeMu.Unlock()
			return nil

		default:
			log.Printf("cimemo-gocache: unknown command %q", req.Command)
		}
	}

	// stdin closed without a close command — still write stats.
	stats.DurationMs = time.Since(start).Milliseconds()
	writeStats(statsPath, &stats)
	return nil
}

// handleGet serves from the standard layout, handing go the data file
// path so it reads the body itself (no double copy).
func handleGet(cacheDir string, req *Request) Response {
	actionID := hex.EncodeToString(req.ActionID)
	outputID, size, ok := readActionRecord(cacheDir, actionID)
	if !ok {
		return Response{ID: req.ID, Miss: true}
	}
	dataPath := dataPath(cacheDir, outputID)
	if info, err := os.Stat(dataPath); err != nil || info.Size() != size {
		// Action record exists but data is gone (partial GC) — miss;
		// go will recompile and re-put (self-healing).
		return Response{ID: req.ID, Miss: true}
	}
	out, _ := hex.DecodeString(outputID)
	// Touch the action record so go's LRU sees recent use, same as
	// its own cache would.
	os.Chtimes(actionPath(cacheDir, actionID), time.Now(), time.Now())
	return Response{
		ID:       req.ID,
		OutputID: out,
		Size:     size,
		DiskPath: dataPath,
	}
}

// handlePut writes the output data and action record in the standard
// format so bare `go` builds see them too. Returns the stored data
// file path and size for the put response.
func handlePut(cacheDir string, req *Request, body []byte) (string, int64) {
	actionID := hex.EncodeToString(req.ActionID)

	var outputIDHex string
	if len(req.OutputID) > 0 {
		outputIDHex = hex.EncodeToString(req.OutputID)
		// Integrity check (the CA-work lesson): OutputID must be the
		// content hash. If it is not, trust the content — name the
		// data file by the real hash and record that.
		if len(body) > 0 {
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) != outputIDHex {
				log.Printf("cimemo-gocache: put %s: OutputID != sha256(body); using content hash",
					actionID[:12])
				outputIDHex = hex.EncodeToString(sum[:])
			}
		}
	} else if len(body) > 0 {
		// No OutputID given: derive it (content addressing).
		sum := sha256.Sum256(body)
		outputIDHex = hex.EncodeToString(sum[:])
	}

	size := int64(len(body))
	if len(body) > 0 {
		writeDataFile(cacheDir, outputIDHex, body)
	} else if req.DiskPath != "" {
		// go wrote the file itself and may delete it right after our
		// response — ingest NOW, streaming (constant memory for large
		// link outputs).
		outputIDHex, size = ingestDiskPathStreaming(cacheDir, outputIDHex, req.DiskPath)
		if outputIDHex == "" {
			return "", 0
		}
	} else {
		return "", 0 // nothing to store
	}

	writeActionRecord(cacheDir, actionID, outputIDHex, size)
	return dataPath(cacheDir, outputIDHex), size
}

// --- standard layout helpers ---

func actionPath(cacheDir, actionIDHex string) string {
	return filepath.Join(cacheDir, actionIDHex[:2], actionIDHex+"-a")
}

func dataPath(cacheDir, outputIDHex string) string {
	return filepath.Join(cacheDir, outputIDHex[:2], outputIDHex+"-d")
}

// readActionRecord parses "v1 <actionID> <outputID> <size:20> <timeNs>".
func readActionRecord(cacheDir, actionIDHex string) (outputIDHex string, size int64, ok bool) {
	data, err := os.ReadFile(actionPath(cacheDir, actionIDHex))
	if err != nil {
		return "", 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 4 || fields[0] != "v1" {
		return "", 0, false
	}
	var sz int64
	fmt.Sscanf(fields[3], "%d", &sz)
	return fields[2], sz, true
}

// writeActionRecord writes the standard-format record atomically.
func writeActionRecord(cacheDir, actionIDHex, outputIDHex string, size int64) {
	p := actionPath(cacheDir, actionIDHex)
	os.MkdirAll(filepath.Dir(p), 0o755)
	rec := fmt.Sprintf("v1 %s %s %20d %d\n", actionIDHex, outputIDHex, size, time.Now().UnixNano())
	atomicWrite(p, []byte(rec))
}

func writeDataFile(cacheDir, outputIDHex string, body []byte) {
	p := dataPath(cacheDir, outputIDHex)
	if _, err := os.Stat(p); err == nil {
		return // dedup: content-addressed, same bytes
	}
	os.MkdirAll(filepath.Dir(p), 0o755)
	atomicWrite(p, body)
}

// atomicWrite writes temp+rename so go never sees partial entries.
func atomicWrite(path string, data []byte) {
	tmp := path + ".tmp-cimemo"
	if err := os.WriteFile(tmp, data, 0o666); err != nil {
		log.Printf("cimemo-gocache: write %s: %v", path, err)
		return
	}
	os.Rename(tmp, path)
}

func writeStats(path string, stats *Stats) {
	if path == "" {
		return
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
}

// ReadStats loads a miss log written by a Serve run.
func ReadStats(path string) (*Stats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Stats
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
