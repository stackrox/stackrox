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

// Package moss stands for "memory-oriented sorted segments", and
// provides a data structure that manages an ordered Collection of
// key-val entries, with optional persistence.
//
// The design is similar to a simplified LSM tree (log structured
// merge tree), but is more like a "LSM array", in that a stack of
// immutable, sorted key-val arrays or "segments" is maintained.  When
// there's an incoming Batch of key-val mutations (see:
// ExecuteBatch()), the Batch, which is an array of key-val mutations,
// is sorted in-place and becomes an immutable "segment".  Then, the
// segment is atomically pushed onto a stack of segment pointers.  A
// higher segment in the stack will shadow mutations of the same key
// from lower segments.
//
// Separately, an asynchronous goroutine (the "merger") will
// continuously merge N sorted segments to keep stack height low.
//
// In the best case, a remaining, single, large sorted segment will be
// efficient in memory usage and efficient for binary search and range
// iteration.
//
// Iterations when the stack height is > 1 are implementing using a
// N-way heap merge.
//
// A Batch and a segment is actually two arrays: a byte array of
// contiguous key-val entries; and an uint64 array of entry offsets
// and key-val lengths that refer to the previous key-val entries byte
// array.
//
// In this design, stacks are treated as immutable via a copy-on-write
// approach whenever a stack is "modified".  So, readers and writers
// essentially don't block each other, and taking a Snapshot is also a
// relatively simple operation of atomically cloning the stack of
// segment pointers.
//
// Of note: mutations are only supported through Batch operations,
// which acknowledges the common practice of using batching to achieve
// higher write performance and embraces it.  Additionally, higher
// performance can be attained by using the batch memory
// pre-allocation parameters and the Batch.Alloc() API, allowing
// applications to serialize keys and vals directly into memory
// maintained by a batch, which can avoid extra memory copying.
//
// IMPORTANT: The keys in a Batch must be unique.  That is,
// myBatch.Set("x", "foo"); myBatch.Set("x", "bar") is not supported.
// Applications that do not naturally meet this requirement might
// maintain their own map[key]val data structures to ensure this
// uniqueness constraint.
//
// An optional, asynchronous persistence goroutine (the "persister")
// can drain mutations to a lower level, ordered key-value storage
// layer.  An optional, built-in storage layer ("mossStore") is
// available, that will asynchronously write segments to the end of a
// file (append only design), with reads performed using mmap(), and
// with user controllable compaction configuration.  See:
// OpenStoreCollection().
//
// NOTE: the mossStore persistence design does not currently support
// moving files created on one machine endian'ness type to another
// machine with a different endian'ness type.
//
package moss

import (
	"errors"
	"sync"
	"time"

	"github.com/couchbase/ghistogram"
)

// ErrAllocTooLarge is returned when the requested allocation cannot
// be satisfied by the pre-allocated buffer.
var ErrAllocTooLarge = errors.New("alloc-too-large")

// ErrAlreadyInitialized is returned when initialization was
// attempted on an already initialized object.
var ErrAlreadyInitialized = errors.New("already-initialized")

// ErrCanceled is used when an operation has been canceled.
var ErrCanceled = errors.New("canceled")

// ErrClosed is returned when the collection is already closed.
var ErrClosed = errors.New("closed")

// ErrNoSuchCollection is returned when attempting to access or delete
// an unknown child collection.
var ErrNoSuchCollection = errors.New("no-such-collection")

// ErrBadCollectionName is returned when the child collection
// name is invalid, for example "".
var ErrBadCollectionName = errors.New("bad-collection-name")

// ErrIteratorDone is returned when the iterator has reached the end
// range of the iterator or the end of the collection.
var ErrIteratorDone = errors.New("iterator-done")

// ErrMaxTries is returned when a max number of tries or attempts for
// some operation has been reached.
var ErrMaxTries = errors.New("max-tries")

// ErrMergeOperatorNil is returned if a merge operation is performed
// without specifying a MergeOperator in the CollectionOptions.
var ErrMergeOperatorNil = errors.New("merge-operator-nil")

// ErrMergeOperatorFullMergeFailed is returned when the provided
// MergeOperator fails during the FullMerge operations.
var ErrMergeOperatorFullMergeFailed = errors.New("merge-operator-full-merge-failed")

