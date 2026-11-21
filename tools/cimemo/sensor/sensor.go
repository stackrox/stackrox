// Package sensor is the registry and fusion layer for dependency
// discovery sensors. Each sensor answers some variant of "which files
// does this command depend on?" with a different soundness/cost
// profile; fusion combines them (see SENSORS.md).
package sensor

import (
	"fmt"
	"os"
	"path/filepath"
)

// Tier ranks how much a sensor's answer can be trusted.
type Tier int

const (
	TierExact        Tier = iota // sound, re-derivable (go list, dep files)
	TierLearned                  // inferred from history; validated or not
	TierHeuristic                // observation with known blind spots (lsof)
	TierConservative             // over-approximates on purpose (git-tracked)
)

func (t Tier) String() string {
	switch t {
	case TierExact:
		return "exact"
	case TierLearned:
		return "learned"
	case TierHeuristic:
		return "heuristic"
	default:
		return "conservative"
	}
}

// Reading is one sensor's answer: the input files a command depends on.
type Reading struct {
	Sensor string
	Tier   Tier
	Files  []string
}

// Fusion combines readings by the SENSORS.md rules:
// union for coverage, exact beats heuristic. All readings are unioned
// (a heuristic never removes an exact sensor's files), and the fused
// tier is the strongest tier that contributed.
func Fuse(readings []Reading) Reading {
	if len(readings) == 0 {
		return Reading{}
	}
	seen := make(map[string]bool)
	var files []string
	best := TierConservative
	name := ""
	for _, r := range readings {
		if r.Tier < best {
			best = r.Tier
			name = r.Sensor
		}
		for _, f := range r.Files {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	if name == "" {
		name = readings[0].Sensor
	}
	return Reading{Sensor: name, Tier: best, Files: files}
}

// --- Active sensors are implemented in their own packages:
//   key.GoClosure   (exact, pre-exec)   — go list -deps
//   observe.Reads   (heuristic, during) — lsof polling
//   model.Model     (learned, pre-exec) — co-change history
//   watch.Monitor   (meta, continuous)  — fsnotify change distribution
//   store.Validate  (ground truth)      — mtime/hash/dir-listing diffs

// --- Planned sensors below. Each has a design sketch but no exercise
// against a real codebase yet; implement when the matching case study
// exists.

// DepFiles discovers inputs from compiler dependency files (.d files).
//
// TODO(sensor #7): gcc/clang -MD -MF <path>.d emits a make-style rule
// "output.o: src.c header.h ..." listing every header transitively
// read during compilation. Most C/C++ build systems already generate
// these and discard them; the sensor would glob **/*.d, parse the
// prerequisite lists, and union them. rustc: --emit=dep-info produces
// the equivalent. Needs a C/C++ case study to exercise.
func DepFiles(dir string) (Reading, error) {
	return Reading{}, fmt.Errorf("dep-files sensor: not implemented (needs C/C++ case study)")
}

// DryRunTrace discovers the build graph from a tool's verbose or
// dry-run mode.
//
// TODO(sensor #8): `go build -x`, `make -n`, `ninja -t commands`,
// `tsc --listFiles`, `mvn -X` all narrate the commands/inputs they
// would touch. Run the dry form once, extract file paths from argv,
// and treat the resulting set as an exact closure. Works generically
// for any tool with a verbose mode; no kernel APIs, no observation
// limits. Needs a non-Go case study (Makefile-based project) to
// exercise.
func DryRunTrace(dir string, args []string) (Reading, error) {
	return Reading{}, fmt.Errorf("dry-run trace sensor: not implemented (needs Makefile case study)")
}

// CacheProtocol discovers which compilation units were rebuilt from a
// compiler cache's hit/miss log.
//
// TODO(sensor #9): GOCACHEPROG (see tools/gocache-serve in the
// stackrox CA work — the protocol plumbing already exists there)
// reports ActionID→outputID per package; a miss reveals exactly which
// packages the input change invalidated. ccache -s / sccache logs are
// the same sensor for C/C++. This is an during-execution EXACT sensor
// and doubles as the reconciliation oracle for the learned model.
func CacheProtocol(dir string) (Reading, error) {
	return Reading{}, fmt.Errorf("cache-protocol sensor: not implemented (reuse gocache-serve)")
}

// Coverage discovers test→code mappings for test-selection memoization.
//
// TODO(sensor #10): `go test -coverprofile` (and jest/vitest coverage)
// maps each test to the code it exercises — the closure for "which
// tests must re-run for this change". Exact, already produced by most
// CI. Needs a test-selection case study to exercise.
func Coverage(dir string) (Reading, error) {
	return Reading{}, fmt.Errorf("coverage sensor: not implemented (test-selection case study)")
}

// GitHistory seeds the learned model from commit co-change statistics.
//
// TODO(sensor #11): `git log --name-status` gives years of
// (changed files, build result) pairs before cimemo has run once.
// Seed model edges from file co-change clusters to skip the cold-start
// convergence period. Statistical tier — seeds only; never serves hits
// without validation.
func GitHistory(dir string) (Reading, error) {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return Reading{}, fmt.Errorf("git-history sensor: not a git repo")
	}
	return Reading{}, fmt.Errorf("git-history sensor: not implemented")
}
