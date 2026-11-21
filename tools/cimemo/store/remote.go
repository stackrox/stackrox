package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/davdhacs/cimemo/gocache"
)

// Remote tier for the cimemo blob store: manifests and output blobs in
// Azure, namespaced under store/ (the gocache proxy uses a/, d/, m/ —
// no collision). Level-1 receipts become cross-machine: a fresh runner
// computes the input key, fetches the manifest, restores the outputs —
// the build tool never runs.
//
// Integrity: blob keys ARE content hashes; every remote-sourced blob
// is re-verified on arrival (the CA-work lesson — a corrupt or
// poisoned blob must degrade to a miss, never serve wrong bytes).

const remoteManifestPrefix = "store/m/"
const remoteBlobPrefix = "store/b/"

// SetRemote attaches a cloud backend to the store.
func (s *Store) SetRemote(r *gocache.Remote) {
	s.remote = r
}

func (s *Store) hasRemote() bool { return s.remote != nil }

// remoteUploadManifest pushes the manifest JSON and every referenced
// blob (content-addressed, If-None-Match dedup).
func (s *Store) remoteUploadManifest(m *Manifest) error {
	for _, ref := range m.Blobs {
		data, err := s.GetBlob(ref.Hash)
		if err != nil {
			continue // blob pruned locally; manifest still references it
		}
		if _, err := s.remote.PutIfNotExistPub(remoteBlobPrefix+ref.Hash, data); err != nil {
			return fmt.Errorf("blob upload %s: %w", ref.Hash[:12], err)
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.remote.PutPub(remoteManifestPrefix+m.Key, data)
}

// remoteFetchManifest retrieves a manifest by key, or nil when absent.
func (s *Store) remoteFetchManifest(key string) (*Manifest, error) {
	data, err := s.remote.Get(remoteManifestPrefix + key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("remote manifest parse %s: %w", key, err)
	}
	return &m, nil
}

// remoteFetchBlob pulls a blob into the local store, verifying its
// content hash (the key) before accepting it.
func (s *Store) remoteFetchBlob(hash string) error {
	// Check the file directly — calling GetBlob here would recurse
	// (GetBlob calls remoteFetchBlob on local miss).
	if _, err := os.Stat(s.blobPath(hash)); err == nil {
		return nil // already local
	}
	data, err := s.remote.Get(remoteBlobPrefix + hash)
	if err != nil || data == nil {
		return fmt.Errorf("blob fetch %s: %v", hash[:12], err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != hash {
		return fmt.Errorf("blob INTEGRITY %s: content hash mismatch — rejecting", hash[:12])
	}
	p := s.blobPath(hash)
	if err := writeAtomic(p, data); err != nil {
		return err
	}
	s.markBlob(hash)
	return nil
}