// ErrUnexpected is returned on an unexpected situation.
var ErrUnexpected = errors.New("unexpected")

// ErrUnimplemented is returned when an unimplemented feature has been
// used.
var ErrUnimplemented = errors.New("unimplemented")

// ErrKeyTooLarge is returned when the length of the key exceeds the limit of 2^24.
var ErrKeyTooLarge = errors.New("key-too-large")

// ErrValueTooLarge is returned when the length of the value exceeds the limit of 2^28.
var ErrValueTooLarge = errors.New("value-too-large")

// ErrAborted is returned when any operations are aborted.
var ErrAborted = errors.New("operation-aborted")

// ErrSegmentCorrupted is returned upon any segment corruptions.
var ErrSegmentCorrupted = errors.New("segment-corrupted")

// A Collection represents an ordered mapping of key-val entries,
// where a Collection is snapshot'able and atomically updatable.
type Collection interface {
	// Start kicks off required background tasks.
	Start() error

	// Close synchronously stops background tasks and releases
	// resources.
	Close() error

	// Options returns the options currently being used.
	Options() CollectionOptions

	// Snapshot returns a stable Snapshot of the key-value entries.
	Snapshot() (Snapshot, error)

	// Get retrieves a value from the collection for a given key
	// and returns nil if the key is not found.
	Get(key []byte, readOptions ReadOptions) ([]byte, error)

	// NewBatch returns a new Batch instance with preallocated
	// resources.  See the Batch.Alloc() method.
	NewBatch(totalOps, totalKeyValBytes int) (Batch, error)

	// ExecuteBatch atomically incorporates the provided Batch into
	// the Collection.  The Batch instance should be Close()'ed and
	// not reused after ExecuteBatch() returns.
	ExecuteBatch(b Batch, writeOptions WriteOptions) error

	// Stats returns stats for this collection.  Note that stats might
	// be updated asynchronously.
	Stats() (*CollectionStats, error)

	// Histograms returns a snapshot of the histograms for this
	// collection.  Note that histograms might be updated
	// asynchronously.
	Histograms() ghistogram.Histograms
}

// CollectionOptions allows applications to specify config settings.
type CollectionOptions struct {
	// MergeOperator is an optional func provided by an application
	// that wants to use Batch.Merge()'ing.
	MergeOperator MergeOperator `json:"-"`

	// DeferredSort allows ExecuteBatch() to operate more quickly by
	// deferring the sorting of an incoming batch until it is needed
	// by a reader.  The tradeoff is that later read operations can
	// take longer as the sorting is finally performed.
	DeferredSort bool

	// MinMergePercentage allows the merger to avoid premature merging
	// of segments that are too small, where a segment X has to reach
	// a certain size percentage compared to the next lower segment
	// before segment X (and all segments above X) will be merged.
	MinMergePercentage float64

	// MaxPreMergerBatches is the max number of batches that can be
	// accepted into the collection through ExecuteBatch() and held
	// for merging but that have not been actually processed by the
	// merger yet.  When the number of held but unprocessed batches
	// reaches MaxPreMergerBatches, then ExecuteBatch() will block to
	// allow the merger to catch up.
	MaxPreMergerBatches int

	// MergerCancelCheckEvery is the number of ops the merger will
	// perform before it checks to see if a merger operation was
	// canceled.
	MergerCancelCheckEvery int

	// MergerIdleRunTimeoutMS is the idle time in milliseconds after which the
	// background merger will perform an "idle run" which can trigger
	// incremental compactions to speed up queries.
	MergerIdleRunTimeoutMS int64

	// MaxDirtyOps, when greater than zero, is the max number of dirty
	// (unpersisted) ops allowed before ExecuteBatch() blocks to allow
	// the persister to catch up.  It only has effect with a non-nil
	// LowerLevelUpdate.
	MaxDirtyOps uint64

	// MaxDirtyKeyValBytes, when greater than zero, is the max number
	// of dirty (unpersisted) key-val bytes allowed before
	// ExecuteBatch() blocks to allow the persister to catch up.  It
	// only has effect with a non-nil LowerLevelUpdate.
	MaxDirtyKeyValBytes uint64

	// CachePersisted allows the collection to cache clean, persisted
	// key-val's, and is considered when LowerLevelUpdate is used.
	CachePersisted bool

	// LowerLevelInit is an optional Snapshot implementation that
	// initializes the lower-level storage of a Collection.  This
	// might be used, for example, for having a Collection be a
	// write-back cache in front of a persistent implementation.
	LowerLevelInit Snapshot `json:"-"`

	// LowerLevelUpdate is an optional func that is invoked when the
	// lower-level storage should be updated.
	LowerLevelUpdate LowerLevelUpdate `json:"-"`

	Debug int // Higher means more logging, when Log != nil.

	// Log is a callback invoked when the Collection needs to log a
	// debug message.  Optional, may be nil.
	Log func(format string, a ...interface{}) `json:"-"`

	// OnError is an optional callback invoked when the Collection
	// encounters an error.  This might happen when the background
	// goroutines of moss encounter errors, such as during segment
	// merging or optional persistence operations.
	OnError func(error) `json:"-"`

	// OnEvent is an optional callback invoked on Collection related
	// processing events.  If the application's callback
	// implementation blocks, it may pause processing and progress,
	// depending on the type of callback event kind.
	OnEvent func(event Event) `json:"-"`

	// ReadOnly means that persisted data and storage files if any,
	// will remain unchanged.
	ReadOnly bool
}

