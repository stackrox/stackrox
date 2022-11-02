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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/ghistogram"
)

// A collection implements the Collection interface.
type collection struct {
	options *CollectionOptions

	stopCh          chan struct{}
	pingMergerCh    chan ping
	doneMergerCh    chan struct{}
	donePersisterCh chan struct{}
	idleMergerTimer *time.Timer

	m sync.Mutex // Protects the fields that follow.

	// When ExecuteBatch() has pushed a new segment onto
	// stackDirtyTop, it can notify waiters like the merger via
	// waitDirtyIncomingCh (if non-nil).
	waitDirtyIncomingCh chan struct{}

	// When the persister has finished a persistence cycle, it can
	// notify waiters like the merger via waitDirtyOutgoingCh (if
	// non-nil).
	waitDirtyOutgoingCh chan struct{}

	// stats leverage sync/atomic counters.
	// All child collections simply point back to the same stats instance.
	// TODO: Have child collection specific stats.
	stats *CollectionStats

	// latestSnapshot caches the most recent collection snapshot to avoid
	// new snapshot creations in the absence of new mutations.
	latestSnapshot Snapshot

	// highestIncarNum is the highest descendant collection
	// incarnation number seen by this collection hierarchy.  It
	// monotonically increases every time a new child collection is
	// created and helps distinguish child collection recreations with
	// the same name.
	highestIncarNum uint64

	// ----------------------------------------

	// stackDirtyTopCond is used to wait for space in stackDirtyTop.
	stackDirtyTopCond *sync.Cond

	// stackDirtyBaseCond is used to wait for non-nil stackDirtyBase.
	stackDirtyBaseCond *sync.Cond

	// ----------------------------------------

	// ExecuteBatch() will push new segments onto stackDirtyTop if
	// there is space.
	stackDirtyTop *segmentStack

	// The merger goroutine asynchronously, atomically grabs all
	// segments from stackDirtyTop and atomically moves them into
	// stackDirtyMid.  The merger will also merge segments in
	// stackDirtyMid to keep its height low.
	stackDirtyMid *segmentStack

	// stackDirtyBase represents the segments currently being
	// persisted.  It is optionally populated by the merger when there
	// are merged segments ready for persistence.  Will be nil when
	// persistence is not being used.
	stackDirtyBase *segmentStack

	// stackClean represents the segments that have been optionally
	// persisted by the persister, and can now be safely evicted, as
	// the lowerLevelSnapshot will contain the entries from
	// stackClean.  Will be nil when persistence is not being used.
	stackClean *segmentStack

	// lowerLevelSnapshot provides an optional, lower-level storage
	// implementation, when using the Collection as a cache.
	lowerLevelSnapshot *SnapshotWrapper

	// histograms from collection operations
	histograms ghistogram.Histograms

	// incarNum is a unique incarnation number assigned to this child
	// collection at the time of its creation.  It helps process child
	// collection recreations and is zero in the top-level collection.
	incarNum uint64

	// Map of child collection by name.
	// TODO: Most of the fields of the child collections are nil, so
	// it might be lighter to use a dedicated struct instead of
	// reusing the collection struct.
	childCollections map[string]*collection
}

// ------------------------------------------------------

// Start kicks off required background gouroutines.
func (m *collection) Start() error {
	if !m.options.ReadOnly {
		// Kick off merger and persister only when not in Read-Only mode
		go m.runMerger()
		go m.runPersister()
	}
	return nil
}

