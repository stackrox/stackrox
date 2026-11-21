// cimemo: content-addressed memoization for any build command.
//
// v0.4 architecture — multi-sensor with learned edges (see SENSORS.md):
//
//	COLD RUN:  discover the read-set via the sensor stack (go list -deps
//	           for Go — exact; lsof observation otherwise — heuristic),
//	           run the command, snapshot ONLY the outputs, record
//	           provenance in the manifest, and record the build in the
//	           learned model (edges from co-changes).
//
//	WARM RUN:  validate the latest manifest (mtime fast path → hash
//	           verify → dir-listing check → toolchain check), restore
//	           outputs on hit.
//
//	--shadow:  compute learned predictions but do NOT trust them;
//	           build fully anyway, score predictions against ground
//	           truth, harvest evidence. Zero behavioral difference.
//	           Intended for master/nightly runs.
//
//	audit:     force a full rebuild, score the model, repair missed
//	           edges, report divergences. For release certainty.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davdhacs/cimemo/buildinfo"
	"github.com/davdhacs/cimemo/gocache"
	"github.com/davdhacs/cimemo/key"
	"github.com/davdhacs/cimemo/model"
	"github.com/davdhacs/cimemo/observe"
	"github.com/davdhacs/cimemo/store"
)

const version = "0.4.0"

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	// Global flags before the command.
	shadow := false
	args := os.Args[1:]
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		switch args[0] {
		case "--shadow":
			shadow = true
		case "--trust-learned":
			// Reserved: allow learned edges to serve hits. Not yet
			// implemented — predictions stay advisory until they've
			// survived audit validation, and even then only per-output.
			fmt.Fprintln(os.Stderr, "[cimemo] --trust-learned is not implemented yet; predictions stay advisory")
		default:
			fmt.Fprintf(os.Stderr, "[cimemo] unknown flag %s\n", args[0])
			usage()
		}
		args = args[1:]
	}
	if len(args) == 0 {
		usage()
	}

	switch {
	case args[0] == "gocache":
		// GOCACHEPROG entry point: re-exec'ed by `go` as the external
		// build cache. Pass-through over the standard GOCACHE layout,
		// writing a miss log to CIMEMO_GOCACHE_STATS.
		if err := gocache.Serve(); err != nil {
			fatal("gocache: %v", err)
		}
	case args[0] == "status":
		cmdStatus()
	case args[0] == "gc":
		cmdGC(args[1:])
	case args[0] == "clear":
		cmdClear()
	case args[0] == "audit" && len(args) > 1:
		cmdMemoize(args[1:], true)
	default:
		cmdMemoize(args, shadow)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "cimemo v%s — content-addressed memoization\n", version)
	fmt.Fprintf(os.Stderr, "\nUsage:\n")
	fmt.Fprintf(os.Stderr, "  cimemo [--shadow] <command> [args...]  memoize a command\n")
	fmt.Fprintf(os.Stderr, "  cimemo audit <command> [args...]       force full rebuild + score model\n")
	fmt.Fprintf(os.Stderr, "  cimemo status | gc | clear\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	fmt.Fprintf(os.Stderr, "  --shadow   score learned predictions without trusting them\n")
	os.Exit(1)
}

