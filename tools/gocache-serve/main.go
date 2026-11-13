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
	"os/exec"
	"path/filepath"
	"sort"
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

// tryMountResourceDisk looks for an unmounted Azure resource disk (the
// physically-local SSD) and mounts it for our use. Pure use-if-found:
// every failure path falls back to the normal ladder. Idempotent across
// the sessions of a job (mountpoint check first). Only ever formats the
// /dev/disk/azure/resource symlink target — never an arbitrary device.
func tryMountResourceDisk() string {
	res := "/dev/disk/azure/resource"
	target := "/mnt/azresource"
	if _, err := os.Stat(res); err != nil {
		return "" // no azure device symlinks (non-Azure or image change)
	}
	// Already mounted by us (earlier session in this job)? Reuse.
	if st, err := os.Stat(target + "/.gocache-serve"); err == nil && st.IsDir() {
		return target
	}
	// Refuse if something else is mounted there or the resource disk is
	// already in use elsewhere.
	if out, err := os.ReadFile("/proc/mounts"); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, target+" ") {
				return "" // mounted but not ours — don't touch
			}
		}
	}
	// Mount (format only if no filesystem present).
	os.MkdirAll(target, 0o755)
	if sh("sudo", "mount", res, target) != nil {
		// No filesystem: format first (resource disk is scratch by
		// definition — safe to reformat).
		if sh("sudo", "mkfs.ext4", "-F", res) != nil {
			return ""
		}
		if sh("sudo", "mount", res, target) != nil {
			return ""
		}
	}
	os.MkdirAll(target+"/.gocache-serve", 0o755)
	log.Printf("gocache-serve: mounted Azure resource disk %s at %s", res, target)
	return target
}

func sh(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// pickLocalDir selects the local cache location by capability, with
// graceful fallback for runners that lack a given tier:
//  1. Azure resource disk — the physically-local SSD, mounted if found
//     raw (fsync-probed; use-if-found)
//  2. /mnt — a pre-mounted local temp disk, if present and probed local
//  3. /dev/shm — tmpfs (RAM), with the disk spill tier for overflow
//  4. /tmp — the network OS disk (always present; today's default)
//
// The fsync probe is the honest "is it actually local" test: write 64KB,
// fsync, measure. Local SSD: sub-millisecond. Network storage: tens of ms.
// spillDirFor mirrors the SetSpill condition: the disk fallback tier for
// tmpfs/local-disk primaries.
func spillDirFor(cacheDir string) string {
	if os.Getenv("GOCACHE_SERVE_SPILL") != "" {
		return os.Getenv("GOCACHE_SERVE_SPILL")
	}
	if strings.HasPrefix(cacheDir, "/dev/shm") || strings.HasPrefix(cacheDir, "/mnt/") {
		return "/tmp/gocache-serve-spill"
	}
	return ""
}

func pickLocalDir() string {
	type candidate struct {
		dir     string
		maxSync time.Duration
		spill   string // spill tier for capacity overflow ("" = none needed)
		why     string
	}
	var candidates []candidate
	if d := tryMountResourceDisk(); d != "" {
		candidates = append(candidates, candidate{d, 5 * time.Millisecond, "/tmp/gocache-serve-spill", "Azure resource disk (mounted)"})
	}
	candidates = append(candidates,
		candidate{"/mnt/gocache-serve", 5 * time.Millisecond, "/tmp/gocache-serve-spill", "local temp disk (fsync-probed)"},
		candidate{"/dev/shm/gocache-serve", 0, "/tmp/gocache-serve-spill", "tmpfs (RAM) + disk spill"},
		candidate{"/tmp/gocache-serve", 0, "", "network OS disk (default)"},
	)
	for _, c := range candidates {
		if err := os.MkdirAll(c.dir, 0o755); err != nil {
			continue
		}
		if c.maxSync > 0 {
			d, err := probeSyncLatency(c.dir)
			if err != nil || d > c.maxSync {
				log.Printf("gocache-serve: %s rejected (fsync %v, err %v) — falling back", c.dir, d, err)
				os.RemoveAll(c.dir)
				continue
			}
			log.Printf("gocache-serve: local cache on %s — %s, fsync %v", c.dir, c.why, d)
			return c.dir
		}
		log.Printf("gocache-serve: local cache on %s — %s", c.dir, c.why)
		return c.dir
	}
	return "/tmp/gocache-serve"
}

// probeSyncLatency measures median write+fsync latency in a directory.
func probeSyncLatency(dir string) (time.Duration, error) {
	var times []time.Duration
	for i := 0; i < 3; i++ {
		p := filepath.Join(dir, fmt.Sprintf(".probe-%d", i))
		start := time.Now()
		err := os.WriteFile(p, make([]byte, 64<<10), 0o644)
		if err == nil {
			f, ferr := os.Open(p)
			if ferr == nil {
				err = f.Sync()
				f.Close()
			}
		}
		os.Remove(p)
		if err != nil {
			return 0, err
		}
		times = append(times, time.Since(start))
	}
	sort.Slice(times, func(a, b int) bool { return times[a] < times[b] })
	return times[len(times)/2], nil
}

// manifestHash returns the sha256 of a manifest's sorted entry ids —
// order-independent so run-scoped write order doesn't change the key.
// The stable identity of "what source+flags produced this working set",
// used to key link-output caching.
func manifestHash(entries []manifestEntry) string {
	if len(entries) == 0 {
		return ""
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.id)
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, "\n")))
	return fmt.Sprintf("%x", sum[:16])
}

