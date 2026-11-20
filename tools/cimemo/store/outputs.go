package store

import (
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
		if err != nil || info.IsDir() {
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
		parts := strings.Split(rel, "/")
		if strings.HasPrefix(parts[len(parts)-1], ".") {
			return nil
		}
		for _, part := range parts[:len(parts)-1] {
			if strings.HasPrefix(part, ".") {
				return filepath.SkipDir
			}
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
