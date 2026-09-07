package vsockserver

import (
	"bytes"
	"errors"
	"os"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
)

var (
	_ MappingProvider = (*SensorUpdater)(nil)
	_ MappingUpdater  = (*SensorUpdater)(nil)
	_ ScanBusyGate    = (*SensorUpdater)(nil)
)

var errSensorMappingNotReady = errors.New("no repo-to-CPE mapping available yet")

// SensorUpdater applies repo-to-CPE mappings pushed over VSOCK. A Sync
// during a scan is deferred so it cannot replace the file that scan is
// reading.
type SensorUpdater struct {
	active     []byte
	activeHash string
	cachePath  string
	pending    []byte
	busy       bool
	onChange   func()
	mu         sync.Mutex
	// persistMu serializes cachePath writes. persistActive is the only
	// acquirer; it takes persistMu, then mu, never the reverse.
	persistMu sync.Mutex
	// persistWG tracks persistAndNotify's background writes so tests can
	// drain them before TempDir cleanup.
	persistWG sync.WaitGroup
}

// NewSensorUpdater seeds active from cachePath, else bundledPath, else
// stays empty until the first Update. Construct onChange's dependents
// (e.g. the rescanner) first, since it may fire during this call.
func NewSensorUpdater(cachePath, bundledPath string, onChange func()) *SensorUpdater {
	u := &SensorUpdater{cachePath: cachePath, onChange: onChange}
	if u.bootstrapFrom(cachePath) || u.bootstrapFrom(bundledPath) {
		if onChange != nil {
			onChange()
		}
	}
	return u
}

// bootstrapFrom seeds active/activeHash from path if it decodes as a valid
// mapping. A no-op for an empty path, so the optional bundled seed can be
// skipped by passing "". Reports whether it seeded active.
func (u *SensorUpdater) bootstrapFrom(path string) bool {
	if path == "" {
		return false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		// First boot always misses cache; do not treat that as an error.
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("Reading repo-to-CPE mapping at %q: %v", path, err)
		}
		return false
	}
	if err := cpemapping.ValidateMapping(content); err != nil {
		log.Warnf("Ignoring invalid repo-to-CPE mapping at %q: %v", path, err)
		return false
	}
	u.active = content
	u.activeHash = cpemapping.HashMapping(content)
	return true
}

// Ready reports whether a validated mapping is currently active.
func (u *SensorUpdater) Ready() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.active) > 0
}

// Hash returns the active mapping's content hash, or "" if not Ready.
func (u *SensorUpdater) Hash() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.activeHash
}

// UpdatePath reports this updater's mapping source for ResponseMeta.
func (u *SensorUpdater) UpdatePath() pb.RepoCPEMappingUpdatePath {
	return pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR
}

// Bytes returns a copy of the active mapping content.
func (u *SensorUpdater) Bytes() ([]byte, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.active) == 0 {
		return nil, errSensorMappingNotReady
	}
	out := make([]byte, len(u.active))
	copy(out, u.active)
	return out, nil
}

// Path returns cachePath after (re-)writing it with the active mapping, so
// callers always get a file whose content matches Bytes() even if the
// fire-and-forget persistence from the last apply has not landed yet.
func (u *SensorUpdater) Path() (string, error) {
	return u.persistActive()
}

// Update applies content to active immediately, or stages it as pending
// for MarkScanIdleAndApplyPending to promote later if a scan is in
// flight (busy), so it never mutates active out from under that scan.
func (u *SensorUpdater) Update(content []byte) (updated bool, err error) {
	if err := cpemapping.ValidateMapping(content); err != nil {
		return false, err
	}
	// MappingUpdater retains content in active/pending and in the async
	// cache write; clone so a caller reusing the buffer cannot desync
	// those from activeHash.
	content = bytes.Clone(content)
	hash := cpemapping.HashMapping(content)

	updated, deferred := concurrency.WithLock2(&u.mu, func() (bool, bool) {
		if hash == u.activeHash {
			// The newest push matches active, so it supersedes whatever
			// an earlier, now-stale push staged as pending.
			u.pending = nil
			return false, false
		}
		if bytes.Equal(content, u.pending) {
			return false, false
		}
		if u.busy {
			u.pending = content
			return true, true
		}
		u.applyLocked(content, hash)
		return true, false
	})
	if !updated {
		log.Debugf("SyncRepoCPEMapping: mapping unchanged (hash=%s)", hash)
		return false, nil
	}
	if deferred {
		log.Infof("SyncRepoCPEMapping: deferring apply of mapping (hash=%s) until the in-flight scan finishes", hash)
		return true, nil
	}
	log.Infof("SyncRepoCPEMapping: applied mapping (hash=%s)", hash)
	u.persistAndNotify()
	return true, nil
}

// MarkScanBusy sets busy=true so a mapping-Sync during this scan is saved into pending instead of replacing the mapping.
// Replacing the mapping could cause race between the scan reading that mapping and the Sync writing it.
func (u *SensorUpdater) MarkScanBusy() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.busy = true
}

// MarkScanIdleAndApplyPending applies a mapping that arrived during the scan, now that the scan is no longer reading the file.
func (u *SensorUpdater) MarkScanIdleAndApplyPending() {
	pending := concurrency.WithLock1(&u.mu, func() []byte {
		u.busy = false
		pending := u.pending
		if pending == nil {
			return nil
		}
		u.pending = nil
		u.applyLocked(pending, cpemapping.HashMapping(pending))
		return pending
	})
	if pending != nil {
		log.Infof("Applying deferred mapping (hash=%s)", cpemapping.HashMapping(pending))
		u.persistAndNotify()
	}
}

// applyLocked sets active/activeHash to content/hash. Caller must hold mu.
func (u *SensorUpdater) applyLocked(content []byte, hash string) {
	u.active = content
	u.activeHash = hash
}

// persistAndNotify fires onChange now and persists cachePath in the
// background. Path is the write barrier if the file is needed before
// that goroutine finishes.
func (u *SensorUpdater) persistAndNotify() {
	u.persistWG.Go(func() {
		if _, err := u.persistActive(); err != nil {
			log.Warnf("Persisting repo-to-CPE mapping cache to %q: %v", u.cachePath, err)
		}
	})
	if u.onChange != nil {
		u.onChange()
	}
}

// WaitPersist blocks until every persistAndNotify goroutine has finished
// writing cachePath. Tests register it with t.Cleanup so TempDir removal
// cannot race AtomicWriteFile's sibling .tmp files.
func (u *SensorUpdater) WaitPersist() {
	u.persistWG.Wait()
}

// persistActive writes a copy of the current active mapping to cachePath.
// It takes persistMu, then mu, so a late persist cannot overwrite the
// file with a superseded mapping.
func (u *SensorUpdater) persistActive() (string, error) {
	return concurrency.WithLock2(&u.persistMu, func() (string, error) {
		active, cachePath := concurrency.WithLock2(&u.mu, func() ([]byte, string) {
			if len(u.active) == 0 {
				return nil, u.cachePath
			}
			return bytes.Clone(u.active), u.cachePath
		})
		if len(active) == 0 {
			return "", errSensorMappingNotReady
		}
		if err := filedownloader.AtomicWriteFile(cachePath, active); err != nil {
			return "", err
		}
		return cachePath, nil
	})
}
