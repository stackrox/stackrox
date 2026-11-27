package main

// Build-environment fingerprinting: the shared cache only hits when the
// local build environment matches the one that populated it — same Go
// toolchain, CGO setting, GOFLAGS, vendor content. A single mismatch
// (e.g. go 1.27.1 vs CI's 1.26.10) silently zeroes the hit rate with no
// error anywhere. These checks make that loud and actionable.

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BuildFingerprint is the set of environment inputs that enter every
// compile-cache ActionID. Recorded by CI, compared by dev machines.
type BuildFingerprint struct {
	Module     string `json:"module"` // go.mod module path (namespaces the fingerprint)
	GoVersion  string `json:"go_version"`
	CGO        string `json:"cgo"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	GOFLAGS    string `json:"goflags"` // normalized (sorted, space-joined)
	GoSumHash  string `json:"go_sum_hash"`
	Ts         int64  `json:"ts"`
	Source     string `json:"source"` // "ci" or "dev"
}

// normalizeGOFLAGS makes flag comparison order-insensitive.
func normalizeGOFLAGS(s string) string {
	fields := strings.Fields(s)
	sort.Strings(fields)
	return strings.Join(fields, " ")
}

// findModuleRoot walks up from dir looking for go.mod. Returns the root
// and the module path, or empty strings.
func findModuleRoot(dir string) (root, module string) {
	for d := dir; ; d = filepath.Dir(d) {
		data, err := os.ReadFile(filepath.Join(d, "go.mod"))
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return d, strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", ""
		}
	}
}

// collectFingerprint reads the EFFECTIVE build environment (what go will
// actually use, not just env vars). One `go env` exec, ~40ms — callers
// should gate this behind the disk memo so it runs once, not per process.
func collectFingerprint() BuildFingerprint {
	var fp BuildFingerprint
	fp.Ts = time.Now().Unix()
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		fp.Source = "ci"
	} else {
		fp.Source = "dev"
	}
	out, err := exec.Command("go", "env", "GOVERSION", "CGO_ENABLED", "GOOS", "GOARCH", "GOFLAGS").Output()
	if err != nil {
		return fp
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	get := func(i int, def string) string {
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			return strings.TrimSpace(lines[i])
		}
		return def
	}
	fp.GoVersion = get(0, "")
	fp.CGO = get(1, "")
	fp.GOOS = get(2, "")
	fp.GOARCH = get(3, "")
	fp.GOFLAGS = normalizeGOFLAGS(get(4, ""))

	if root, module := findModuleRoot(cwd()); root != "" {
		fp.Module = module
		if data, err := os.ReadFile(filepath.Join(root, "go.sum")); err == nil {
			sum := sha256.Sum256(data)
			fp.GoSumHash = fmt.Sprintf("%x", sum[:8])
		}
	}
	return fp
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

// fingerprintKey namespaces by module so building an unrelated module
// through the same cache never cross-compares fingerprints.
func fingerprintKey(fp BuildFingerprint) string {
	h := sha256.Sum256([]byte(fp.Module))
	return fmt.Sprintf("build-fingerprint/%x/%s-%s", h[:6], fp.GOOS, fp.GOARCH)
}

// fingerprintMemoPath caches the check result: a healthy match is
// re-verified at most once per day; a mismatch re-checks every run so a
// fixed environment is noticed immediately.
func fingerprintMemoPath() string {
	return filepath.Join(os.TempDir(), "gocache-serve-fingerprint-check")
}

type fingerprintMemo struct {
	Warned  bool   `json:"warned"` // last check mismatched
	Text    string `json:"text"`   // cached warning to re-print
	Ts      int64  `json:"ts"`
	Healthy bool   `json:"healthy"`
}

func loadFingerprintMemo() (fingerprintMemo, bool) {
	var m fingerprintMemo
	data, err := os.ReadFile(fingerprintMemoPath())
	if err != nil {
		return m, false
	}
	return m, json.Unmarshal(data, &m) == nil
}

func saveFingerprintMemo(m fingerprintMemo) {
	if data, err := json.Marshal(m); err == nil {
		os.WriteFile(fingerprintMemoPath(), data, 0o644)
	}
}

// compareFingerprints returns human-readable mismatches with remediation.
func compareFingerprints(local, remote BuildFingerprint) []string {
	var diffs []string
	if local.GoVersion != remote.GoVersion {
		diffs = append(diffs, fmt.Sprintf(
			"go version: yours=%s cache=%s  → export GOTOOLCHAIN=%s (GOTOOLCHAIN=auto does NOT downgrade a newer local go)",
			local.GoVersion, remote.GoVersion, remote.GoVersion))
	}
	if local.CGO != remote.CGO {
		diffs = append(diffs, fmt.Sprintf(
			"CGO_ENABLED: yours=%s cache=%s  → export CGO_ENABLED=%s",
			local.CGO, remote.CGO, remote.CGO))
	}
	if normalizeGOFLAGS(local.GOFLAGS) != normalizeGOFLAGS(remote.GOFLAGS) {
		diffs = append(diffs, fmt.Sprintf(
			"GOFLAGS: yours=%q cache=%q  → export GOFLAGS=%q",
			local.GOFLAGS, remote.GOFLAGS, remote.GOFLAGS))
	}
	// go.sum only comparable within the same module (checked by caller).
	if local.GoSumHash != "" && remote.GoSumHash != "" && local.GoSumHash != remote.GoSumHash {
		diffs = append(diffs,
			"go.sum differs from the cache's vendor tree  → restore it: "+
				"gocache-serve -fetch vendor-cache-$(sha256sum go.sum | cut -c1-16) /tmp/v.tar && tar -xf /tmp/v.tar")
	}
	return diffs
}

// maybeCheckFingerprint runs the dev-side mismatch check. CI runs
// (GITHUB_ACTIONS=true) instead WRITE the fingerprint (they are the
// source of truth). The check is memoized: healthy = once/day, mismatch
// = every run (so fixes are noticed; the ~90ms GET only costs the
// already-broken state). Uses the HUB (control plane): CI runners land
// in different regions, and a regionally-written fingerprint would be
// invisible to the hub-reading doctor — the same divergence that hit
// the manifests.
func maybeCheckFingerprint() {
	inCI := os.Getenv("GITHUB_ACTIONS") == "true"
	local := collectFingerprint()
	if local.GoVersion == "" {
		return // no go in PATH (odd context) — skip silently
	}
	b := controlBackend()
	key := fingerprintKey(local)
	if inCI {
		// Memo-gate the write: every CI process would otherwise PUT the
		// same content, and container versioning turns each into a new
		// version. One write per hour is plenty.
		if memo, ok := loadFingerprintMemo(); ok && memo.Healthy && time.Now().Unix()-memo.Ts < 3600 {
			return
		}
		data, _ := json.Marshal(local)
		if err := b.Put(key, data); err != nil {
			log.Printf("fingerprint write failed: %v", err)
			return
		}
		saveFingerprintMemo(fingerprintMemo{Healthy: true, Ts: time.Now().Unix()})
		return
	}
	// Dev: memo gate.
	memo, ok := loadFingerprintMemo()
	now := time.Now().Unix()
	if ok && memo.Healthy && now-memo.Ts < 86400 {
		return // healthy, checked today
	}
	body, err := b.Get(key)
	if err != nil || body == nil {
		return // no fingerprint recorded yet — nothing to compare
	}
	var remote BuildFingerprint
	if json.Unmarshal(body, &remote) != nil {
		return
	}
	var diffs []string
	if local.Module == remote.Module { // only compare within the same module
		diffs = compareFingerprints(local, remote)
	}
	if len(diffs) == 0 {
		saveFingerprintMemo(fingerprintMemo{Healthy: true, Ts: now})
		return
	}
	var sb strings.Builder
	sb.WriteString("WARNING: the shared build cache will MISS for this environment — fix the following:\n")
	for _, d := range diffs {
		fmt.Fprintf(&sb, "  %s\n", d)
	}
	sb.WriteString("  (details: gocache-serve -doctor)")
	log.Printf("gocache-serve: %s", sb.String())
	saveFingerprintMemo(fingerprintMemo{Warned: true, Text: sb.String(), Ts: now, Healthy: false})
}

// doctorCommand prints a full local-vs-cache comparison and exits
// non-zero on mismatch (scriptable for onboarding).
func doctorCommand() error {
	local := collectFingerprint()
	if local.GoVersion == "" {
		return fmt.Errorf("no go toolchain in PATH")
	}
	body, err := controlBackend().Get(fingerprintKey(local))
	if err != nil || body == nil {
		fmt.Printf("no fingerprint recorded for %s %s/%s — cache may be cold or CI has not run\n",
			local.Module, local.GOOS, local.GOARCH)
		return nil
	}
	var remote BuildFingerprint
	if err := json.Unmarshal(body, &remote); err != nil {
		return fmt.Errorf("fingerprint decode: %v", err)
	}
	fmt.Printf("local:  go=%s cgo=%s %s/%s flags=%q vendor=%s\n",
		local.GoVersion, local.CGO, local.GOOS, local.GOARCH, local.GOFLAGS, local.GoSumHash[:8])
	fmt.Printf("cache:  go=%s cgo=%s %s/%s flags=%q vendor=%s (recorded %s, by %s)\n",
		remote.GoVersion, remote.CGO, remote.GOOS, remote.GOARCH, remote.GOFLAGS,
		remote.GoSumHash[:8], time.Unix(remote.Ts, 0).UTC().Format(time.RFC3339), remote.Source)
	if local.Module != remote.Module {
		fmt.Println("note: different module than the recorded fingerprint — skipping detailed comparison")
		return nil
	}
	diffs := compareFingerprints(local, remote)
	if len(diffs) == 0 {
		fmt.Println("MATCH — this environment will hit the shared cache")
		return nil
	}
	fmt.Println("MISMATCHES:")
	for _, d := range diffs {
		fmt.Printf("  %s\n", d)
	}
	return fmt.Errorf("%d mismatch(es)", len(diffs))
}

// warnColdDespiteManifest is the hit-rate backstop at close: catches
// mismatch causes the fingerprint can't enumerate. A warm manifest with
// a near-zero hit rate means the environment diverged.
func warnColdDespiteManifest(gets, hits, manifestEntries int64) {
	if manifestEntries < 100 || gets < 200 {
		return // not enough signal
	}
	if hits*100 < gets*5 { // < 5% hits
		log.Printf("gocache-serve: WARNING: cache was cold (%d gets, %d hits) despite a warm manifest "+
			"(%d entries) — environment mismatch likely; run `gocache-serve -doctor`",
			gets, hits, manifestEntries)
	}
}