// Close synchronously stops background goroutines.
func (m *collection) Close() error {
	m.fireEvent(EventKindCloseStart, 0)
	startTime := time.Now()
	defer func() {
		m.fireEvent(EventKindClose, time.Now().Sub(startTime))
	}()

	atomic.AddUint64(&m.stats.TotCloseBeg, 1)

	m.m.Lock()

	m.invalidateLatestSnapshotLOCKED()

	close(m.stopCh)

	m.stackDirtyTopCond.Broadcast()  // Awake all ExecuteBatch()'ers.
	m.stackDirtyBaseCond.Broadcast() // Awake persister.

	if !m.options.ReadOnly {
		m.m.Unlock()

		<-m.doneMergerCh
		atomic.AddUint64(&m.stats.TotCloseMergerDone, 1)

		<-m.donePersisterCh
		atomic.AddUint64(&m.stats.TotClosePersisterDone, 1)

		m.m.Lock()
	}

	if m.lowerLevelSnapshot != nil {
		atomic.AddUint64(&m.stats.TotCloseLowerLevelBeg, 1)
		m.lowerLevelSnapshot.Close()
		m.lowerLevelSnapshot = nil
		atomic.AddUint64(&m.stats.TotCloseLowerLevelEnd, 1)
	}

	stackDirtyTopPrev := m.stackDirtyTop
	m.stackDirtyTop = nil

	stackDirtyMidPrev := m.stackDirtyMid
	m.stackDirtyMid = nil

	stackDirtyBasePrev := m.stackDirtyBase
	m.stackDirtyBase = nil

	stackCleanPrev := m.stackClean
	m.stackClean = nil

	m.m.Unlock()

	stackDirtyTopPrev.Close()
	stackDirtyMidPrev.Close()
	stackDirtyBasePrev.Close()
	stackCleanPrev.Close()

	atomic.AddUint64(&m.stats.TotCloseEnd, 1)

	return nil
}

func (m *collection) isClosed() bool {
	select {
	case <-m.stopCh:
		return true
	default:
		return false
	}
}

// Options returns the current options.
func (m *collection) Options() CollectionOptions {
	return *m.options
}

// reuseSnapshot addRef()'s the underlying segmentStack.
func reuseSnapshot(snap Snapshot) Snapshot {
	if snap != nil {
		ss, ok := snap.(*segmentStack)
		if ok {
			ss.addRef()
			return snap
		}
	}

	return nil
}

// Snapshot returns a stable snapshot of the key-value entries.
func (m *collection) Snapshot() (rv Snapshot, err error) {
	m.m.Lock()
	rv = reuseSnapshot(m.latestSnapshot)
	if rv == nil { // No cached snapshot.
		rv, err = m.newSnapshotLOCKED()
		if err == nil {
			// collection holds 1 ref count for its cached snapshot copy.
			m.latestSnapshot = reuseSnapshot(rv)
		}
	}
	m.m.Unlock()

	return
}

// invalidateLatestSnapshotLOCKED is invoked whenever new mutations or
// internal modifications (merges/persistence) occur.
func (m *collection) invalidateLatestSnapshotLOCKED() {
	if m.latestSnapshot != nil {
		m.latestSnapshot.Close()
		m.latestSnapshot = nil
	}
}

// newSnapshotLOCKED creates a new stable snapshot of the key-value
// entries.
func (m *collection) newSnapshotLOCKED() (Snapshot, error) {
	if m.isClosed() {
		return nil, ErrClosed
	}

	atomic.AddUint64(&m.stats.TotSnapshotBeg, 1)

	rv, _, _, _, _ := m.snapshot(0, nil, true) // collection lock already held.

	atomic.AddUint64(&m.stats.TotSnapshotEnd, 1)

	return rv, nil
}

// Get retrieves a value by iterating over all the segments within
// the collection, if the key is not found a nil val is returned.
func (m *collection) Get(key []byte, readOptions ReadOptions) ([]byte, error) {
	if m.isClosed() {
		return nil, ErrClosed
	}

	atomic.AddUint64(&m.stats.TotGet, 1)

	val, err := m.get(key, readOptions)

	if err != nil {
		atomic.AddUint64(&m.stats.TotGetErr, 1)
	}

	return val, err
}

// NewBatch returns a new Batch instance with hinted amount of
// resources expected to be required.
func (m *collection) NewBatch(totalOps, totalKeyValBytes int) (
	Batch, error) {
	if m.isClosed() {
		return nil, ErrClosed
	}

	atomic.AddUint64(&m.stats.TotNewBatch, 1)
	atomic.AddUint64(&m.stats.TotNewBatchTotalOps, uint64(totalOps))
	atomic.AddUint64(&m.stats.TotNewBatchTotalKeyValBytes, uint64(totalKeyValBytes))

	return newBatch(m, BatchOptions{totalOps, totalKeyValBytes})
}