// Event represents the information provided in an OnEvent() callback.
type Event struct {
	Kind       EventKind
	Collection Collection
	Duration   time.Duration
}

// EventKind represents an event code for OnEvent() callbacks.
type EventKind int

// EventKindCloseStart is fired when a collection.Close() has begun.
// The closing might take awhile to complete and an EventKindClose
// will follow later.
var EventKindCloseStart = EventKind(1)

// EventKindClose is fired when a collection has been fully closed.
var EventKindClose = EventKind(2)

// EventKindMergerProgress is fired when the merger has completed a
// round of merge processing.
var EventKindMergerProgress = EventKind(3)

// EventKindPersisterProgress is fired when the persister has
// completed a round of persistence processing.
var EventKindPersisterProgress = EventKind(4)

// EventKindBatchExecuteStart is fired when a collection is starting
// to execute a batch.
var EventKindBatchExecuteStart = EventKind(5)

// EventKindBatchExecute is fired when a collection has finished
// executing a batch.
var EventKindBatchExecute = EventKind(6)

// DefaultCollectionOptions are the default configuration options.
var DefaultCollectionOptions = CollectionOptions{
	MergeOperator:          nil,
	MinMergePercentage:     0.8,
	MaxPreMergerBatches:    10,
	MergerCancelCheckEvery: 10000,
	MergerIdleRunTimeoutMS: 0,
	Debug:                  0,
	Log:                    nil,
}

// BatchOptions are provided to NewChildCollectionBatch().
type BatchOptions struct {
	TotalOps         int
	TotalKeyValBytes int
}

