//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"errors"
	"sync"

	"github.com/couchbase/ghistogram"
)

// ErrNoValidFooter is returned when a valid footer could not be found
// in a file.
var ErrNoValidFooter = errors.New("no-valid-footer")

// ErrNothingToCompact is an internal error returned when compact is
// called on a store that is already compacted.
var ErrNothingToCompact = errors.New("nothing-to-compact")

// --------------------------------------------------------

// Store represents data persisted in a directory.
type Store struct {
	dir     string
	options *StoreOptions

	m            sync.Mutex // Protects the fields that follow.
	refs         int
	footer       *Footer
	nextFNameSeq int64

	totPersists           uint64 // Total number of persists
	totCompactions        uint64 // Total number of compactions
	totCompactionsPartial uint64 // Total number of partial compactions into same file

	numLastCompactionBeforeBytes uint64 // File size before last compaction
	numLastCompactionAfterBytes  uint64 // File size after last compaction
	totCompactionDecreaseBytes   uint64 // File size decrease after all compactions
	totCompactionIncreaseBytes   uint64 // File size increase after all compactions
	maxCompactionDecreaseBytes   uint64 // Max file size decrease from any compaction
	maxCompactionIncreaseBytes   uint64 // Max file size increase from any compaction
	totCompactionBeforeBytes     uint64 // total bytes to be compacted
	totCompactionWrittenBytes    uint64 // total bytes written out by compaction

	histograms ghistogram.Histograms // Histograms from store operations
	fileRefMap map[string]*FileRef   // Map to contain the FileRefs
	abortCh    chan struct{}         // Forced close/abort channel
}

// StoreCloseExOptions represents store CloseEx options.
type StoreCloseExOptions struct {
	// Abort means stop as soon as possible, even if data might be lost,
	// such as any mutations not yet persisted.
	Abort bool
}

// StoreOptions are provided to OpenStore().
type StoreOptions struct {
	// CollectionOptions should be the same as used with
	// NewCollection().
	CollectionOptions CollectionOptions

	// CompactionPercentage determines when a compaction will run when
	// CompactionConcern is CompactionAllow.  When the percentage of
	// ops between the non-base level and the base level is greater
	// than CompactionPercentage, then compaction will be run.
	CompactionPercentage float64

	// CompactionLevelMaxSegments determines the number of segments
	// per level exceeding which partial or full compaction will run.
	CompactionLevelMaxSegments int

	// CompactionLevelMultiplier is the factor which determines the
	// next level in terms of segment sizes.
	CompactionLevelMultiplier int

	// CompactionBufferPages is the number of pages to use for
	// compaction, where writes are buffered before flushing to disk.
	CompactionBufferPages int

	// CompactionSync of true means perform a file sync at the end of
	// compaction for additional safety.
	CompactionSync bool

	// CompactionSyncAfterBytes controls the number of bytes after
	// which compaction is allowed to invoke an file sync, followed
	// by an additional file sync at the end of compaction. A value
	// that is < 0 annulls this behavior.
	CompactionSyncAfterBytes int

	// OpenFile allows apps to optionally provide their own file
	// opening implementation.  When nil, os.OpenFile() is used.
	OpenFile OpenFile `json:"-"`

	// Log is a callback invoked when store needs to log a debug
	// message.  Optional, may be nil.
	Log func(format string, a ...interface{}) `json:"-"`

	// KeepFiles means that unused, obsoleted files will not be
	// removed during OpenStore().  Keeping old files might be useful
	// when diagnosing file corruption cases.
	KeepFiles bool

	// Choose which Kind of segment to persist, if unspecified defaults
	// to the value of DefaultPersistKind.
	PersistKind string

	// SegmentKeysIndexMaxBytes is the maximum size in bytes allowed for
	// the segmentKeysIndex. Also, an index will not be built if the
	// segment's total key bytes is less than this parameter.
	SegmentKeysIndexMaxBytes int

	// SegmentKeysIndexMinKeyBytes is the minimum size in bytes that the
	// keys of a segment must reach before a segment key index is built.
	SegmentKeysIndexMinKeyBytes int
}

// DefaultPersistKind determines which persistence Kind to choose when
// none is specified in StoreOptions.
var DefaultPersistKind = SegmentKindBasic

