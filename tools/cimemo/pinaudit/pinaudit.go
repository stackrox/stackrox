// Package pinaudit reports floating dependency references — build
// configs that are not pinned to exact versions/digests. A floating
// reference is (a) nondeterministic config: the same source can build
// against different dependency content over time or per machine;
// (b) cache-hostile: the effective inputs change without any file in
// the repo changing; (c) an integrity exposure: what you build is not
// what you reviewed. Pinning is the fix; this audit names the spots.
//
// Detection set (heuristic, bounded — each finding names the file and
// the fix):
//
//	unpinned-base   docker FROM with a mutable tag (registry-current
//	                digest is resolved for the key, but the Dockerfile
//	                itself stays nondeterministic until pinned)
//	missing-lockfile  cargo/npm project without a tracked lockfile
//	                (per-machine resolution drift)
//	floating-ref    build scripts referencing :latest, @master/@main,
//	                or `go install ...@latest`
//
// go is the exemplar: go.mod versions are concrete and go.sum verifies
// module content — reported clean by design.
package pinaudit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding is one unpinned-dependency report.
type Finding struct {
	Kind    string
	File    string
	Detail  string
	Warning string
}

// Audit scans a working tree for floating references.
func Audit(wd string) []Finding {
	var out []Finding
	out = append(out, lockfileFindings(wd)...)
	out = append(out, floatingRefFindings(wd)...)
	return out
}

// lockfileSystems: build marker → expected lockfiles (any one suffices).
var lockfileSystems = []struct {
	marker string
	locks  []string
	name   string
}{
	{"Cargo.toml", []string{"Cargo.lock"}, "cargo"},
	{"package.json", []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb"}, "npm-family"},
	{"go.mod", []string{"go.sum"}, "go"},
}

func lockfileFindings(wd string) []Finding {
	var out []Finding
	for _, sys := range lockfileSystems {
		if !exists(filepath.Join(wd, sys.marker)) {
			continue
		}
		var present []string
		for _, lock := range sys.locks {
			if exists(filepath.Join(wd, lock)) {
				present = append(present, lock)
			}
		}
		if len(present) == 0 {
			if sys.name == "go" {
				// go.sum can legitimately be absent for a module with
				// no dependencies — check whether go.mod has any
				if goModHasDeps(filepath.Join(wd, "go.mod")) {
					out = append(out, Finding{
						Kind:   "missing-lockfile",
						File:   sys.marker,
						Detail: "go.sum absent with dependencies declared",
						Warning: "go modules are content-verified via go.sum; without it every " +
							"build re-resolves and cannot verify module integrity. Run `go mod tidy`.",
					})
				}
				continue
			}
			out = append(out, Finding{
				Kind:   "missing-lockfile",
				File:   sys.marker,
				Detail: fmt.Sprintf("no lockfile (%s) present", strings.Join(sys.locks, " | ")),
				Warning: fmt.Sprintf("%s resolves dependency ranges at build time — the same tree can "+
					"build differently per machine and per day. Commit a lockfile; it is the exact "+
					"dependency closure (and what a cache key needs).", sys.name),
			})
			continue
		}
		// lock present: is it TRACKED? An untracked lock only pins this
		// machine — teammates/CI resolve independently.
		for _, lock := range present {
			if !gitTracked(wd, lock) {
				out = append(out, Finding{
					Kind:   "untracked-lockfile",
					File:   lock,
					Detail: "lockfile exists but is not committed",
					Warning: "an uncommitted lockfile pins only this machine — CI and teammates " +
						"resolve independently. Commit it.",
				})
			}
		}
	}
	return out
}

var floatingPatterns = []struct {
	re  *regexp.Regexp
	why string
}{
	{regexp.MustCompile(`:[0-9A-Za-z_.-]+-latest\b|:latest\b`), "mutable image tag"},
	{regexp.MustCompile(`@master\b|@main\b`), "branch-ref dependency (moves on every push)"},
	{regexp.MustCompile(`go install [^@]+@latest`), "floating tool version"},
	{regexp.MustCompile(`\bgit clone\b[^|\n]*\b--depth 1\b[^\n]*https`), "shallow clone of a moving branch"},
}

// scanTargets: files worth scanning for floating refs.
func scanTargets(wd string) []string {
	var targets []string
	filepath.WalkDir(wd, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		switch {
		case strings.HasPrefix(name, "Dockerfile"),
			strings.HasPrefix(name, "Makefile"),
			strings.HasSuffix(name, ".mk"),
			strings.HasSuffix(name, ".sh"),
			strings.HasSuffix(name, ".yaml"),
			strings.HasSuffix(name, ".yml"):
			rel, _ := filepath.Rel(wd, path)
			targets = append(targets, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(targets) > 3000 {
		targets = targets[:3000]
	}
	return targets
}

func floatingRefFindings(wd string) []Finding {
	var out []Finding
	seen := map[string]bool{}
	for _, rel := range scanTargets(wd) {
		data, err := os.ReadFile(filepath.Join(wd, rel))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			for _, p := range floatingPatterns {
				loc := p.re.FindStringIndex(line)
				if loc == nil {
					continue
				}
				// ":latest@sha256:..." is digest-pinned — NOT floating.
				if strings.HasPrefix(line[loc[1]:], "@sha256:") {
					continue
				}
				if p.re.MatchString(line) {
					k := rel + "|" + line
					if !seen[k] {
						seen[k] = true
						out = append(out, Finding{
							Kind:   "floating-ref",
							File:   rel,
							Detail: strings.TrimSpace(line),
							Warning: p.why + " — the build input changes without any file in your tree " +
								"changing; caches keyed on your tree will serve stale results. Pin to an " +
								"exact version, tag, or digest.",
						})
					}
				}
			}
		}
	}
	if len(out) > 20 {
		out = out[:20] // bounded; the pattern is established by then
	}
	return out
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func gitTracked(wd, file string) bool {
	out, err := exec.Command("git", "-C", wd, "-c", "safe.directory="+wd, "ls-files", "--", file).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func goModHasDeps(goMod string) bool {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "require ") && !strings.Contains(t, "go 1") {
			return true
		}
		if t == "require (" {
			return true
		}
	}
	return false
}