func (m *collection) ResetStackDirtyTop() error {
	m.m.Lock()
	stackDirtyTopPrev := m.stackDirtyTop
	m.stackDirtyTop = nil
	m.invalidateLatestSnapshotLOCKED()
	m.m.Unlock()
	stackDirtyTopPrev.Close()
	return nil
}

// ExecuteBatch atomically incorporates the provided Batch into the
// collection.  The Batch instance should not be reused after
// ExecuteBatch() returns.
func (m *collection) ExecuteBatch(bIn Batch,
	writeOptions WriteOptions) error {
	startTime := time.Now()

	defer func() {
		m.fireEvent(EventKindBatchExecute, time.Now().Sub(startTime))
	}()

	atomic.AddUint64(&m.stats.TotExecuteBatchBeg, 1)

	b, ok := bIn.(*batch)
	if !ok {
		atomic.AddUint64(&m.stats.TotExecuteBatchErr, 1)

		return fmt.Errorf("wrong Batch implementation type")
	}

	if b == nil || b.isEmpty() {
		atomic.AddUint64(&m.stats.TotExecuteBatchEmpty, 1)

		m.histograms["ExecuteBatchUsecs"].Add(
			uint64(time.Since(startTime).Nanoseconds()/1000), 1)

		return nil
	}

	maxPreMergerBatches := m.options.MaxPreMergerBatches
	if maxPreMergerBatches <= 0 {
		maxPreMergerBatches =
			DefaultCollectionOptions.MaxPreMergerBatches
	}

	if m.options.DeferredSort {
		b.readyDeferredSort() // Recursively ready child batches.
	} else {
		b.doSort() // Recursively sort the child batches.
	}

	// Notify handlers that we are about to execute a batch.
	m.fireEvent(EventKindBatchExecuteStart, 0)

	m.m.Lock()

	for m.stackDirtyTop != nil &&
		len(m.stackDirtyTop.a) >= maxPreMergerBatches {
		if m.isClosed() {
			m.m.Unlock()
			return ErrClosed
		}

		if m.options.DeferredSort {
			go b.RequestSort() // While waiting, might as well sort.
		}

		atomic.AddUint64(&m.stats.TotExecuteBatchWaitBeg, 1)
		m.stackDirtyTopCond.Wait()
		atomic.AddUint64(&m.stats.TotExecuteBatchWaitEnd, 1)
	}

	if m.isClosed() { // Could have been closed while waiting.
		m.m.Unlock()
		return ErrClosed
	}

	m.invalidateLatestSnapshotLOCKED()

	stackDirtyTop := m.buildStackDirtyTop(b, m.stackDirtyTop)

	prevStackDirtyTop := m.stackDirtyTop
	m.stackDirtyTop = stackDirtyTop

	waitDirtyIncomingCh := m.waitDirtyIncomingCh
	m.waitDirtyIncomingCh = nil

	m.m.Unlock()

	prevStackDirtyTop.Close()

	if waitDirtyIncomingCh != nil {
		atomic.AddUint64(&m.stats.TotExecuteBatchAwakeMergerBeg, 1)
		close(waitDirtyIncomingCh)
		atomic.AddUint64(&m.stats.TotExecuteBatchAwakeMergerEnd, 1)
	}

	atomic.AddUint64(&m.stats.TotExecuteBatchEnd, 1)

	m.histograms["ExecuteBatchUsecs"].Add(
		uint64(time.Since(startTime).Nanoseconds()/1000), 1)

	return nil
}

