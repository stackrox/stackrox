// Package store provides a content-addressed blob store with
// manifests for CI memoization.
//
// Architecture:
//
//	Blobs   — raw file content, stored by SHA-256. Deduplicated.
//	Manifests — structured records mapping a cache key to a set of
//	           blob references with their target paths.
//
// A manifest answers: "for this key, what files should exist, and
// where do I find their content?" Multiple manifests referencing the
// same blob share storage automatically.
package store

import (
	"github.com/davdhacs/cimemo/buildinfo"
	"github.com/davdhacs/cimemo/gocache"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BlobRef is a reference to a stored blob.
type BlobRef struct {
	Hash string `json:"hash"` // SHA-256 of content
	Path string `json:"path"` // relative path for restoration
	Size int64  `json:"size"` // content size in bytes
}

// Manifest maps a cache key to a set of output blobs.
type Manifest struct {
	Key      string    `json:"key"`
	Blobs    []BlobRef `json:"blobs"`
	Created  time.Time `json:"created"`
	LastRead time.Time `json:"last_read"`
	Command  string    `json:"command,omitempty"`

	// Provenance: how the read-set was discovered and how to re-derive
	// it. Strategy picks the revalidation tier (see provenance.go);
	// Closure re-derives the read-set without re-running the command.
	Strategy    string            `json:"strategy,omitempty"`
	ReadSet     []string          `json:"read_set,omitempty"`
	InputHashes map[string]string `json:"input_hashes,omitempty"`
	DirListings map[string]string `json:"dir_listings,omitempty"`
	Closure     string            `json:"closure,omitempty"`
	Toolchain   string            `json:"toolchain,omitempty"`
	BuiltAt     int64             `json:"built_at,omitempty"`
	ActionStats *gocache.Stats    `json:"action_stats,omitempty"`
	BuildInfo   *buildinfo.Info   `json:"build_info,omitempty"`
	VCSCommit   string            `json:"vcs_commit,omitempty"`
}

// Store is the content-addressed blob store with manifests.
type Store struct {
	root   string // directory root
	remote *gocache.Remote
}

// New creates a Store rooted at dir.
func New(dir string) (*Store, error) {
	for _, d := range []string{dir, filepath.Join(dir, "blobs"), filepath.Join(dir, "manifests")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("store init %s: %w", d, err)
		}
	}
	return &Store{root: dir}, nil
}

// blobPath returns the on-disk path for a blob hash.
// Blobs are sharded by first 2 hex chars to avoid huge directories.
func (s *Store) blobPath(hash string) string {
	return filepath.Join(s.root, "blobs", hash[:2], hash)
}

// manifestPath returns the on-disk path for a manifest key.
func (s *Store) manifestPath(key string) string {
	return filepath.Join(s.root, "manifests", key+".json")
}

// PutBlob stores content and returns its hash.
func (s *Store) PutBlob(content io.Reader) (string, int64, error) {
	// Read content to compute hash
	data, err := io.ReadAll(content)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	p := s.blobPath(hash)
	if _, err := os.Stat(p); err == nil {
		// Already exists — dedup hit
		return hash, int64(len(data)), nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", 0, err
	}
	return hash, int64(len(data)), nil
}

// GetBlob returns content by hash, fetching from the remote store
// (with integrity verification) when not local.
func (s *Store) GetBlob(hash string) ([]byte, error) {
	data, err := os.ReadFile(s.blobPath(hash))
	if os.IsNotExist(err) && s.hasRemote() {
		if ferr := s.remoteFetchBlob(hash); ferr == nil {
			data, err = os.ReadFile(s.blobPath(hash))
		}
	}
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("blob not found: %s", hash)
	}
	return data, err
}

// writeAtomic + markBlob: small helpers for the remote tier.
func writeAtomic(path string, data []byte) error {
	os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) markBlob(hash string) {}

// PutManifest stores a manifest (and updates LastRead). With a remote
// attached, also uploads the manifest and its blobs — this is what
// makes receipts cross-machine.
func (s *Store) PutManifest(m *Manifest) error {
	m.LastRead = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.manifestPath(m.Key), data, 0o644); err != nil {
		return err
	}
	if s.hasRemote() {
		if err := s.remoteUploadManifest(m); err != nil {
			// Remote failure must not fail the build — the local
			// receipt is still valid on this machine.
			return nil
		}
	}
	return nil
}

