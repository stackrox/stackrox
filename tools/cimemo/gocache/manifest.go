// Session manifests and startup prefetch.
//
// Why prefetch exists: go's GOCACHEPROG client requests actions at its
// own build-parallelism rate (~GOMAXPROCS outstanding requests), so
// per-request cloud serving is request-rate-bound no matter how fast
// the backend is (measured: 46 req/s served vs 86+ req/s available
// from trivially parallel curl). The fix — ported from the CA work's
// gocache-serve — is to front-run go: record each session's action
// set, and at startup bulk-download the expected working set before go
// asks for it.
//
// Manifest key: module identity = sha256(go.mod content + build
// config). Coarser than per-commit keys on purpose — a one-file PR
// still shares ~the whole working set with its parent (the Level-2
// lockfile-keyed side cache from the cimemo design).
package gocache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SessionManifest is the recorded working set for a module identity.
type SessionManifest struct {
	Actions []string `json:"actions"` // actionID hex, most recent last
}

const manifestMaxActions = 20000

// manifestKey derives the module identity from the working directory
// (go.mod found by walking up, like go itself) + build config env.
func manifestKey() string {
	h := sha256.New()
	mod := findGoMod()
	if mod != "" {
		data, err := os.ReadFile(mod)
		if err == nil {
			h.Write(data)
		}
	}
	for _, v := range []string{"CGO_ENABLED", "GOOS", "GOARCH", "GOFLAGS"} {
		fmt.Fprintf(h, "|%s=%s", v, os.Getenv(v))
	}
	// Scope: the build target (set by cimemo main from the go command's
	// package patterns). Module-identity-only keys prefetch the union of
	// every target's working set — the CI serve-change leg prefetched
	// 17K actions for a 5K build. Scoped keys fetch exactly the needed
	// set; the first build of a target falls back to the module-wide
	// behavior naturally (its manifest starts empty).
	if scope := os.Getenv("CIMEMO_GOCACHE_SCOPE"); scope != "" {
		fmt.Fprintf(h, "|scope=%s", scope)
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func findGoMod() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		p := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// loadSessionManifest fetches and parses the cloud manifest, or an
// empty one on any failure (prefetch is best-effort by design).
func loadSessionManifest(r *Remote) *SessionManifest {
	m := &SessionManifest{}
	data, err := r.Get("m/" + manifestKey())
	if err != nil || data == nil {
		return m
	}
	json.Unmarshal(data, m)
	return m
}

// saveSessionManifest merges this session's actions into the cloud
// manifest (union, bounded to the most recent N) and uploads it.
func saveSessionManifest(r *Remote, sessionActions []string) {
	if len(sessionActions) == 0 {
		return
	}
	old := loadSessionManifest(r)
	seen := make(map[string]bool, len(old.Actions)+len(sessionActions))
	var merged []string
	for _, a := range append(append([]string{}, old.Actions...), sessionActions...) {
		if !seen[a] {
			seen[a] = true
			merged = append(merged, a)
		}
	}
	if len(merged) > manifestMaxActions {
		merged = merged[len(merged)-manifestMaxActions:]
	}
	data, _ := json.Marshal(&SessionManifest{Actions: merged})
	if err := r.put("m/"+manifestKey(), data); err != nil {
		logf("manifest upload: %v", err)
	}
}

// prefetch bulk-downloads the manifest's actions into the local cache.
// Returns when done (or failed — always returns, prefetch is
// best-effort). Concurrency is bounded; existing local entries are
// skipped (content-addressed, same bytes).
func prefetch(r *Remote, cacheDir string, neg *Negatives, tt *TimeoutTracker, done chan<- struct{}) {
	defer close(done)
	if os.Getenv("CIMEMO_GOCACHE_NOPREFETCH") == "1" {
		return
	}
	m := loadSessionManifest(r)
	if len(m.Actions) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 32) // bounded workers
	var fetched, skipped int64
	var mu sync.Mutex

	for _, actionID := range m.Actions {
		if _, _, ok := readActionRecord(cacheDir, actionID); ok {
			mu.Lock()
			skipped++
			mu.Unlock()
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(aid string) {
			defer wg.Done()
			defer func() { <-sem }()
			if remoteGetAction(r, cacheDir, aid, neg, tt) {
				mu.Lock()
				fetched++
				mu.Unlock()
			}
		}(actionID)
	}
	wg.Wait()
	logf("prefetch: %d actions in manifest, %d fetched, %d already local, %d absent",
		len(m.Actions), fetched, skipped, int64(len(m.Actions))-fetched-skipped)
}
