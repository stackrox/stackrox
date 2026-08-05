package vsockserver

import (
	"bytes"
	"errors"
	"os"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/filedownloader"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ MappingProvider = (*SensorUpdater)(nil)
	_ MappingUpdater  = (*SensorUpdater)(nil)
	_ ScanBusyGate    = (*SensorUpdater)(nil)
)

var errSensorMappingNotReady = errors.New("no repo-to-CPE mapping available yet")

// SensorUpdater is the MappingProvider/MappingUpdater/ScanBusyGate for an
// agent whose mapping is pushed in over VSOCK via SyncRepoCPEMapping. It
// defers applying a new mapping into active while busy is set, so an
// in-flight VM-disk scan (and the GetReport response for it) never
// observes the active mapping change mid-flight.
type SensorUpdater struct {
	active     []byte
	activeHash string
	cachePath  string
	pending    []byte
	busy       bool
	onChange   func()
	mu         sync.Mutex
}

// NewSensorUpdater seeds active from cachePath if it holds a validated
// mapping, else from bundledPath if non-empty and validated, else stays
// empty until the first Update. onChange must already be non-nil by the
// time any bootstrap content is found, so construct dependents (e.g. the
// rescanner) before calling this.
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
		return false
	}
	if err := repositorytocpe.ValidateMapping(content); err != nil {
		log.Warnf("Ignoring invalid repo-to-CPE mapping at %q: %v", path, err)
		return false
	}
	u.active = content
	u.activeHash = repositorytocpe.HashMapping(content)
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
	u.mu.Lock()
	active := u.active
	cachePath := u.cachePath
	u.mu.Unlock()
	if len(active) == 0 {
		return "", errSensorMappingNotReady
	}
	if err := filedownloader.AtomicWriteFile(cachePath, active); err != nil {
		return "", err
	}
	return cachePath, nil
}

// Update validates content and either applies it to active immediately or,
// if a scan is in flight (busy), stages it as pending for
// MarkScanIdleAndApplyPending to promote later. updated is false only when
// content is invalid or already equal to the current active/pending
// content; a validation error never touches state.
func (u *SensorUpdater) Update(content []byte) (updated bool, err error) {
	if err := repositorytocpe.ValidateMapping(content); err != nil {
		return false, err
	}
	hash := repositorytocpe.HashMapping(content)

	u.mu.Lock()
	if hash == u.activeHash || bytes.Equal(content, u.pending) {
		u.mu.Unlock()
		return false, nil
	}
	if u.busy {
		u.pending = content
		u.mu.Unlock()
		log.Infof("SyncRepoCPEMapping: deferring apply of mapping (hash=%s) until the in-flight scan finishes", hash)
		return true, nil
	}
	u.applyLocked(content, hash)
	u.mu.Unlock()
	u.persistAndNotify(content)
	return true, nil
}

// MarkScanBusy marks a VM-disk scan in progress, so a concurrent Update
// stages new content as pending instead of mutating active underneath it.
func (u *SensorUpdater) MarkScanBusy() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.busy = true
}

// MarkScanIdleAndApplyPending clears busy and, if Update staged content
// while busy, promotes it to active now that the scan (and its GetReport
// response) has finished.
func (u *SensorUpdater) MarkScanIdleAndApplyPending() {
	u.mu.Lock()
	u.busy = false
	pending := u.pending
	if pending == nil {
		u.mu.Unlock()
		return
	}
	u.pending = nil
	u.applyLocked(pending, repositorytocpe.HashMapping(pending))
	u.mu.Unlock()
	u.persistAndNotify(pending)
}

// applyLocked sets active/activeHash to content/hash. Caller must hold mu.
func (u *SensorUpdater) applyLocked(content []byte, hash string) {
	u.active = content
	u.activeHash = hash
}

// persistAndNotify writes content to cachePath in the background - a scan
// or another Update must never block on disk I/O - then synchronously
// fires onChange so the caller's later reads already see the new active
// mapping.
func (u *SensorUpdater) persistAndNotify(content []byte) {
	cachePath := u.cachePath
	go func() {
		if err := filedownloader.AtomicWriteFile(cachePath, content); err != nil {
			log.Warnf("Persisting repo-to-CPE mapping cache to %q: %v", cachePath, err)
		}
	}()
	if u.onChange != nil {
		u.onChange()
	}
}