// writeStub writes a minimal ELF64 file containing a single PT_NOTE
// segment with a Go buildID note. cmd/internal/buildid reads only the
// note (zeroing section headers, never hashing content), so this file
// satisfies go's up-to-date check for the given buildID. Layout:
// ELF header (64B) + one program header (56B) + note at offset 120.
func writeStub(buildID, outfile string) error {
	if len(buildID) == 0 {
		return fmt.Errorf("empty buildID")
	}
	// Validate: a go buildID is 1-4 slash-separated URL-safe base64
	// segments. A malformed string would produce a stub that silently
	// fails go's check — catch it here.
	for _, seg := range strings.Split(buildID, "/") {
		if len(seg) == 0 {
			return fmt.Errorf("empty segment in buildID %q", buildID)
		}
		for _, c := range seg {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
				return fmt.Errorf("invalid character %q in buildID segment (URL-safe base64 expected)", c)
			}
		}
	}
	noteLen := 12 + 4 + len(buildID)
	buf := make([]byte, 120+noteLen)
	le := func(off int, n int, v uint64) {
		for i := 0; i < n; i++ {
			buf[off+i] = byte(v >> (8 * i))
		}
	}
	// ELF64 header, LSB.
	copy(buf[0:4], "\x7fELF")
	buf[4] = 2    // ELFCLASS64
	buf[5] = 1    // ELFDATA2LSB
	buf[6] = 1    // EV_CURRENT
	le(16, 2, 2)  // e_type = ET_EXEC
	le(18, 2, 62) // e_machine = EM_X86_64
	le(20, 4, 1)  // e_version
	le(32, 8, 64) // e_phoff = 64
	le(52, 2, 64) // e_ehsize
	le(54, 2, 56) // e_phentsize
	le(56, 2, 1)  // e_phnum
	le(58, 2, 64) // e_shentsize (shnum=0: parser zeroes anyway)
	// Program header: PT_NOTE at offset 64.
	le(64+0, 4, 4)                // p_type = PT_NOTE
	le(64+4, 4, 4)                // p_flags = PF_R
	le(64+8, 8, 120)              // p_offset
	le(64+16, 8, 120)             // p_vaddr
	le(64+24, 8, 120)             // p_paddr
	le(64+32, 8, uint64(noteLen)) // p_filesz
	le(64+40, 8, uint64(noteLen)) // p_memsz
	le(64+48, 8, 4)               // p_align
	// Note: namesz=4, descsz=len(buildID), type=4 (go build ID),
	// name "Go\0\0", value = buildID.
	le(120, 4, 4)
	le(124, 4, uint64(len(buildID)))
	le(128, 4, 4)
	copy(buf[132:136], "Go\x00\x00")
	copy(buf[136:], buildID)
	return os.WriteFile(outfile, buf, 0o755)
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("gocache-serve: ")

	// Workflow-level CLI subcommands (link-output caching). These run
	// outside the GOCACHEPROG protocol: -hash-manifest prints the sorted-
	// entry sha of a stored manifest (the key predicting a build's output
	// binary); -fetch/-store move blobs (e.g. output binaries) through the
	// same backend + auth as the protocol mode.
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-") {
		// -stub needs no backend; the others do.
		if os.Args[1] == "-stub" {
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: -stub BUILDID OUTFILE")
				os.Exit(2)
			}
			if err := writeStub(os.Args[2], os.Args[3]); err != nil {
				fmt.Fprintf(os.Stderr, "stub: %v\n", err)
				os.Exit(1)
			}
			return
		}
		backend, err := newBackend()
		if err != nil {
			fmt.Fprintf(os.Stderr, "backend: %v\n", err)
			os.Exit(1)
		}
		switch os.Args[1] {
		case "-hash-manifest":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: -hash-manifest JOB")
				os.Exit(2)
			}
			entries, _, err := DownloadManifest(backend, os.Args[2])
			if err != nil {
				fmt.Fprintf(os.Stderr, "download: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(manifestHash(entries))
		case "-doctor":
			// Compare local build environment against the CI-recorded
			// fingerprint; exit 1 on mismatch (scriptable for onboarding).
			if err := doctorCommand(); err != nil {
				fmt.Fprintf(os.Stderr, "doctor: %v\n", err)
				os.Exit(1)
			}
		case "-metrics-report":
			// Ad-hoc aggregation over metric blobs: a UTC date (20261112),
			// a run-id prefix, or "" for everything.
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: -metrics-report [DATE|RUN-ID-PREFIX]")
				os.Exit(2)
			}
			if err := reportMetrics(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "metrics report: %v\n", err)
				os.Exit(1)
			}
		case "-metrics-push":
			// Batch-ingest metric blobs into Log Analytics (scheduled).
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: -metrics-push [DATE|RUN-ID-PREFIX]")
				os.Exit(2)
			}
			if err := pushMetrics(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "metrics push: %v\n", err)
				os.Exit(1)
			}
		case "-fetch":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: -fetch KEY OUTFILE")
				os.Exit(2)
			}
			body, err := backend.Get(os.Args[2])
			if err != nil {
				fmt.Fprintf(os.Stderr, "get: %v\n", err)
				os.Exit(1)
			}
			if body == nil {
				os.Exit(1) // miss
			}
			if err := os.WriteFile(os.Args[3], body, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "write: %v\n", err)
				os.Exit(1)
			}
		case "-store":
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: -store KEY FILE")
				os.Exit(2)
			}
			body, err := os.ReadFile(os.Args[3])
			if err != nil {
				fmt.Fprintf(os.Stderr, "read: %v\n", err)
				os.Exit(1)
			}
			if err := backend.Put(os.Args[2], body); err != nil {
				fmt.Fprintf(os.Stderr, "put: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "stored %s (%d bytes)\n", os.Args[2], len(body))
		case "-stub":
			fmt.Fprintln(os.Stderr, "unreachable: -stub handled above")
		case "-verify":
			// Verify a binary's actual buildID matches the expected one —
			// the post-restore validation for the paranoid path, and the
			// way to confirm the link-bypass is working correctly.
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: -verify EXPECTED_BUILdid FILE")
				os.Exit(2)
			}
			actual, err := exec.Command("go", "tool", "buildid", os.Args[3]).Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "buildid: %v\n", err)
				os.Exit(1)
			}
			expected := os.Args[2]
			got := strings.TrimSpace(string(actual))
			if got != expected {
				fmt.Fprintf(os.Stderr, "MISMATCH: expected %s got %s\n", expected, got)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "verified: %s\n", got)
		case "-docker-push":
			if len(os.Args) < 5 {
				fmt.Fprintln(os.Stderr, "usage: -docker-push IMAGE TAG KEY_PREFIX")
				os.Exit(2)
			}
			if err := dockerPush(backend, os.Args[2], os.Args[3], os.Args[4]); err != nil {
				fmt.Fprintf(os.Stderr, "docker-push: %v\n", err)
				os.Exit(1)
			}
		case "-docker-pull":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: -docker-pull KEY_PREFIX")
				os.Exit(2)
			}
			if err := dockerPull(backend, os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "docker-pull: %v\n", err)
				os.Exit(1)
			}
		case "-docker-prefetch":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: -docker-prefetch KEY_PREFIX")
				os.Exit(2)
			}
			if err := dockerPrefetch(backend, os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "docker-prefetch: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand %s\n", os.Args[1])
			os.Exit(2)
		}
		return
	}

	cacheDir := os.Getenv("GOCACHE_SERVE_DIR")
	if cacheDir == "" {
		cacheDir = pickLocalDir()
	}

	backend, err := newBackend()
	if err != nil {
		log.Fatalf("backend init: %v", err)
	}
	// Control plane (manifests, prefetch bundles, indexes) always uses the
	// raw backend — bundling must never swallow these keys.
	rawBackend := backend
	// Environment fingerprint: CI records, devs get warned on mismatch
	// (a wrong go version or CGO setting silently zeroes cache hits).
	// Async — it shells out to `go env` and may do one blob GET; the
	// warning can land any time during the build.
	go maybeCheckFingerprint()
	// Bundle backend (order-based BundledBackend) was REMOVED: the A/B
	// data showed it added index cost without measurable prefetch
	// benefit at 64-way parallel individual fetch; the size-gated
	// per-job prefetch bundles (pfb.go) replaced the concept. See git
	// history for the implementation and its measurements.
	// Entry read-through: on a regional miss, try the hub before
	// declaring miss (and backfill regionally). Entry stores are
	// regionally partitioned by which runners populated them; without
	// this, a runner in one region recompiles everything another
	// region's runners already built (observed: operator-bundle +80s).
	// Azure only — the dir test backend is single-store by construction.
	if os.Getenv("GOCACHE_SERVE_BACKEND") != "dir" && os.Getenv("AZURE_CLIENT_ID") != "" &&
		os.Getenv("AZURE_STORAGE_ACCOUNT") == "" {
		// Not when explicitly pinned to one account (A/B experiments).
		backend = &fallbackBackend{primary: backend, secondary: hubBackend()}
	}
	defer backend.Close() // flushes pending bundle + uploads index

	local, err := NewLocalCache(cacheDir)
	if err != nil {
		log.Fatalf("local cache init: %v", err)
	}
	// tmpfs or local-temp-disk primary: enable the disk spill tier so a
	// working set overflowing the tier's capacity lands on slower storage
	// instead of failing the job. /dev/shm is ~3.4GB on runners; the Azure
	// temp disk ~14GB; the cli working set measures ~16GB.
	if strings.HasPrefix(cacheDir, "/dev/shm") || strings.HasPrefix(cacheDir, "/mnt/") || os.Getenv("GOCACHE_SERVE_SPILL") != "" {
		spill := os.Getenv("GOCACHE_SERVE_SPILL")
		if spill == "" {
			spill = "/tmp/gocache-serve-spill"
		}
		local.SetSpill(spill)
		log.Printf("gocache-serve: spill tier enabled: %s (primary %s)", spill, cacheDir)
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
	// Metadata lives on the spill tier when one is configured: the tmpfs
	// primary can be 100% full of cache entries (measured ENOSPC on
	// served-set writes while the cache itself worked fine via spill).
	negatives := newNegativeCache(cacheDir, spillDirFor(cacheDir))
	// In-flight download coordination between prefetch and get paths.
	inflights := newInflightMap()
	var conversions int64 // gets that waited on a prefetch (prefetch behind need)

	// Hot-core measurement: the intersection of the previous two sessions'
	// served sets = entries re-read across sessions — the population a
	// page-cache pin/rewarm should protect. Logged once per session.
	if dir := metaDir(cacheDir, spillDirFor(cacheDir)); dir != "" {
		loadSet := func(name string) map[string]bool {
			s := map[string]bool{}
			if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				for _, id := range strings.Fields(string(b)) {
					s[id] = true
				}
			}
			return s
		}
		p1, p2 := loadSet("served-prev.txt"), loadSet("served-prev2.txt")
		log.Printf("gocache-serve: served sets: prev=%d prev2=%d (dir %s)", len(p1), len(p2), dir)
		if len(p1) > 0 && len(p2) > 0 {
			core := 0
			for id := range p1 {
				if p2[id] {
					core++
				}
			}
			log.Printf("gocache-serve: hot core (served in 2 consecutive sessions): %d of %d entries", core, len(p1))
		}
	}

	// Manifest load (synchronous: one small blob GET, and the get handler
	// needs the metas to serve correct OutputIDs on cloud hits — Go's
	// client rejects any get response whose OutputID isn't exactly 32
	// bytes, silently treating the hit as a miss and recompiling).
	var manifestEntries []manifestEntry
	metaBy := make(map[string][]byte) // actionID -> raw decoded OutputID
	if jobName != "" {
		var err error
		// Manifests are HUB-authoritative (control plane): a single
		// convergent copy instead of per-region snapshots that diverged
		// (observed: west 2429 entries, east 850, central 0 — regional
		// runners each replaced their region's manifest with their own
		// run's snapshot). Cross-region manifest reads cost ~100ms once
		// per process; convergence is worth far more.
		manifestEntries, _, err = DownloadManifest(controlBackend(), jobName)
		if err != nil {
			log.Printf("manifest download failed: %v", err)
		}
		// Manifest content hash: the entry set is a deterministic function
		// of source+flags, so manifest-hash + platform + flags predicts the
		// output binary — the key for link-output caching (go's on-disk
		// buildID check then skips the relink when the binary is restored).
		if h := manifestHash(manifestEntries); h != "" {
			log.Printf("gocache-serve: manifest sha256=%s (%d entries)", h, len(manifestEntries))
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

	// Prefetch from manifest (pfb bundles unpack separately; see pfb.go)
	if os.Getenv("GOCACHE_SERVE_PREFETCH") != "0" && jobName != "" {
		go func() {
			// Only the gated mode owns the channel's close; ungated closed
			// it at startup so compilation overlaps prefetch immediately.
			if gatePrefetch {
				defer close(prefetchDone)
			}
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
			// Prefetch concurrency: the direct prefetch is slot-bound, not
			// bandwidth-bound (measured ~160 entries/s at 64 slots cross-
			// region — raising range-parallelism per entry didn't move it;
			// the slot count is the limiter). The bench measured flat
			// latency to 100+ connections, so 128 slots. Overridable.
			pfConc := 128
			if v := os.Getenv("GOCACHE_SERVE_PREFETCH_CONCURRENCY"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					pfConc = n
				}
			}
			sem := make(chan struct{}, pfConc)
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
					// In-flight coordination with the get path: if a get is
					// already downloading this entry, skip (it will local.Put).
					if inflights.start(ent.id, "prefetch") == nil {
						return
					}
					defer inflights.finish(ent.id)
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
			mergedEntries, err := manifest.Upload(controlBackend())
			if err != nil {
				log.Printf("manifest upload failed: %v", err)
			}
			negatives.Persist() // phantom entries: skip cloud checks next session
			// Persist the served set: the previous two sessions' sets give
			// the hot core (cross-session intersection) at startup, and are
			// the rewarm input for page-cache pinning. Rotation keeps two.
			// (cacheDir, not the env: CI uses the default dir.)
			{
				mdir := metaDir(cacheDir, spillDirFor(cacheDir))
				prev := filepath.Join(mdir, "served-prev.txt")
				prev2 := filepath.Join(mdir, "served-prev2.txt")
				os.Rename(prev, prev2) // best-effort
				ids := manifest.LocalIDs()
				if len(ids) > 0 {
					if err := os.WriteFile(prev, []byte(strings.Join(ids, "\n")), 0o644); err != nil {
						log.Printf("gocache-serve: served-set write failed: %v", err)
					}
				}
				log.Printf("gocache-serve: served-set close: %d ids -> %s", len(ids), prev)
			}
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
			if n := atomic.LoadInt64(&conversions); n > 0 {
				concurrentStats += fmt.Sprintf(", %d conversions (prefetch behind need)", n)
			}
			log.Printf("closing: %d gets (%d hits, %.1f%%), %d puts, elapsed %v%s%s",
				atomic.LoadInt64(&totalGets), atomic.LoadInt64(&totalHits),
				float64(atomic.LoadInt64(&totalHits))/float64(atomic.LoadInt64(&totalGets)+1)*100,
				atomic.LoadInt64(&totalPuts), elapsed.Round(time.Millisecond), netStats, concurrentStats)
			// Metrics: one record per process, appended locally (free) and the
			// job's file uploaded as one blob per job. Never breaks the build.
			if gets, puts := atomic.LoadInt64(&totalGets), atomic.LoadInt64(&totalPuts); gets+puts > 0 {
				region, account := "", ""
				if ab, ok := rawBackend.(*AzureBackend); ok {
					region, account = ab.region, ab.account
				}
				rec := MetricsRecord{
					Ts:             time.Now().Unix(),
					RunID:          manifest.runID,
					Job:            jobName,
					Region:         region,
					Account:        account,
					Gets:           gets,
					Hits:           atomic.LoadInt64(&totalHits),
					Puts:           puts,
					GetBytes:       getBytes,
					GetMs:          getNs / 1e6,
					ElapsedMs:      elapsed.Milliseconds(),
					ConcurrentHits: atomic.LoadInt64(&concurrentHits),
				}
				appendMetric(rec)
				uploadJobMetrics(manifest.runID, rec)
				// Backstop: cold-despite-warm-manifest means environment
				// mismatch (fingerprints catch known causes; this catches the rest).
				warnColdDespiteManifest(gets, atomic.LoadInt64(&totalHits), int64(len(mergedEntries)))
			}
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
					if b, err := os.ReadFile(local.resolve(actionID)); err == nil {
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
					resp.DiskPath = local.resolve(actionID)
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
				// In-flight coordination: if a prefetch (or another get) is
				// already downloading this entry, wait for it and reuse the
				// local result — a get joining a prefetch is a "conversion"
				// (prefetch fell behind need; each one cost a blocked worker).
				if e := inflights.join(actionID); e != nil {
					waitStart := time.Now()
					<-e.done
					// The download finished (or failed); local is the truth.
					if body, meta, err := local.Get(actionID); err == nil && len(meta) > 0 {
						resp.DiskPath = local.resolve(actionID)
						if raw, err := hex.DecodeString(string(meta)); err == nil {
							resp.OutputID = raw
						}
						resp.Size = int64(len(body))
						atomic.AddInt64(&conversions, 1)
						if ms := time.Since(waitStart).Milliseconds(); ms > 100 {
							log.Printf("gocache-serve: conversion %s waited %dms (prefetch started %v ago)",
								shortID(actionID), ms, time.Since(e.started).Round(time.Millisecond))
						}
						break
					}
					// Download failed — fall through to our own fetch.
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
				// Register our download so concurrent gets join it and the
				// prefetch skips it.
				mine := inflights.start(actionID, "get")
				defer func() {
					if mine != nil {
						inflights.finish(actionID)
					}
				}()
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