// buildStackDirtyTop recursively builds a segmentStack out of a
// recursive batch with potential child batches.
// This function does a 3 way merge.
// Consider the example below:
//    Incoming batch (b)    existing (curStackTop)   childCollections map
//   /       |      \           /     |     \           /     |     \
// child1  child2'  child4   child1 child2 child3    child1 child2 child3
// (del)  (update)  (new)
//
// The result is to build a new stackTop & update childCollection map as:
//    returned segmentStack (rv)     childCollections map
//       /         |      \             /     |     \
// child2+child2' child3  child4     child2 child3 child4
func (m *collection) buildStackDirtyTop(b *batch, curStackTop *segmentStack) (
	rv *segmentStack) {
	numDirtyTop := 0
	if curStackTop != nil {
		numDirtyTop = len(curStackTop.a)
	}

	rv = &segmentStack{options: m.options, refs: 1}
	rv.a = make([]Segment, 0, numDirtyTop+1)
	if curStackTop != nil {
		rv.a = append(rv.a, curStackTop.a...)
	}

	rv.incarNum = m.incarNum

	if b != nil {
		if b.Len() > 0 {
			rv.a = append(rv.a, b.segment)
		}

		for cName, cBatch := range b.childBatches {
			if cBatch == deletedChildBatchMarker { // child1 in diagram above.
				delete(m.childCollections, cName)
				continue
			}

			if len(m.childCollections) == 0 {
				m.childCollections = make(map[string]*collection)
			}
			childCollection, exists := m.childCollections[cName]
			if !exists { // Child collection being created for first time.
				m.highestIncarNum++
				childCollection = &collection{ // child4 in diagram above.
					options:         m.options,
					stats:           m.stats,
					highestIncarNum: m.highestIncarNum,
					incarNum:        m.highestIncarNum,
				}
				m.childCollections[cName] = childCollection
			}

			if len(rv.childSegStacks) == 0 {
				rv.childSegStacks = make(map[string]*segmentStack)
			}
			var prevChildSegStack *segmentStack
			if curStackTop != nil && len(curStackTop.childSegStacks) > 0 {
				// child2 from existing stackDirtyTop in diagram above.
				prevChildSegStack = curStackTop.childSegStacks[cName]
			}

			// Recursively merge & build the child collection batches.
			rv.childSegStacks[cName] = childCollection.buildStackDirtyTop(
				cBatch, prevChildSegStack)
		}
	}

	if curStackTop == nil {
		return rv
	}

	// There could be child collections in existing curStackTop that
	// were not in the batch, so copy over those recursively too.
	for cName, childStack := range curStackTop.childSegStacks {
		if len(rv.childSegStacks) == 0 {
			rv.childSegStacks = make(map[string]*segmentStack)
		}
		if rv.childSegStacks[cName] != nil {
			// This child collection was already processed as part of
			continue // batch.  Do not copy over to new stackDirtyTop.
		}

		// Else we have a child collection in existing stackDirtyTop
		// that was NOT in the incoming batch.
		childCollection, exists := m.childCollections[cName]
		if !exists || // This child collection was deleted OR
			// it was quickly recreated in the incoming batch.
			childCollection.incarNum != childStack.incarNum {
			continue // Do not copy over to new stackDirtyTop.
		}

		// Case of child3 from existing curStackTop in diagram above.
		rv.childSegStacks[cName] =
			childCollection.buildStackDirtyTop(nil, childStack)
	}

	return rv
}

// ------------------------------------------------------

// Update stats/histograms given an immutable segment.
func (m *collection) updateStats(a *segment) {
	if m == nil || m.isClosed() {
		return
	}

	m.histograms["ExecuteBatchOpsCount"].Add(uint64(a.Len()), 1)
	m.histograms["ExecuteBatchBytes"].Add(a.totKeyByte+a.totValByte, 1)

	recordKeyLens := func(hist ghistogram.HistogramMutator) {
		for i := 0; i < len(a.kvs); i += 2 {
			opklvl := a.kvs[i]
			_, length, _ := decodeOpKeyLenValLen(opklvl)
			hist.Add(uint64(length), 1)
		}
	}

	recordValLens := func(hist ghistogram.HistogramMutator) {
		for i := 0; i < len(a.kvs); i += 2 {
			opklvl := a.kvs[i]
			_, _, length := decodeOpKeyLenValLen(opklvl)
			hist.Add(uint64(length), 1)
		}
	}

	m.histograms["MutationKeyBytes"].CallSyncEx(recordKeyLens)
	m.histograms["MutationValBytes"].CallSyncEx(recordValLens)
}