func cmdMemoize(args []string, forceFull bool) {
	command := strings.Join(args, " ")
	wd, _ := os.Getwd()

	st, err := store.New(filepath.Join(wd, ".cimemo"))
	if err != nil {
		fatal("store: %v", err)
	}
	// CIMEMO_REMOTE=1: receipts + output blobs in the cloud store, and
	// the gocache proxy's remote tier on for Go commands (both levels
	// share the backend and container).
	if os.Getenv("CIMEMO_REMOTE") == "1" {
		os.Setenv("CIMEMO_GOCACHE_REMOTE", "1")
		st.SetRemote(gocache.NewRemoteFromEnv())
	}
	mod, err := model.Load(filepath.Join(wd, ".cimemo", "model.json"))
	if err != nil {
		fatal("model: %v", err)
	}

	goPatterns := key.GoPatterns(args)
	isGo := goPatterns != nil
	var toolchain string
	if isGo {
		// Toolchain + build config: CGO_ENABLED, GOOS, GOARCH, GOFLAGS
		// change outputs without changing any file content — they must
		// be in the key or a warm hit can serve a wrong-binary.
		toolchain = key.BuildConfig(wd)
	}

	start := time.Now()

	// === Warm path (skipped for audit / shadow: both build fully) ===
	if !forceFull {
		if m := st.LatestManifestForCommand(command); m != nil {
			toolchainOK := !isGo || m.Toolchain == toolchain
			vcsOK := !isGo || !vcsFoldActive() || m.VCSCommit == gitHead(wd)
			if !toolchainOK {
				fmt.Printf("[cimemo] MISS: toolchain changed\n")
			} else if !vcsOK {
				fmt.Printf("[cimemo] MISS: commit changed (VCS-stamped build)\n")
			} else if valid, reasons := st.Validate(m, wd); valid {
				count, bytes, err := st.RestoreManifest(m, wd)
				if err == nil {
					fmt.Printf("[cimemo] CACHE HIT: %s\n", m.Key)
					fmt.Printf("[cimemo]   %d outputs, %.1f MB, strategy=%s (%s)\n",
						count, float64(bytes)/1024/1024, m.Strategy,
						time.Since(start).Round(time.Millisecond))
					recordHit(mod, command)
					return
				}
				fmt.Fprintf(os.Stderr, "[cimemo] restore error: %v — rebuilding\n", err)
			} else {
				fmt.Printf("[cimemo] MISS: %s\n", store.FormatReasons(reasons))
			}
		}
	}

	// === Miss: re-derive the read-set ===
	var readSet []string
	strategy := store.StrategyObserved
	closure := ""

	if isGo {
		strategy = store.StrategyGoList
		closure = "go list -deps " + strings.Join(goPatterns, " ")
		readSet, err = key.GoClosure(wd, goPatterns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[cimemo] %v — running un-memoized\n", err)
			runPlain(args)
			return
		}
		k := key.FromFiles(wd, readSet, command+"|"+toolchain+vcsFold(wd))
		if !forceFull && st.HasManifest(k) {
			if m, err := st.GetManifest(k); err == nil {
				count, bytes, err := st.RestoreManifest(m, wd)
				if err == nil {
					verifyRestored(m, wd)
					fmt.Printf("[cimemo] CACHE HIT (keyed): %s — closure re-derived\n", k)
					fmt.Printf("[cimemo]   %d outputs, %.1f MB (%s)\n",
						count, float64(bytes)/1024/1024,
						time.Since(start).Round(time.Millisecond))
					recordHit(mod, command)
					return
				}
			}
		}
	}

	// === Shadow: predict (advisory) before building ===
	var shadowPrediction map[string]bool
	var shadowChanged []string
	if forceFull {
		// Compute what changed since the last full build for scoring.
		current := probeInputs(wd, readSet)
		shadowChanged = mod.ChangedSince(model.BuildRecord{Inputs: current})
		if len(shadowChanged) > 0 {
			pred, conservative := mod.Predict(shadowChanged)
			shadowPrediction = pred
			if conservative {
				fmt.Printf("[cimemo] shadow: %d changed inputs include unknown files → prediction: conservative (all outputs)\n",
					len(shadowChanged))
			} else {
				fmt.Printf("[cimemo] shadow: %d changed inputs → prediction: %d output(s) affected\n",
					len(shadowChanged), len(pred))
			}
		}
	}

	// === Run the command ===
	if forceFull {
		fmt.Printf("[cimemo] audit: forcing full rebuild\n")
	} else {
		fmt.Printf("[cimemo] running: %s\n", command)
	}
	buildStart := time.Now()
	builtAt := buildStart.UnixNano()

	reads := observe.NewReads()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Go commands run through the GOCACHEPROG pass-through proxy:
	// cohabits the standard cache, produces the action miss log
	// (sensor #9). Failure to set it up is non-fatal — worst case is
	// go using its own cache directly.
	var statsPath string
	if isGo {
		if self, err := os.Executable(); err == nil {
			statsPath = filepath.Join(wd, ".cimemo", fmt.Sprintf("gocache-stats-%d.json", os.Getpid()))
			cmd.Env = append(os.Environ(),
				"GOCACHEPROG="+self+" gocache",
				"CIMEMO_GOCACHE_STATS="+statsPath)
		}
	}

	if err := cmd.Start(); err != nil {
		fatal("start: %v", err)
	}
	if !isGo {
		go reads.Track(cmd.Process.Pid, 50*time.Millisecond)
	}
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal("run: %v", err)
	}
	buildTime := time.Since(buildStart)

	// Harvest the action miss log (sensor #9): exact work accounting.
	var actionStats *gocache.Stats
	if statsPath != "" {
		if st, err := gocache.ReadStats(statsPath); err == nil && st.Gets+st.Puts > 0 {
			actionStats = st
			fmt.Printf("[cimemo] gocache: %d gets (%d hit, %d miss), %d puts (%.1f MB)\n",
				st.Gets, st.Hits, st.Misses, st.Puts, float64(st.PutBytes)/1024/1024)
		}
		os.Remove(statsPath)
	}

	// === Discover the read-set (non-Go commands) ===
	if !isGo {
		if observed := reads.Files(wd); len(observed) > 0 {
			readSet = observed
			strategy = store.StrategyObserved
		} else if gitFiles := gitTrackedFiles(wd); gitFiles != nil {
			readSet = gitFiles
			strategy = store.StrategyFallback
		}
	}
	if len(readSet) == 0 {
		fmt.Println("[cimemo] no read-set derivable — output not cached")
		return
	}

	// === Snapshot OUTPUTS ONLY and record provenance ===
	k := key.FromFiles(wd, readSet, command+"|"+toolchain+vcsFold(wd))
	m, err := st.SnapshotOutputs(k, wd, readSet, builtAt)
	if err != nil {
		fatal("snapshot: %v", err)
	}
	m.Command = command
	m.Strategy = strategy
	m.ReadSet = readSet
	m.InputHashes, m.DirListings = store.BuildProvenance(wd, readSet)
	m.Closure = closure
	m.Toolchain = toolchain
	m.BuiltAt = builtAt
	m.ActionStats = actionStats
	m.VCSCommit = gitHead(wd)

	// Buildinfo audit (tier 1): inspect go-binary outputs, record in
	// the manifest, warn on memoization-blind inputs.
	for _, ref := range m.Blobs {
		if info, err := buildinfo.Inspect(filepath.Join(wd, ref.Path)); err == nil && info != nil {
			m.BuildInfo = info
			for _, f := range buildinfo.Audit(info, command) {
				fmt.Printf("[cimemo] AUDIT %s: %s\n", f.Kind, f.Detail)
				fmt.Printf("[cimemo]   warn: %s\n", f.Warning)
			}
			break // first go binary is the story; others are similar
		}
	}
	if err := st.PutManifest(m); err != nil {
		fatal("manifest store: %v", err)
	}

	fmt.Printf("[cimemo] CACHED: %s\n", k)
	fmt.Printf("[cimemo]   %d outputs, %.1f MB, %d inputs, strategy=%s (build: %s)\n",
		len(m.Blobs), float64(sumSizes(m.Blobs))/1024/1024,
		len(readSet), strategy, buildTime.Round(time.Millisecond))

	// === Record in the learned model ===
	prev := mod.LastFullBuild()
	mod.RecordBuild(model.BuildRecord{
		Time:    time.Now(),
		Command: command,
		Full:    true,
		Shadow:  forceFull,
		Inputs:  m.InputHashes,
		Outputs: m.OutputHashes(),
	})

	// === Shadow/audit scoring ===
	if forceFull && prev != nil && len(shadowChanged) > 0 {
		actualChanged := diffOutputHashes(prev.Outputs, m.OutputHashes())
		rep := mod.Score(shadowChanged, actualChanged)
		fmt.Printf("[cimemo] audit: ground truth — %d output(s) changed; prediction had %d\n",
			len(actualChanged), len(shadowPrediction))
		if len(rep.MissedEdges) > 0 {
			fmt.Printf("[cimemo] audit: %d missed edge(s) repaired into model\n", len(rep.MissedEdges))
		}
		if rep.Spurious > 0 {
			fmt.Printf("[cimemo] audit: %d spurious prediction(s) (imprecision, not error)\n", rep.Spurious)
		}
		fmt.Printf("[cimemo] audit: recall %.0f%%\n", rep.Recall*100)
	} else if forceFull && prev == nil {
		fmt.Printf("[cimemo] audit: baseline recorded (no prior full build to score against)\n")
	}

	if err := mod.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "[cimemo] model save: %v\n", err)
	}
}