// A Batch is a set of mutations that will be incorporated atomically
// into a Collection.  NOTE: the keys in a Batch must be unique.
//
// Concurrent Batch's are allowed, but to avoid races, concurrent
// Batches should only be used by concurrent goroutines that can
// ensure the mutation keys are partitioned or non-overlapping between
// Batch instances.
type Batch interface {
	// Close must be invoked to release resources.
	Close() error

	// Set creates or updates an key-val entry in the Collection.  The
	// key must be unique (not repeated) within the Batch.  Set()
	// copies the key and val bytes into the Batch, so the memory
	// bytes of the key and val may be reused by the caller.
	Set(key, val []byte) error

	// Del deletes a key-val entry from the Collection.  The key must
	// be unique (not repeated) within the Batch.  Del copies the key
	// bytes into the Batch, so the memory bytes of the key may be
	// reused by the caller.  Del() on a non-existent key results in a
	// nil error.
	Del(key []byte) error

	// Merge creates or updates a key-val entry in the Collection via
	// the MergeOperator defined in the CollectionOptions.  The key
	// must be unique (not repeated) within the Batch.  Merge() copies
	// the key and val bytes into the Batch, so the memory bytes of
	// the key and val may be reused by the caller.
	Merge(key, val []byte) error

	// ----------------------------------------------------

	// Alloc provides a slice of bytes "owned" by the Batch, to reduce
	// extra copying of memory.  See the Collection.NewBatch() method.
	Alloc(numBytes int) ([]byte, error)

	// AllocSet is like Set(), but the caller must provide []byte
	// parameters that came from Alloc().
	AllocSet(keyFromAlloc, valFromAlloc []byte) error

	// AllocDel is like Del(), but the caller must provide []byte
	// parameters that came from Alloc().
	AllocDel(keyFromAlloc []byte) error

	// AllocMerge is like Merge(), but the caller must provide []byte
	// parameters that came from Alloc().
	AllocMerge(keyFromAlloc, valFromAlloc []byte) error

	// NewChildCollectionBatch returns a new Batch instance with preallocated
	// resources for a specific child collection given its unique name.
	// The child Batch will be executed atomically along with any
	// other child batches and with the top-level Batch
	// when the top-level Batch is executed.
	// The child collection name should not start with a '.' (period)
	// as those are reserved for future moss usage.
	NewChildCollectionBatch(collectionName string, options BatchOptions) (Batch, error)

	// DelChildCollection records a child collection deletion given the name.
	// It only takes effect when the top-level batch is executed.
	DelChildCollection(collectionName string) error
}

// A Snapshot is a stable view of a Collection for readers, isolated
// from concurrent mutation activity.
type Snapshot interface {
	// Close must be invoked to release resources.
	Close() error

	// Get retrieves a val from the Snapshot, and will return nil val
	// if the entry does not exist in the Snapshot.
	Get(key []byte, readOptions ReadOptions) ([]byte, error)

	// StartIterator returns a new Iterator instance on this Snapshot.
	//
	// On success, the returned Iterator will be positioned so that
	// Iterator.Current() will either provide the first entry in the
	// range or ErrIteratorDone.
	//
	// A startKeyInclusive of nil means the logical "bottom-most"
	// possible key and an endKeyExclusive of nil means the logical
	// key that's above the "top-most" possible key.
	StartIterator(startKeyInclusive, endKeyExclusive []byte,
		iteratorOptions IteratorOptions) (Iterator, error)

	// ChildCollectionNames returns an array of child collection name strings.
	ChildCollectionNames() ([]string, error)

	// ChildCollectionSnapshot returns a Snapshot on a given child
	// collection by its name.
	ChildCollectionSnapshot(childCollectionName string) (Snapshot, error)
}

// An Iterator allows enumeration of key-val entries.
type Iterator interface {
	// Close must be invoked to release resources.
	Close() error

	// Next moves the Iterator to the next key-val entry and will
	// return ErrIteratorDone if the Iterator is done.
	Next() error

	// SeekTo moves the Iterator to the lowest key-val entry whose key
	// is >= the given seekToKey, and will return ErrIteratorDone if
	// the Iterator is done.  SeekTo() will respect the
	// startKeyInclusive/endKeyExclusive bounds, if any, that were
	// specified with StartIterator().  Seeking to before the
	// startKeyInclusive will end up on the first key.  Seeking to or
	// after the endKeyExclusive will result in ErrIteratorDone.
	SeekTo(seekToKey []byte) error

	// Current returns ErrIteratorDone if the iterator is done.
	// Otherwise, Current() returns the current key and val, which
	// should be treated as immutable or read-only.  The key and val
	// bytes will remain available until the next call to Next() or
	// Close().
	Current() (key, val []byte, err error)

	// CurrentEx is a more advanced form of Current() that returns
	// more metadata for each entry.  It is more useful when used with
	// IteratorOptions.IncludeDeletions of true.  It returns
	// ErrIteratorDone if the iterator is done.  Otherwise, the
	// current EntryEx, key, val are returned, which should be treated
	// as immutable or read-only.
	CurrentEx() (entryEx EntryEx, key, val []byte, err error)
}

// WriteOptions are provided to Collection.ExecuteBatch().
type WriteOptions struct {
}