// DefaultStoreOptions are the default store options when the
// application hasn't provided a meaningful configuration value.
// Advanced applications can use these to fine tune performance.
var DefaultStoreOptions = StoreOptions{
	CompactionPercentage:        0.65,
	CompactionLevelMaxSegments:  4,
	CompactionLevelMultiplier:   9,
	CompactionBufferPages:       512,
	CompactionSyncAfterBytes:    16000000,
	SegmentKeysIndexMaxBytes:    100000,
	SegmentKeysIndexMinKeyBytes: 10000000,
}

// StorePersistOptions are provided to Store.Persist().
type StorePersistOptions struct {
	// NoSync means do not perform a file sync at the end of
	// persistence (before returning from the Store.Persist() method).
	// Using NoSync of true might provide better performance, but at
	// the cost of data safety.
	NoSync bool

	// CompactionConcern controls whether compaction is allowed or
	// forced as part of persistence.
	CompactionConcern CompactionConcern
}

// CompactionConcern is a type representing various possible compaction
// behaviors associated with persistence.
type CompactionConcern int

// CompactionDisable means no compaction.
var CompactionDisable = CompactionConcern(0)

// CompactionAllow means compaction decision is automated and based on
// the configed policy and parameters, such as CompactionPercentage.
var CompactionAllow = CompactionConcern(1)

// CompactionForce means compaction should be performed immediately.
var CompactionForce = CompactionConcern(2)

// --------------------------------------------------------

// SegmentLoc represents a persisted segment.
type SegmentLoc struct {
	Kind string // Used as the key for SegmentLoaders.

	KvsOffset uint64 // Byte offset within the file.
	KvsBytes  uint64 // Number of bytes for the persisted segment.kvs.

	BufOffset uint64 // Byte offset within the file.
	BufBytes  uint64 // Number of bytes for the persisted segment.buf.

	TotOpsSet  uint64
	TotOpsDel  uint64
	TotKeyByte uint64
	TotValByte uint64

	mref *mmapRef // Immutable and ephemeral / non-persisted.
}

// TotOps returns number of ops in a segment loc.
func (sloc *SegmentLoc) TotOps() int { return int(sloc.KvsBytes / 8 / 2) }

// --------------------------------------------------------

// SegmentLocs represents a slice of SegmentLoc
type SegmentLocs []SegmentLoc

// AddRef increases the ref count on each SegmentLoc in this SegmentLocs
func (slocs SegmentLocs) AddRef() {
	for _, sloc := range slocs {
		if sloc.mref != nil {
			sloc.mref.AddRef()
		}
	}
}

// DecRef decreases the ref count on each SegmentLoc in this SegmentLocs
func (slocs SegmentLocs) DecRef() {
	for _, sloc := range slocs {
		if sloc.mref != nil {
			sloc.mref.DecRef()
		}
	}
}

// Close allows the SegmentLocs to implement the io.Closer interface.
// It actually just performs what should be the final DecRef() call
// which takes the reference count to 0.
func (slocs SegmentLocs) Close() error {
	slocs.DecRef()
	return nil
}

// --------------------------------------------------------

// A SegmentLoaderFunc is able to load a segment from a SegmentLoc.
type SegmentLoaderFunc func(
	sloc *SegmentLoc) (Segment, error)

// SegmentLoaders is a registry of available segment loaders, which
// should be immutable after process init()'ialization.  It is keyed
// by SegmentLoc.Kind.
var SegmentLoaders = map[string]SegmentLoaderFunc{}

// A SegmentPersisterFunc is able to persist a segment to a file,
// and return a SegmentLoc describing it.
type SegmentPersisterFunc func(
	s Segment, f File, pos int64, options *StoreOptions) (SegmentLoc, error)

// SegmentPersisters is a registry of available segment persisters,
// which should be immutable after process init()'ialization.  It is
// keyed by SegmentLoc.Kind.
var SegmentPersisters = map[string]SegmentPersisterFunc{}

// --------------------------------------------------------

// OpenStore returns a store instance for a directory.  An empty
// directory results in an empty store.
func OpenStore(dir string, options StoreOptions) (*Store, error) {
	return openStore(dir, options)
}