// recordHit logs a cache-hit build in the model (Full=false: produces
// no evidence — see model package safety rules).
func recordHit(mod *model.Model, command string) {
	mod.RecordBuild(model.BuildRecord{Time: time.Now(), Command: command, Full: false})
	if err := mod.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "[cimemo] model save: %v\n", err)
	}
}

// probeInputs hashes the read-set files without building anything —
// used to compute "what changed" for shadow scoring.
func probeInputs(wd string, readSet []string) map[string]string {
	out := make(map[string]string, len(readSet))
	for _, rel := range readSet {
		data, err := os.ReadFile(filepath.Join(wd, rel))
		if err != nil {
			continue
		}
		out[rel] = fmt.Sprintf("%x", sha256Sum(data))
	}
	return out
}

func diffOutputHashes(a, b map[string]string) []string {
	var out []string
	for k, v := range b {
		if av, ok := a[k]; !ok || av != v {
			out = append(out, k)
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

// runPlain executes the command with no memoization.
func runPlain(args []string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal("run: %v", err)
	}
}

func cmdStatus() {
	wd, _ := os.Getwd()
	storeDir := filepath.Join(wd, ".cimemo")
	if _, err := os.Stat(storeDir); os.IsNotExist(err) {
		fmt.Println("no cache (run cimemo <command> first)")
		return
	}

	var manifestCount, blobCount int
	var blobSize int64
	filepath.Walk(filepath.Join(storeDir, "manifests"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			manifestCount++
		}
		return nil
	})
	filepath.Walk(filepath.Join(storeDir, "blobs"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			blobCount++
			blobSize += info.Size()
		}
		return nil
	})

	fmt.Printf("store: %s\n", storeDir)
	fmt.Printf("  manifests: %d\n", manifestCount)
	fmt.Printf("  blobs:     %d (%.1f MB)\n", blobCount, float64(blobSize)/1024/1024)

	if mod, err := model.Load(filepath.Join(storeDir, "model.json")); err == nil {
		var validated int
		for _, e := range mod.Edges {
			if e.Validated {
				validated++
			}
		}
		fmt.Printf("  model:     %d edges (%d validated), %d builds\n",
			len(mod.Edges), validated, len(mod.Builds))
	}
}

