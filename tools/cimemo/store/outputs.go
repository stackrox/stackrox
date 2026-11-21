package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SnapshotOutputs stores ONLY the files a build produced: files that
// (a) are not in the input read-set, and (b) were created or modified
// after builtAt. Everything else — user source, files edited before the
// build — is excluded, so restoring can never clobber user files.
//
// This is the correctness-critical distinction from SnapshotDir (which
// captured the whole tree and, on a warm hit, overwrote non-input
// edits — see SENSORS.md "Correctness rules").
func (s *Store) SnapshotOutputs(key, dir string, readSet []string, builtAt int64) (*Manifest, error) {
	inputs := make(map[string]bool, len(readSet))
	for _, f := range readSet {
		inputs[filepath.ToSlash(f)] = true
	}

	m := &Manifest{
		Key:     key,
		Created: time.Now(),
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Hidden dirs are pruned (VCS, caches) EXCEPT when the
			// build itself created them (mtime > builtAt): build
			// systems keep validity state in dot-dirs (cargo's
			// target/debug/.fingerprint) that a receipt must restore
			// for the tool to accept the outputs.
			if strings.HasPrefix(info.Name(), ".") && info.ModTime().UnixNano() <= builtAt {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)

		// Never snapshot cache/VCS internals.
		if strings.HasPrefix(rel, ".cimemo") || strings.HasPrefix(rel, ".git/") {
			return nil
		}
		// Dot-FILES (e.g. .DS_Store) are skipped individually; dot-DIRECTORIES
		// (.git, .idea) are pruned. SkipDir on a file would skip the rest of
		// its directory — .DS_Store in the repo root would silently end the
		// walk and capture nothing.
		// Hidden FILES are skipped (state files are handled at the
		// directory level below).
		if strings.HasPrefix(filepath.Base(rel), ".") {
			return nil
		}

		// Rule (a): inputs are never outputs.
		if inputs[rel] {
			return nil
		}
		// Rule (b): only files the build touched.
		if info.ModTime().UnixNano() <= builtAt {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			if os.Getenv("CIMEMO_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "[dbg] open failed %s: %v\n", rel, err)
			}
			return nil
		}
		defer f.Close()
		hash, size, err := s.PutBlob(f)
		if err != nil {
			return nil
		}
		m.Blobs = append(m.Blobs, BlobRef{Hash: hash, Path: rel, Size: size})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Record directories the build CREATED (mtime > builtAt), even
	// empty ones: make-style builds mkdir output dirs (collector's
	// container/libs) that hold no files — invisible to the blob set
	// but their absence must not invalidate input-dir listings.
	dirs := map[string]bool{}
	err2 := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") && info.ModTime().UnixNano() <= builtAt {
			return filepath.SkipDir
		}
		if info.ModTime().UnixNano() > builtAt {
			dirs[rel] = true
		}
		return nil
	})
	if err2 == nil {
		for d := range dirs {
			m.OutputDirs = append(m.OutputDirs, d)
		}
		sort.Strings(m.OutputDirs)
	}

	sort.Slice(m.Blobs, func(i, j int) bool { return m.Blobs[i].Path < m.Blobs[j].Path })
	return m, nil
}

// OutputHashes returns path → content hash for every output blob in a
// manifest — the "what did this build produce" ground truth that the
// learned model diffs against.
func (m *Manifest) OutputHashes() map[string]string {
	out := make(map[string]string, len(m.Blobs))
	for _, b := range m.Blobs {
		out[b.Path] = b.Hash
	}
	return out
}

// LatestManifestForCommand returns the most recently created manifest
// for a command, or nil. (ManifestsForCommand returns ReadDir order,
// which is filename-sorted, not time-sorted.)
func (s *Store) LatestManifestForCommand(command string) *Manifest {
	var best *Manifest
	for _, m := range s.ManifestsForCommand(command) {
		if best == nil || m.Created.After(best.Created) {
			best = m
		}
	}
	return best
}

// formatReasons renders validation reasons compactly for logs.
func FormatReasons(reasons []string) string {
	if len(reasons) > 3 {
		reasons = append(append([]string{}, reasons[:3]...), fmt.Sprintf("… +%d more", len(reasons)-3))
	}
	return strings.Join(reasons, ", ")
}

// LatestForClosure returns the most recent manifest with the given
// closure hash (excluding excludeKey), or nil. The nondeterminism
// detector's index: same closure + different key/output = divergence.
func (s *Store) LatestForClosure(closureHash, excludeKey string) *Manifest {
	var best *Manifest
	manifestDir := filepath.Join(s.root, "manifests")
	entries, _ := os.ReadDir(manifestDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(manifestDir, e.Name()))
		if err != nil {
			continue
		}
		var m Manifest
		if json.Unmarshal(data, &m) == nil && m.ClosureHash == closureHash && m.Key != excludeKey {
			if best == nil || m.Created.After(best.Created) {
				best = &m
			}
		}
	}
	return best
}