// ------------------------------------------------------

// Log invokes the user's configured Log callback, if any, if the
// debug levels are met.
func (m *collection) Logf(format string, a ...interface{}) {
	if m.options.Debug > 0 &&
		m.options.Log != nil {
		m.options.Log(format, a...)
	}
}

// OnError invokes the user's configured OnError callback, in which
// the application might take further action, for example, such as
// Close()'ing the Collection in order to fix underlying
// storage/resource issues.
func (m *collection) OnError(err error) {
	atomic.AddUint64(&m.stats.TotOnError, 1)

	if m.options.OnError != nil {
		m.options.OnError(err)
	}
}

func (m *collection) fireEvent(kind EventKind, dur time.Duration) {
	if m.options.OnEvent != nil {
		m.options.OnEvent(Event{Kind: kind, Collection: m, Duration: dur})
	}
}

// ------------------------------------------------------

const snapshotSkipDirtyTop = uint32(0x00000001)
const snapshotSkipDirtyMid = uint32(0x00000002)
const snapshotSkipDirtyBase = uint32(0x00000004)
const snapshotSkipClean = uint32(0x00000008)

// snapshot() atomically clones the various stacks into a new, single
// segmentStack, controllable by skip flags, and also invokes the
// optional callback while holding the collection lock.
func (m *collection) snapshot(skip uint32, cb func(*segmentStack),
	gotLock bool) (*segmentStack, int, int, int, int) {
	atomic.AddUint64(&m.stats.TotSnapshotInternalBeg, 1)

	rv := &segmentStack{options: m.options, refs: 1, stats: m.stats}

	heightDirtyTop := 0
	heightDirtyMid := 0
	heightDirtyBase := 0
	heightClean := 0

	if !gotLock {
		m.m.Lock()
	}

	rv.lowerLevelSnapshot = m.lowerLevelSnapshot.addRef()
	if rv.lowerLevelSnapshot != nil {
		rv = m.appendChildLLSnapshot(rv, rv.lowerLevelSnapshot.ss)
	}

	if m.stackDirtyTop != nil && (skip&snapshotSkipDirtyTop == 0) {
		heightDirtyTop = len(m.stackDirtyTop.a)
	}

	if m.stackDirtyMid != nil && (skip&snapshotSkipDirtyMid == 0) {
		heightDirtyMid = len(m.stackDirtyMid.a)
	}

	if m.stackDirtyBase != nil && (skip&snapshotSkipDirtyBase == 0) {
		heightDirtyBase = len(m.stackDirtyBase.a)
	}

	if m.stackClean != nil && (skip&snapshotSkipClean == 0) {
		heightClean = len(m.stackClean.a)
	}

	rv.a = make([]Segment, 0,
		heightDirtyTop+heightDirtyMid+heightDirtyBase+heightClean)

	if m.stackClean != nil && (skip&snapshotSkipClean == 0) {
		rv = m.appendChildStacks(rv, m.stackClean)
	}

	if m.stackDirtyBase != nil && (skip&snapshotSkipDirtyBase == 0) {
		rv = m.appendChildStacks(rv, m.stackDirtyBase)
	}

	if m.stackDirtyMid != nil && (skip&snapshotSkipDirtyMid == 0) {
		rv = m.appendChildStacks(rv, m.stackDirtyMid)
	}

	if m.stackDirtyTop != nil && (skip&snapshotSkipDirtyTop == 0) {
		rv = m.appendChildStacks(rv, m.stackDirtyTop)
	}

	if cb != nil {
		cb(rv)
	}

	if !gotLock {
		m.m.Unlock()
	}

	atomic.AddUint64(&m.stats.TotSnapshotInternalEnd, 1)

	return rv, heightClean, heightDirtyBase, heightDirtyMid, heightDirtyTop
}