func cmdGC(args []string) {
	maxAge := 720 * time.Hour
	for i := 0; i < len(args); i++ {
		if args[i] == "--max-age" && i+1 < len(args) {
			if d, err := time.ParseDuration(args[i+1]); err == nil {
				maxAge = d
			}
		}
	}
	wd, _ := os.Getwd()
	st, err := store.New(filepath.Join(wd, ".cimemo"))
	if err != nil {
		fatal("store: %v", err)
	}
	// CIMEMO_REMOTE=1: receipts + output blobs in the cloud store, and
	// the gocache proxy's remote tier on for Go commands (both levels
	// share the backend and container).
	if os.Getenv("CIMEMO_REMOTE") == "1" {
		os.Setenv("CIMEMO_GOCACHE_REMOTE", "1")
		st.SetRemote(gocache.NewRemoteFromEnv())
	}
	freed, removed, err := st.GC(maxAge)
	if err != nil {
		fatal("gc: %v", err)
	}
	fmt.Printf("[cimemo] GC: freed %.1f MB, removed %d items (max age: %s)\n",
		float64(freed)/1024/1024, removed, maxAge)
}

func cmdClear() {
	wd, _ := os.Getwd()
	if err := os.RemoveAll(filepath.Join(wd, ".cimemo")); err != nil {
		fatal("clear: %v", err)
	}
	fmt.Println("[cimemo] cache cleared")
}

func gitTrackedFiles(dir string) []string {
	cmd := exec.Command("git", "-C", dir, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files
}

func sumSizes(blobs []store.BlobRef) int64 {
	var total int64
	for _, b := range blobs {
		total += b.Size
	}
	return total
}

func sha256Sum(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[cimemo] ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// vcsFoldActive reports whether VCS state participates in keys for
// this run: stamping not provably off (GOFLAGS) and not disabled via
// .cimemo.json.
func vcsFoldActive() bool {
	if strings.Contains(os.Getenv("GOFLAGS"), "-buildvcs=false") {
		return false
	}
	return loadConfig(".").Deterministic.BuildVCS != "off"
}

// gitHead returns the current commit, or "" outside a repo.
func gitHead(wd string) string {
	out, err := exec.Command("git", "-C", wd, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// vcsFold returns the key fragment that makes VCS-stamped builds
// sound: git HEAD. Skipped when stamping is provably off (GOFLAGS
// carries -buildvcs=false) or the user opted out via .cimemo.json
// ("deterministic": {"buildvcs": "off"} — a promise that they disable
// stamping themselves; cherry-picks then hit by content, correctly).
func vcsFold(wd string) string {
	if !vcsFoldActive() {
		return ""
	}
	out, err := exec.Command("git", "-C", wd, "rev-parse", "HEAD").Output()
	if err != nil {
		return "" // not a git repo — nothing to fold
	}
	return "|commit:" + strings.TrimSpace(string(out))
}

// configFile is the checkable-in cimemo configuration.
type configFile struct {
	Deterministic struct {
		BuildVCS string `json:"buildvcs"` // "off" to skip VCS key-folding
	} `json:"deterministic"`
}

func loadConfig(wd string) configFile {
	var c configFile
	data, err := os.ReadFile(filepath.Join(wd, ".cimemo.json"))
	if err == nil {
		json.Unmarshal(data, &c)
	}
	return c
}

// verifyRestored re-inspects a restored go binary against the
// manifest's recorded buildinfo — the semantic identity check.
func verifyRestored(m *store.Manifest, wd string) {
	if m.BuildInfo == nil {
		return
	}
	for _, ref := range m.Blobs {
		info, err := buildinfo.Inspect(filepath.Join(wd, ref.Path))
		if err != nil || info == nil {
			continue
		}
		if ok, mismatch := buildinfo.Verify(m.BuildInfo, info); !ok {
			fmt.Printf("[cimemo] VERIFY FAILED on %s: %s — manifest may be stale\n", ref.Path, mismatch)
		} else {
			fmt.Printf("[cimemo] verified: %s buildinfo matches manifest\n", ref.Path)
		}
		break
	}
}
