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
	"github.com/davdhacs/cimemo/dissect"
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
	case args[0] == "dissect" && len(args) == 3:
		rep, err := dissect.Diff(args[1], args[2])
		if err != nil {
			fatal("dissect: %v", err)
		}
		fmt.Print(rep)
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
	isDocker := isDockerBuild(args)
	var toolchain string
	if isGo {
		// Toolchain + build config: CGO_ENABLED, GOOS, GOARCH, GOFLAGS
		// change outputs without changing any file content — they must
		// be in the key or a warm hit can serve a wrong-binary.
		toolchain = key.BuildConfig(wd)
	}

	start := time.Now()

	// === Commit-receipt fast path ===
	// A clean tree at HEAD is exactly determined: SHA → tree → closure.
	// This skips `go list`, closure hashing, and (in CI) the module
	// cache + mtime normalization the content path needs. Falls
	// through silently to the slower paths when the tree is dirty or
	// no commit receipt exists.
	if !forceFull && !isDocker {
		keyCommand := command
		if ck, ok := commitReceiptKey(wd, keyCommand, toolchain); ok && st.HasManifest(ck) {
			if m, err := st.GetManifest(ck); err == nil && len(m.Blobs) > 0 {
				verifyRestored(m, wd)
				if count, bytes, err := st.RestoreManifest(m, wd); err == nil {
					fmt.Printf("[cimemo] CACHE HIT (commit): %s — clean tree at HEAD implies closure\n", ck)
					fmt.Printf("[cimemo]   %d outputs, %.1f MB (%s)\n",
						count, float64(bytes)/1024/1024, time.Since(start).Round(time.Millisecond))
					recordHit(mod, command)
					return
				}
			}
		}
	}

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
				if len(m.ImageOutputs) > 0 {
					// Image receipt: re-tag instead of restoring files.
					allOK := true
					for _, ref := range m.ImageOutputs {
						if err := dockerRetag(ref.ImageID, ref.Tag); err != nil {
							fmt.Fprintf(os.Stderr, "[cimemo] retag %s failed: %v — rebuilding\n", ref.Tag, err)
							allOK = false
							break
						}
					}
					if allOK {
						fmt.Printf("[cimemo] CACHE HIT (image re-tag): %s\n", m.Key)
						for _, ref := range m.ImageOutputs {
							fmt.Printf("[cimemo]   %s -> %s\n", ref.ImageID[:19], ref.Tag)
						}
						recordHit(mod, command)
						return
					}
				}
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

	// Docker keyed check BEFORE running: read-set is git-tracked
	// (computable pre-build), and the receipt re-tags instead of
	// restoring files.
	if isDocker && !forceFull {
		if ctxFiles := dockerContextClosure(wd, args); len(ctxFiles) > 0 {
			dk := key.FromFiles(wd, ctxFiles, dockerKeyCommand(command, args)+"|"+toolchain+vcsFold(wd))
			if st.HasManifest(dk) {
				if m, err := st.GetManifest(dk); err == nil && len(m.ImageOutputs) > 0 {
					allOK := true
					for _, ref := range m.ImageOutputs {
						if err := dockerRetag(ref.ImageID, ref.Tag); err != nil {
							allOK = false
							break
						}
					}
					if allOK {
						// Also satisfy the tags this invocation asked for.
						for _, t := range dockerBuildTags(args) {
							dockerRetag(m.ImageOutputs[0].ImageID, t)
						}
						fmt.Printf("[cimemo] CACHE HIT (image re-tag): %s\n", dk)
						for _, ref := range m.ImageOutputs {
							fmt.Printf("[cimemo]   %s -> %s\n", ref.ImageID[:19], ref.Tag)
						}
						recordHit(mod, command)
						return
					}
				}
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
		k := key.FromFiles(wd, readSet, dockerKeyCommand(command, args)+"|"+toolchain+vcsFold(wd))
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
				"CIMEMO_GOCACHE_STATS="+statsPath,
				"CIMEMO_GOCACHE_SCOPE="+strings.Join(goPatterns, " "))
		}
	}

	if err := cmd.Start(); err != nil {
		fatal("start: %v", err)
	}
	if !isGo {
		// Linux: poll the whole process tree via /proc (workers do the
		// source reads; lsof on the top PID only sees the orchestrator).
		// Elsewhere: lsof on the top PID.
		if _, err := os.Stat("/proc"); err == nil {
			go reads.TrackProcTree(cmd.Process.Pid, 20*time.Millisecond)
		} else {
			go reads.Track(cmd.Process.Pid, 50*time.Millisecond)
		}
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
	if isDocker {
		// The build context IS the input closure — including files
		// that are outputs of earlier commands (binaries COPYied into
		// the image). Git-tracked fallback is unsound for image builds
		// (untracked context files would be invisible to the key).
		readSet = dockerContextClosure(wd, args)
		strategy = store.StrategyFallback
	} else if !isGo {
		if observed := filterBuildState(wd, reads.Files(wd), builtAt); len(observed) > 0 {
			readSet = observed
			strategy = store.StrategyObserved
			// Orchestrator blindness: builds that delegate into
			// containers (make -> docker exec -> cmake) do their real
			// reads where the host cannot observe them — lsof saw ONE
			// file (the Makefile) for the collector build. Union with
			// git-tracked files so the key stays sound; observation
			// then only ADDS precision when it catches real reads.
			if gitFiles := gitTrackedFiles(wd); gitFiles != nil {
				readSet = unionFiles(readSet, gitFiles)
			}
		} else if gitFiles := gitTrackedFiles(wd); gitFiles != nil {
			readSet = gitFiles
			strategy = store.StrategyFallback
		}
	}
	if len(readSet) == 0 {
		fmt.Printf("[cimemo] no read-set derivable (observed raw=%d, git ok=%v) — output not cached\n",
			len(reads.Files(wd)), gitTrackedFiles(wd) != nil)
		return
	}

	// === Snapshot OUTPUTS ONLY and record provenance ===
	keyCommand := command
	if isDocker {
		// Tags are output NAMES, not inputs: the same context built to
		// different tags is the same receipt (found via the divergence
		// detector: same closure, same image, different key).
		for _, t := range dockerBuildTags(args) {
			keyCommand = strings.ReplaceAll(keyCommand, " "+t+" ", " ")
		}
	}
	k := key.FromFiles(wd, readSet, keyCommand+"|"+toolchain+vcsFold(wd))
	var m *store.Manifest
	if isDocker {
		// Image-level receipt: outputs are image IDs, not files.
		m = &store.Manifest{Key: k, Created: time.Now()}
		for _, tag := range dockerBuildTags(args) {
			id, ierr := dockerImageID(tag)
			if ierr != nil {
				fatal("docker inspect %s: %v", tag, ierr)
			}
			m.ImageOutputs = append(m.ImageOutputs, store.ImageRef{ImageID: id, Tag: tag})
		}
	} else {
		m, err = st.SnapshotOutputs(k, wd, readSet, builtAt)
		if err != nil {
			fatal("snapshot: %v", err)
		}
	}
	m.Command = command
	m.Strategy = strategy
	m.ReadSet = readSet
	// Every ancestor directory of every output is an output dir:
	// wiping any of them (cmake-build/, collector/container/bin) must
	// not invalidate input-dir listings.
	outputDirs := map[string]bool{}
	for _, b := range m.Blobs {
		p := b.Path
		for {
			if i := strings.LastIndexByte(p, '/'); i > 0 {
				p = p[:i]
				outputDirs[p] = true
			} else {
				break
			}
		}
	}
	for _, d := range m.OutputDirs {
		outputDirs[d] = true
	}
	m.InputHashes, m.DirListings = store.BuildProvenanceFiltered(wd, readSet, outputDirs)
	m.Closure = closure
	m.Toolchain = toolchain
	m.BuiltAt = builtAt
	m.ActionStats = actionStats
	m.VCSCommit = gitHead(wd)
	m.ClosureHash = key.ClosureHash(wd, readSet)

	// Nondeterminism detector: same closure, different key or different
	// output = something invisible changed (or the build is
	// nondeterministic). Dissect and warn.
	if prev := st.LatestForClosure(m.ClosureHash, m.Key); prev != nil {
		warnDivergence(st, prev, m, wd)
	}

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
	// Commit receipt: same manifest under the git-HEAD key (blobs are
	// content-addressed — zero extra storage; just a second index).
	if isDocker == false && m.KeyKind == "" {
		if ck, ok := commitReceiptKey(wd, keyCommand, toolchain); ok {
			cm := *m
			cm.Key = ck
			cm.KeyKind = "commit"
			if err := st.PutManifest(&cm); err == nil {
				fmt.Printf("[cimemo] commit receipt: %s\n", ck)
			}
		}
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

// dockerKeyCommand strips output tags from a docker build command for
// keying (tags are output names, not inputs).
func dockerKeyCommand(command string, args []string) string {
	if !isDockerBuild(args) {
		return command
	}
	for _, t := range dockerBuildTags(args) {
		command = strings.ReplaceAll(command, " "+t+" ", " ")
	}
	return command
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
	// -s: stage mode prefixes. Submodule entries are gitlinks (mode
	// 160000): hashing them as files fails silently, making submodule
	// content invisible to the key (found via the fact case study —
	// its build reads protos from third_party/stackrox). Replace each
	// gitlink with the submodule's HEAD commit as a pseudo-input.
	// -c safe.directory: tar-copied and container checkouts trip git's
	// ownership check, silently degrading cimemo to no-cache (found in
	// the fact-in-container case study).
	cmd := exec.Command("git", "-C", dir, "-c", "safe.directory="+dir, "ls-files", "-s")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		mode, path := fields[0], fields[3]
		if mode == "160000" {
			// path is the submodule dir; its HEAD is the content key
			if head, err := exec.Command("git", "-C", filepath.Join(dir, path), "-c", "safe.directory="+filepath.Join(dir, path), "rev-parse", "HEAD").Output(); err == nil {
				files = append(files, path+"@gitlink:"+strings.TrimSpace(string(head)))
			}
			continue
		}
		files = append(files, path)
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
	out, err := exec.Command("git", "-C", wd, "-c", "safe.directory="+wd, "rev-parse", "HEAD").Output()
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
	out, err := exec.Command("git", "-C", wd, "-c", "safe.directory="+wd, "rev-parse", "HEAD").Output()
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

// LatestForClosure-style divergence warning: compare the outputs of
// two builds with identical input closures.
func warnDivergence(st *store.Store, prev, cur *store.Manifest, wd string) {
	sameOutputs := len(prev.Blobs) == len(cur.Blobs)
	if sameOutputs {
		for i := range prev.Blobs {
			if prev.Blobs[i].Hash != cur.Blobs[i].Hash {
				sameOutputs = false
				break
			}
		}
	}
	fmt.Printf("[cimemo] DIVERGENCE: same input closure built twice with different results\n")
	fmt.Printf("[cimemo]   previous key %s, current key %s, outputs identical: %v\n",
		prev.Key, cur.Key, sameOutputs)
	if sameOutputs {
		fmt.Printf("[cimemo]   (outputs match — only the key moved: hidden input like commit/env/toolchain)\n")
		return
	}
	// Outputs differ with identical inputs: dissect.
	for i := range cur.Blobs {
		if i >= len(prev.Blobs) || prev.Blobs[i].Hash != cur.Blobs[i].Hash {
			a := filepath.Join(".cimemo", "blobs", prev.Blobs[i].Hash[:2], prev.Blobs[i].Hash)
			b := filepath.Join(wd, cur.Blobs[i].Path)
			if rep, err := dissect.Diff(a, b); err == nil {
				if rep.ContentEquivalent {
					fmt.Printf("[cimemo]   dissection of %s: content-equivalent (archive naming only) — not a divergence\n", cur.Blobs[i].Path)
					return
				}
				fmt.Printf("[cimemo]   dissection of %s:\n", cur.Blobs[i].Path)
				for _, line := range strings.Split(rep.String(), "\n") {
					if line != "" {
						fmt.Printf("[cimemo]     %s\n", line)
					}
				}
			}
			break
		}
	}
}

// filterBuildState removes observed reads that live under OUTPUT ROOTS:
// top-level directories containing build-fresh files (cargo's target/,
// cimemo CI's bin-exp/). Build systems read their own intermediate
// state there (cargo's fingerprints) — neither source nor stable
// input; keeping it makes receipt validation fail whenever outputs
// are wiped. Ancestor-mtime was tried first and is too fragile: any
// stray touch of a source dir nukes the whole read-set (fact case
// study).
func filterBuildState(wd string, files []string, builtAt int64) []string {
	roots := outputRoots(wd, builtAt)
	var out []string
	for _, f := range files {
		top := f
		if i := strings.IndexByte(f, '/'); i > 0 {
			top = f[:i]
		}
		if !roots[top] {
			out = append(out, f)
		}
	}
	return out
}

// outputRoots finds top-level directories containing anything modified
// after builtAt (bounded walk).
func outputRoots(wd string, builtAt int64) map[string]bool {
	roots := map[string]bool{}
	entries, err := os.ReadDir(wd)
	if err != nil {
		return roots
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if dirHasRecent(filepath.Join(wd, e.Name()), builtAt, 0) {
			roots[e.Name()] = true
		}
	}
	return roots
}

func dirHasRecent(dir string, builtAt int64, depth int) bool {
	if depth > 6 {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > builtAt {
			return true
		}
		if e.IsDir() && dirHasRecent(filepath.Join(dir, e.Name()), builtAt, depth+1) {
			return true
		}
	}
	return false
}

func unionFiles(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var out []string
	for _, f := range a {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	for _, f := range b {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}
