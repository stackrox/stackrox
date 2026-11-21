package store

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/davdhacs/cimemo/key"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Cache-entry strategies. The strategy determines how a read-set was
// discovered and therefore how much the warm path can trust itself.
const (
	// StrategyGoList: read-set from `go list -deps` — exact, sound,
	// re-derivable before the command runs.
	StrategyGoList = "go-list"
	// StrategyObserved: read-set from lsof polling during execution —
	// heuristic (short opens missed), not re-derivable without re-running.
	StrategyObserved = "observed"
	// StrategyFallback: git-tracked or whole-tree scan — sound but
	// over-invalidates (README changes cause misses).
	StrategyFallback = "fallback"
)

// BuildProvenance records the state that Validate checks on the warm
// path: a content hash for every read-set file, and a listing hash
// for every directory containing them (catches new files entering an
// input directory without any existing file changing).
//
// Returns (inputHashes, dirListings).
func BuildProvenance(dir string, readSet []string) (map[string]string, map[string]string) {
	return BuildProvenanceFiltered(dir, readSet, nil)
}

// BuildProvenanceFiltered records provenance with directory listings
// that exclude the given names — callers pass the manifest's output
// roots so validation and recording agree (output churn must not
// invalidate input-dir listings).
func BuildProvenanceFiltered(dir string, readSet []string, exclude map[string]bool) (map[string]string, map[string]string) {
	inputs := make(map[string]string, len(readSet))
	for _, rel := range readSet {
		if v, ok := key.HashInput(dir, rel); ok {
			inputs[rel] = v
		}
	}

	dirs := make(map[string]bool)
	for _, rel := range readSet {
		dirs[filepath.Dir(rel)] = true
	}
	listings := make(map[string]string, len(dirs))
	for d := range dirs {
		listings[d] = dirListingHashFiltered(dir, d, exclude)
	}
	return inputs, listings
}

// dirListingHash hashes the sorted entry names of a directory.
// Names only — not content, not mtimes — so it answers exactly
// "did a file appear or disappear here?"
func dirListingHash(dir string) string {
	return dirListingHashFiltered(dir, ".", nil)
}

// dirListingHashFiltered hashes a directory's entry names, excluding
// entries whose RELATIVE PATH is an output dir or under one. Output
// trees can nest arbitrarily deep (collector: collector/container/bin)
// — name-at-one-level was too shallow and missed wiped output dirs.
// dirListingHashFiltered hashes the entry names of relDir (under
// root), excluding entries whose RELATIVE PATH is an output dir or
// under one. The exclusion set is keyed by full relative paths
// (collector/container/bin) — entries are joined with relDir before
// lookup; comparing bare names against paths never matched (the bug
// behind the collector's permanent "dir changed: collector/container/"
// receipt misses).
func dirListingHashFiltered(root, relDir string, excludeDirs map[string]bool) string {
	entries, err := os.ReadDir(filepath.Join(root, relDir))
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		entryPath := path.Join(relDir, e.Name())
		if excludeDirs[entryPath] {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	h := sha256.Sum256([]byte(strings.Join(names, "\n")))
	return hex.EncodeToString(h[:])
}

// Validate checks whether a manifest's cached output is still valid,
// using a two-tier check designed to keep the common path cheap:
//
//	tier 1: stat every read-set file — mtime <= BuiltAt means the
//	        content cannot have changed since the build (fast, ~3µs/file)
//	tier 2: for files with newer mtimes, hash and compare against
//	        the recorded input hash (catches mtime-only edits)
//	plus:   directory listing hashes (catches files added/removed
//	        from input directories — the closure-staleness case)
//
// Returns (valid, reasons). An empty reason list with valid=true means
// a clean hit. Manifests without provenance (legacy) are never valid —
// the caller must re-derive.
func (s *Store) Validate(m *Manifest, dir string) (bool, []string) {
	if len(m.ReadSet) == 0 || m.BuiltAt == 0 {
		return false, []string{"no provenance (legacy manifest)"}
	}

	var reasons []string

	for _, rel := range m.ReadSet {
		// Gitlinks hash via the submodule HEAD (not a file read).
		if strings.Contains(rel, key.GitlinkMarker) {
			if v, ok := key.HashInput(dir, rel); !ok {
				reasons = append(reasons, "deleted: "+rel)
			} else if m.InputHashes[rel] != "" && v != m.InputHashes[rel] {
				reasons = append(reasons, "changed: "+rel)
			}
			continue
		}
		abs := filepath.Join(dir, rel)
		info, err := os.Stat(abs)
		if err != nil {
			reasons = append(reasons, "deleted: "+rel)
			continue
		}
		if info.ModTime().UnixNano() <= m.BuiltAt {
			continue // cannot have changed since the build
		}
		// Newer mtime — verify content.
		v, ok := key.HashInput(dir, rel)
		if !ok {
			reasons = append(reasons, "unreadable: "+rel)
			continue
		}
		if v != m.InputHashes[rel] {
			reasons = append(reasons, "changed: "+rel)
		}
		// mtime newer but same content: not a reason, but the file's
		// new mtime stays — future validations hash it again. Cheap.
	}

	// Output dirs (every ancestor of each output path) are excluded
	// from listing checks: wiping/restoring outputs churns those
	// listings by design, at any depth (collector/container/bin).
	outputRoots := map[string]bool{}
	for _, b := range m.Blobs {
		p := b.Path
		for {
			if i := strings.LastIndexByte(p, '/'); i > 0 {
				p = p[:i]
				outputRoots[p] = true
			} else {
				break
			}
		}
	}
	for _, d := range m.OutputDirs {
		outputRoots[d] = true
	}
	for d, want := range m.DirListings {
		if got := dirListingHashFiltered(dir, d, outputRoots); got != want {
			reasons = append(reasons, "dir changed: "+d+"/")
		}
	}

	return len(reasons) == 0, reasons
}