// get() retrieves a value by iterating over all the segment stacks,
// and then the lower level snapshot of the collection in pursuit of
// the key, if not found, a nil val is returned.
func (m *collection) get(key []byte, readOptions ReadOptions) ([]byte, error) {
	// Create a pointer to the lower level snapshot by incrementing it's ref
	// count and then pointers to stackClean, stackDirtyBase, stackDirtyMid
	// and stackDirtyTop for the collection within lock.
	m.m.Lock()

	lowerLevelSnapshot := m.lowerLevelSnapshot.addRef()
	stackClean := m.stackClean
	stackDirtyBase := m.stackDirtyBase
	stackDirtyMid := m.stackDirtyMid
	stackDirtyTop := m.stackDirtyTop

	m.m.Unlock()

	var val []byte
	var err error

	// Avoid going to the lower-level snapshot for the
	// stackDirtyTop/Mid/Base/Clean Get()s since their lower level
	// snapshots may be modified concurrently by
	// collection_merger/persister.
	readOptionsSLL := readOptions
	readOptionsSLL.SkipLowerLevel = true

	// Look for the key-value in the collection's segment stacks
	// starting with the latest (stackDirtyTop), followed by
	// stackDirtyMid, stackDirtyBase, stackClean, and if still not
	// found look for it in the lowerLevelSnapshot.
	if stackDirtyTop != nil {
		val, err = stackDirtyTop.Get(key, readOptionsSLL)
	}

	if val == nil && err == nil && stackDirtyMid != nil {
		val, err = stackDirtyMid.Get(key, readOptionsSLL)
	}

	if val == nil && err == nil && stackDirtyBase != nil {
		val, err = stackDirtyBase.Get(key, readOptionsSLL)
	}

	if val == nil && err == nil && stackClean != nil {
		val, err = stackClean.Get(key, readOptionsSLL)
	}

	if lowerLevelSnapshot != nil {
		if val == nil && err == nil {
			val, err = lowerLevelSnapshot.Get(key, readOptions)
		}

		lowerLevelSnapshot.decRef()
	}

	return val, err
}

func (m *collection) getOrInitChildStack(ss *segmentStack,
	childCollName string) *segmentStack {
	if len(ss.childSegStacks) == 0 {
		ss.childSegStacks = make(map[string]*segmentStack)
	}

	dstChildStack, exists := ss.childSegStacks[childCollName]
	if !exists {
		dstChildStack = &segmentStack{
			options:  m.options,
			refs:     1,
			incarNum: m.incarNum,
		}
	}

	return dstChildStack
}

// appendChildLLSnapshot recursively appends lower level child snapshots.
func (m *collection) appendChildLLSnapshot(dst *segmentStack,
	src Snapshot) *segmentStack {
	if m.incarNum != 0 {
		dst.lowerLevelSnapshot = NewSnapshotWrapper(src, nil)
	}

	for cName, childCollection := range m.childCollections {
		dstChildStack := childCollection.getOrInitChildStack(dst, cName)

		var childSnap Snapshot
		if src != nil {
			childSnap, _ = src.ChildCollectionSnapshot(cName)
		}

		dst.childSegStacks[cName] =
			childCollection.appendChildLLSnapshot(dstChildStack, childSnap)
	}

	return dst
}

// appendChildStacks recursively appends child segment stacks.
func (m *collection) appendChildStacks(dst, src *segmentStack) *segmentStack {
	if src == nil {
		return dst
	}

	dst.a = append(dst.a, src.a...)

	for cName, srcChildStack := range src.childSegStacks {
		childCollection, exists := m.childCollections[cName]
		if !exists || // This child collection was dropped recently, OR
			// this child collection was recreated quickly.
			childCollection.incarNum != srcChildStack.incarNum {
			continue
		}

		dstChildStack := childCollection.getOrInitChildStack(dst, cName)

		dst.childSegStacks[cName] =
			childCollection.appendChildStacks(dstChildStack, srcChildStack)
	}

	return dst
}