// ReadOptions are provided to Snapshot.Get().
type ReadOptions struct {
	// By default, the value returned during lookups or Get()'s are
	// copied.  Specifying true for NoCopyValue means don't copy the
	// value bytes, where the caller should copy the value themselves
	// if they need the value after the lifetime of the enclosing
	// snapshot.  When true, the caller must treat the value returned
	// by a lookup/Get() as immutable.
	NoCopyValue bool

	// SkipLowerLevel is an advanced flag that specifies that a
	// point lookup should fail on a cache-miss and not attempt to access
	// key-val entries from the optional, chained,
	// lower-level snapshot (disk based). See
	// CollectionOptions.LowerLevelInit/LowerLevelUpdate.
	SkipLowerLevel bool
}

// IteratorOptions are provided to StartIterator().
type IteratorOptions struct {
	// IncludeDeletions is an advanced flag that specifies that an
	// Iterator should include deletion operations in its enuemration.
	// See also the Iterator.CurrentEx() method.
	IncludeDeletions bool

	// SkipLowerLevel is an advanced flag that specifies that an
	// Iterator should not enumerate key-val entries from the
	// optional, chained, lower-level iterator.  See
	// CollectionOptions.LowerLevelInit/LowerLevelUpdate.
	SkipLowerLevel bool

	// MinSegmentLevel is an advanced parameter that specifies that an
	// Iterator should skip segments at a level less than
	// MinSegmentLevel.  MinSegmentLevel is 0-based level, like an
	// array index.
	MinSegmentLevel int

	// MaxSegmentHeight is an advanced parameter that specifies that
	// an Iterator should skip segments at a level >= than
	// MaxSegmentHeight.  MaxSegmentHeight is 1-based height, like an
	// array length.
	MaxSegmentHeight int

	// base is used internally to provide the iterator with a
	// segmentStack to use instead of a lower-level snapshot.  It's
	// used so that segment merging consults the stackDirtyBase.
	base *segmentStack
}

// EntryEx provides extra, advanced information about an entry from
// the Iterator.CurrentEx() method.
type EntryEx struct {
	// Operation is an OperationXxx const.
	Operation uint64
}

// OperationSet replaces the value associated with the key.
const OperationSet = uint64(0x0100000000000000)

// OperationDel removes the value associated with the key.
const OperationDel = uint64(0x0200000000000000)

// OperationMerge merges the new value with the existing value associated with
// the key, as described by the configured MergeOperator.
const OperationMerge = uint64(0x0300000000000000)

// A MergeOperator may be implemented by applications that wish to
// optimize their read-compute-write use cases.  Write-heavy counters,
// for example, could be implemented efficiently by using the
// MergeOperator functionality.
type MergeOperator interface {
	// Name returns an identifier for this merge operator, which might
	// be used for logging / debugging.
	Name() string

	// FullMerge the full sequence of operands on top of an
	// existingValue and returns the merged value.  The existingValue
	// may be nil if no value currently exists.  If full merge cannot
	// be done, return (nil, false).
	FullMerge(key, existingValue []byte, operands [][]byte) ([]byte, bool)

	// Partially merge two operands.  If partial merge cannot be done,
	// return (nil, false), which will defer processing until a later
	// FullMerge().
	PartialMerge(key, leftOperand, rightOperand []byte) ([]byte, bool)
}

// LowerLevelUpdate is the func callback signature used when a
// Collection wants to update its optional, lower-level storage.
type LowerLevelUpdate func(higher Snapshot) (lower Snapshot, err error)