// GetManifest retrieves a manifest by key, updating LastRead. Falls
// back to the remote store when the local copy is absent.
func (s *Store) GetManifest(key string) (*Manifest, error) {
	data, err := os.ReadFile(s.manifestPath(key))
	if os.IsNotExist(err) && s.hasRemote() {
		if m, rerr := s.remoteFetchManifest(key); rerr == nil && m != nil {
			// Cache it locally for subsequent reads.
			if jdata, jerr := json.MarshalIndent(m, "", "  "); jerr == nil {
				os.WriteFile(s.manifestPath(key), jdata, 0o644)
			}
			return m, nil
		}
	}
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("manifest not found: %s", key)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest parse %s: %w", key, err)
	}
	m.LastRead = time.Now()
	_ = os.WriteFile(s.manifestPath(key), data, 0o644) // update atime via rewrite
	return &m, nil
}

// HasManifest reports whether a manifest exists (locally or remote).
func (s *Store) HasManifest(key string) bool {
	if _, err := os.Stat(s.manifestPath(key)); err == nil {
		return true
	}
	if s.hasRemote() {
		ok, err := s.remote.ExistsPub(remoteManifestPrefix + key)
		return err == nil && ok
	}
	return false
}

// RestoreManifest restores all blobs referenced by a manifest into dir.
// Returns the number of files restored and total bytes.
func (s *Store) RestoreManifest(m *Manifest, dir string) (int, int64, error) {
	var count int
	var total int64
	for _, ref := range m.Blobs {
		data, err := s.GetBlob(ref.Hash)
		if err != nil {
			return count, total, fmt.Errorf("restore %s (blob %s): %w", ref.Path, ref.Hash[:12], err)
		}
		target := filepath.Join(dir, ref.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return count, total, err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return count, total, err
		}
		count++
		total += int64(len(data))
	}
	return count, total, nil
}

// SnapshotDir walks dir and stores every file as a blob, returning
// a Manifest with the results. Files matching exclude patterns are
// skipped.
func (s *Store) SnapshotDir(key, dir string, exclude []string) (*Manifest, error) {
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
		base := filepath.Base(rel)

		// Always skip hidden files and VCS dirs
		if strings.HasPrefix(base, ".") {
			return nil
		}
		parts := strings.Split(rel, "/")
		for _, part := range parts[:len(parts)-1] {
			if strings.HasPrefix(part, ".") {
				return filepath.SkipDir
			}
		}

		// Check exclusions
		for _, ex := range exclude {
			if matched, _ := filepath.Match(ex, rel); matched {
				return nil
			}
			if matched, _ := filepath.Match(ex, filepath.Base(rel)); matched {
				return nil
			}
			// Also check if any parent dir matches
			for _, part := range strings.Split(rel, "/") {
				if matched, _ := filepath.Match(ex, part); matched {
					return nil
				}
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return nil // skip unreadable
		}
		defer f.Close()

		hash, size, err := s.PutBlob(f)
		if err != nil {
			return nil // skip on error
		}

		m.Blobs = append(m.Blobs, BlobRef{
			Hash: hash,
			Path: rel,
			Size: size,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort for deterministic manifests
	sort.Slice(m.Blobs, func(i, j int) bool {
		return m.Blobs[i].Path < m.Blobs[j].Path
	})

	return m, nil
}

// OutputPaths returns all file paths that appear in any manifest,
// representing the union of all cached output files. Callers use this
// to exclude previous outputs from key computation, making keys stable
// across repeated runs of the same command.
func (s *Store) OutputPaths() []string {
	var paths []string
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
		if json.Unmarshal(data, &m) == nil {
			for _, ref := range m.Blobs {
				paths = append(paths, ref.Path)
			}
		}
	}
	return paths
}

// ManifestsForCommand returns all manifests that were created by the
// given command, sorted by creation time (oldest first).
func (s *Store) ManifestsForCommand(command string) []*Manifest {
	var out []*Manifest
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
		if json.Unmarshal(data, &m) == nil && m.Command == command {
			out = append(out, &m)
		}
	}
	return out
}

// GC removes blobs and manifests not accessed within maxAge.
// Returns bytes freed and count of items removed.
func (s *Store) GC(maxAge time.Duration) (int64, int, error) {
	cutoff := time.Now().Add(-maxAge)
	var freed int64
	var removed int

	// GC manifests
	manifestDir := filepath.Join(s.root, "manifests")
	entries, _ := os.ReadDir(manifestDir)
	var liveHashes = make(map[string]bool)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(manifestDir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(p)
			removed++
		} else {
			// Read manifest to find live blobs
			data, _ := os.ReadFile(p)
			var m Manifest
			if json.Unmarshal(data, &m) == nil {
				for _, ref := range m.Blobs {
					liveHashes[ref.Hash] = true
				}
			}
		}
	}

	// GC blobs (remove those not referenced by any live manifest)
	blobDir := filepath.Join(s.root, "blobs")
	filepath.Walk(blobDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		hash := filepath.Base(path)
		if !liveHashes[hash] {
			os.Remove(path)
			freed += info.Size()
			removed++
		}
		return nil
	})

	return freed, removed, nil
}