// Dir returns the directory for this store
func (s *Store) Dir() string {
	return s.dir
}

// Options a copy of this Store's StoreOptions
func (s *Store) Options() StoreOptions {
	return *s.options // Copy.
}

// Snapshot creates a Snapshot to access this Store
func (s *Store) Snapshot() (Snapshot, error) {
	return s.snapshot()
}

func (s *Store) snapshot() (*Footer, error) {
	s.m.Lock()
	footer := s.footer
	if footer != nil {
		footer.AddRef()
	}
	s.m.Unlock()
	return footer, nil
}

// AddRef increases the ref count on this store
func (s *Store) AddRef() {
	s.m.Lock()
	s.refs++
	s.m.Unlock()
}

// Close decreases the ref count on this store, and if the count is 0
// proceeds to actually close the store.
func (s *Store) Close() error {
	s.m.Lock()
	defer s.m.Unlock()

	s.refs--
	if s.refs > 0 || s.footer == nil {
		return nil
	}

	footer := s.footer
	s.footer = nil

	return footer.Close()
}

// CloseEx provides more advanced closing options.
func (s *Store) CloseEx(options StoreCloseExOptions) error {
	if options.Abort {
		close(s.abortCh)
	}
	return s.Close()
}

// IsAborted returns whether the store operations are aborted.
func (s *Store) IsAborted() bool {
	select {
	case <-s.abortCh:
		return true
	default:
		return false
	}
}

// --------------------------------------------------------

// Persist helps the store implement the lower-level-update func, and
// normally is not called directly by applications.  The higher
// snapshot may be nil.  Advanced users who wish to call Persist()
// directly MUST invoke it in single threaded manner only.
func (s *Store) Persist(higher Snapshot, persistOptions StorePersistOptions) (
	Snapshot, error) {
	return s.persist(higher, persistOptions)
}

// --------------------------------------------------------

// OpenStoreCollection returns collection based on a persisted store
// in a directory.  Updates to the collection will be persisted.  An
// empty directory starts an empty collection.  Both the store and
// collection should be closed by the caller when done.
func OpenStoreCollection(dir string, options StoreOptions,
	persistOptions StorePersistOptions) (*Store, Collection, error) {
	store, err := OpenStore(dir, options)
	if err != nil {
		return nil, nil, err
	}

	coll, err := store.OpenCollection(options, persistOptions)
	if err != nil {
		store.Close()
		return nil, nil, err
	}

	return store, coll, nil
}

// --------------------------------------------------------

// OpenCollection opens a collection based on a store.  Applications
// should open at most a single collection per store for performing
// read/write work.
func (s *Store) OpenCollection(options StoreOptions,
	persistOptions StorePersistOptions) (Collection, error) {
	return s.openCollection(options, persistOptions)
}

// --------------------------------------------------------

// SnapshotPrevious returns the next older, previous snapshot based on
// a given snapshot, allowing the application to walk backwards into
// the history of a store at previous points in time.  The given
// snapshot must come from the same store.  A nil returned snapshot
// means no previous snapshot is available.  Of note, store
// compactions will trim previous history from a store.
func (s *Store) SnapshotPrevious(ss Snapshot) (Snapshot, error) {
	return s.snapshotPrevious(ss)
}

// --------------------------------------------------------

// SnapshotRevert atomically and durably brings the store back to the
// point-in-time as represented by the revertTo snapshot.
// SnapshotRevert() should only be passed a snapshot that came from
// the same store, such as from using Store.Snapshot() or
// Store.SnapshotPrevious().
//
// SnapshotRevert() must not be invoked concurrently with
// Store.Persist(), so it is recommended that SnapshotRevert() should
// be invoked only after the collection has been Close()'ed, which
// helps ensure that you are not racing with concurrent, background
// persistence goroutines.
//
// SnapshotRevert() can fail if the given snapshot is too old,
// especially w.r.t. compactions.  For example, navigate back to an
// older snapshot X via SnapshotPrevious().  Then, do a full
// compaction.  Then, SnapshotRevert(X) will give an error.
func (s *Store) SnapshotRevert(revertTo Snapshot) error {
	return s.snapshotRevert(revertTo)
}