// CollectionStats fields that are prefixed like CurXxxx are gauges
// (can go up and down), and fields that are prefixed like TotXxxx are
// monotonically increasing counters.
type CollectionStats struct {
	TotOnError uint64

	TotCloseBeg           uint64
	TotCloseMergerDone    uint64
	TotClosePersisterDone uint64
	TotCloseLowerLevelBeg uint64
	TotCloseLowerLevelEnd uint64
	TotCloseEnd           uint64

	TotSnapshotBeg           uint64
	TotSnapshotEnd           uint64
	TotSnapshotInternalBeg   uint64
	TotSnapshotInternalEnd   uint64
	TotSnapshotInternalClose uint64

	TotGet    uint64
	TotGetErr uint64

	TotNewBatch                 uint64
	TotNewBatchTotalOps         uint64
	TotNewBatchTotalKeyValBytes uint64

	TotExecuteBatchBeg            uint64
	TotExecuteBatchErr            uint64
	TotExecuteBatchEmpty          uint64
	TotExecuteBatchWaitBeg        uint64
	TotExecuteBatchWaitEnd        uint64
	TotExecuteBatchAwakeMergerBeg uint64
	TotExecuteBatchAwakeMergerEnd uint64
	TotExecuteBatchEnd            uint64

	TotNotifyMergerBeg uint64
	TotNotifyMergerEnd uint64

	TotMergerEnd                  uint64
	TotMergerLoop                 uint64
	TotMergerLoopRepeat           uint64
	TotMergerAll                  uint64
	TotMergerInternalBeg          uint64
	TotMergerInternalErr          uint64
	TotMergerInternalEnd          uint64
	TotMergerInternalSkip         uint64
	TotMergerLowerLevelNotify     uint64
	TotMergerLowerLevelNotifySkip uint64
	TotMergerEmptyDirtyMid        uint64

	TotMergerWaitIncomingBeg  uint64
	TotMergerWaitIncomingStop uint64
	TotMergerWaitIncomingEnd  uint64
	TotMergerWaitIncomingSkip uint64
	TotMergerIdleSleeps       uint64
	TotMergerIdleRuns         uint64

	TotMergerWaitOutgoingBeg  uint64
	TotMergerWaitOutgoingStop uint64
	TotMergerWaitOutgoingEnd  uint64
	TotMergerWaitOutgoingSkip uint64

	TotPersisterLoop       uint64
	TotPersisterLoopRepeat uint64
	TotPersisterWaitBeg    uint64
	TotPersisterWaitEnd    uint64
	TotPersisterEnd        uint64

	TotPersisterLowerLevelUpdateBeg uint64
	TotPersisterLowerLevelUpdateErr uint64
	TotPersisterLowerLevelUpdateEnd uint64

	CurDirtyOps      uint64
	CurDirtyBytes    uint64
	CurDirtySegments uint64

	CurDirtyTopOps      uint64
	CurDirtyTopBytes    uint64
	CurDirtyTopSegments uint64

	CurDirtyMidOps      uint64
	CurDirtyMidBytes    uint64
	CurDirtyMidSegments uint64

	CurDirtyBaseOps      uint64
	CurDirtyBaseBytes    uint64
	CurDirtyBaseSegments uint64

	CurCleanOps      uint64
	CurCleanBytes    uint64
	CurCleanSegments uint64
}

// ------------------------------------------------------------

// NewCollection returns a new, unstarted Collection instance.
func NewCollection(options CollectionOptions) (
	Collection, error) {
	histograms := make(ghistogram.Histograms)
	histograms["ExecuteBatchBytes"] =
		ghistogram.NewNamedHistogram("ExecuteBatchBytes", 10, 4, 4)
	histograms["ExecuteBatchOpsCount"] =
		ghistogram.NewNamedHistogram("ExecuteBatchOpsCount", 10, 4, 4)
	histograms["ExecuteBatchUsecs"] =
		ghistogram.NewNamedHistogram("ExecuteBatchUsecs", 10, 4, 4)
	histograms["MergerUsecs"] =
		ghistogram.NewNamedHistogram("MergerUsecs", 10, 4, 4)
	histograms["MutationKeyBytes"] =
		ghistogram.NewNamedHistogram("MutationKeyBytes", 10, 4, 4)
	histograms["MutationValBytes"] =
		ghistogram.NewNamedHistogram("MutationValBytes", 10, 4, 4)

	c := &collection{
		options:            &options,
		stopCh:             make(chan struct{}),
		pingMergerCh:       make(chan ping, 10),
		doneMergerCh:       make(chan struct{}),
		donePersisterCh:    make(chan struct{}),
		lowerLevelSnapshot: NewSnapshotWrapper(options.LowerLevelInit, nil),
		stats:              &CollectionStats{},
		histograms:         histograms,
	}

	c.stackDirtyTopCond = sync.NewCond(&c.m)
	c.stackDirtyBaseCond = sync.NewCond(&c.m)

	return c, nil
}
