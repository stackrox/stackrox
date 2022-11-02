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
	"math"
	"sync/atomic"
	"time"
)

// NotifyMerger sends a message (optionally synchronously) to the merger
// to run another cycle.  Providing a kind of "mergeAll" forces a full
// merge and can be useful for applications that are no longer
// performing mutations and that want to optimize for retrievals.
func (m *collection) NotifyMerger(kind string, synchronous bool) error {
	atomic.AddUint64(&m.stats.TotNotifyMergerBeg, 1)

	var pongCh chan struct{}
	if synchronous {
		pongCh = make(chan struct{})
	}

	m.pingMergerCh <- ping{
		kind:   kind,
		pongCh: pongCh,
	}

	if pongCh != nil {
		<-pongCh
	}

	atomic.AddUint64(&m.stats.TotNotifyMergerEnd, 1)

	return nil
}

// ------------------------------------------------------

// runMerger() implements the background merger task.
func (m *collection) runMerger() {
	defer func() {
		close(m.doneMergerCh)

		atomic.AddUint64(&m.stats.TotMergerEnd, 1)
	}()

	maxPreMergerBatches := m.options.MaxPreMergerBatches
	if maxPreMergerBatches <= 0 {
		maxPreMergerBatches =
			DefaultCollectionOptions.MaxPreMergerBatches
	}

	if maxPreMergerBatches >= math.MaxInt32 {
		return // Way to disable merger.
	}

	pings := []ping{}

	defer func() {
		replyToPings(pings)
		pings = pings[0:0]
	}()

	go m.idleMergerWaker()

OUTER:
	for {
		atomic.AddUint64(&m.stats.TotMergerLoop, 1)

		// ---------------------------------------------
		// Notify ping'ers from the previous loop.

		replyToPings(pings)
		pings = pings[0:0]

		// ---------------------------------------------
		// Wait for new stackDirtyTop entries and/or pings.

		var stopped, mergeAll bool
		stopped, mergeAll, pings = m.mergerWaitForWork(pings)
		if stopped {
			return
		}

		// ---------------------------------------------
		// Atomically ingest stackDirtyTop into stackDirtyMid.

		var stackDirtyTopPrev *segmentStack
		var stackDirtyMidPrev *segmentStack
		var stackDirtyBase *segmentStack

		stackDirtyMid, _, _, _, _ :=
			m.snapshot(snapshotSkipClean|snapshotSkipDirtyBase,
				func(ss *segmentStack) {
					m.invalidateLatestSnapshotLOCKED()

					// m.stackDirtyMid takes 1 refs, and
					// stackDirtyMid takes 1 refs.
					ss.refs++

					stackDirtyTopPrev = m.stackDirtyTop
					m.stackDirtyTop = nil

					stackDirtyMidPrev = m.stackDirtyMid
					m.stackDirtyMid = ss

					stackDirtyBase = m.stackDirtyBase
					if stackDirtyBase != nil {
						// While waiting for persistence, might as well do
						mergeAll = true // a full merge to optimize reads.

						stackDirtyBase.addRef()
					}

					// Awake writers waiting for space in stackDirtyTop.
					m.stackDirtyTopCond.Broadcast()
				},
				false) // The collection level lock needs to be acquired.

		stackDirtyTopPrev.Close()
		stackDirtyMidPrev.Close()

		// ---------------------------------------------
		// Merge multiple stackDirtyMid layers.

		startTime := time.Now()

		mergerWasOk := m.mergerMain(stackDirtyMid, stackDirtyBase, mergeAll)
		if !mergerWasOk {
			continue OUTER
		}

		m.histograms["MergerUsecs"].Add(
			uint64(time.Since(startTime).Nanoseconds()/1000), 1)

		// ---------------------------------------------
		// Notify persister.

		m.mergerNotifyPersister()

		// ---------------------------------------------

		atomic.AddUint64(&m.stats.TotMergerLoopRepeat, 1)

		m.fireEvent(EventKindMergerProgress, time.Now().Sub(startTime))
	}

	// TODO: Concurrent merging of disjoint slices of stackDirtyMid
	// instead of the current, single-threaded merger?
	//
	// TODO: A busy merger means no feeding of the persister?
	//
	// TODO: Delay merger until lots of deletion tombstones?
	//
	// TODO: The base layer is likely the largest, so instead of heap
	// merging the base layer entries, treat the base layer with
	// special case to binary search to find better start points?
	//
	// TODO: Dynamically calc'ed soft max dirty top height, for
	// read-heavy (favor lower) versus write-heavy (favor higher)
	// situations?
}

// ------------------------------------------------------

func (m *collection) idleMergerWaker() {
	idleTimeout := m.options.MergerIdleRunTimeoutMS
	if idleTimeout == 0 {
		idleTimeout = DefaultCollectionOptions.MergerIdleRunTimeoutMS
	}
	if idleTimeout <= 0 { // -1 will disable idle compactions.
		return
	}
	var prevRunID uint64 // Helps run idle merger IFF new data has come in.
	for {
		if atomic.LoadUint64(&m.stats.TotCloseBeg) > 0 {
			return
		}
		atomic.AddUint64(&m.stats.TotMergerIdleSleeps, 1)
		napIDBefore := atomic.LoadUint64(&m.stats.TotMergerWaitIncomingEnd)

		time.Sleep(time.Duration(idleTimeout) * time.Millisecond)

		napIDAfter := atomic.LoadUint64(&m.stats.TotMergerWaitIncomingEnd)
		napID := atomic.LoadUint64(&m.stats.TotMergerWaitIncomingBeg)
		if napID == napIDAfter+1 && // Merger is indeed asleep.
			napIDBefore == napIDAfter && // Nap only while merger naps.
			atomic.LoadUint64(&m.stats.TotMergerInternalEnd) > prevRunID { // New data.
			m.NotifyMerger("from-idle-merger", false)
			prevRunID = atomic.LoadUint64(&m.stats.TotMergerInternalEnd)
		}
	}
}

// mergerWaitForWork() is a helper method that blocks until there's
// either pings or incoming segments (from ExecuteBatch()) of work for
// the merger.
func (m *collection) mergerWaitForWork(pings []ping) (
	stopped, mergeAll bool, pingsOut []ping) {
	var waitDirtyIncomingCh chan struct{}

	m.m.Lock()

	if m.stackDirtyTop == nil || len(m.stackDirtyTop.a) <= 0 {
		m.waitDirtyIncomingCh = make(chan struct{})
		waitDirtyIncomingCh = m.waitDirtyIncomingCh
	}

	m.m.Unlock()

	if waitDirtyIncomingCh != nil {
		atomic.AddUint64(&m.stats.TotMergerWaitIncomingBeg, 1)

		select {
		case <-m.stopCh:
			atomic.AddUint64(&m.stats.TotMergerWaitIncomingStop, 1)
			return true, mergeAll, pings

		case pingVal := <-m.pingMergerCh:
			pings = append(pings, pingVal)
			if pingVal.kind == "mergeAll" {
				mergeAll = true
			} else if pingVal.kind == "from-idle-merger" {
				atomic.AddUint64(&m.stats.TotMergerIdleRuns, 1)
				mergeAll = true
			}

		case <-waitDirtyIncomingCh:
			// NO-OP.
		}

		atomic.AddUint64(&m.stats.TotMergerWaitIncomingEnd, 1)
	} else {
		atomic.AddUint64(&m.stats.TotMergerWaitIncomingSkip, 1)
	}

	pings, mergeAll = receivePings(m.pingMergerCh, pings, "mergeAll", mergeAll)

	return false, mergeAll, pings
}

// ------------------------------------------------------

// mergerMain() is a helper method that performs the merging work on
// the stackDirtyMid and swaps the merged result into the collection.
func (m *collection) mergerMain(stackDirtyMid, stackDirtyBase *segmentStack,
	mergeAll bool) (ok bool) {
	if stackDirtyMid != nil && !stackDirtyMid.isEmpty() {
		atomic.AddUint64(&m.stats.TotMergerInternalBeg, 1)
		mergedStackDirtyMid, numFullMerges, err := stackDirtyMid.merge(mergeAll,
			stackDirtyBase)
		if err != nil {
			atomic.AddUint64(&m.stats.TotMergerInternalErr, 1)

			m.Logf("collection: mergerMain stackDirtyMid.merge,"+
				" numFullMerges: %d, err: %v", numFullMerges, err)

			m.OnError(err)

			stackDirtyMid.Close()
			stackDirtyBase.Close()

			return false
		}

		atomic.AddUint64(&m.stats.TotMergerAll, numFullMerges)
		atomic.AddUint64(&m.stats.TotMergerInternalEnd, 1)

		stackDirtyMid.Close()

		mergedStackDirtyMid.addRef()
		stackDirtyMid = mergedStackDirtyMid

		m.m.Lock()
		stackDirtyMidPrev := m.stackDirtyMid
		m.stackDirtyMid = mergedStackDirtyMid
		m.m.Unlock()

		stackDirtyMidPrev.Close()

	} else {
		if stackDirtyMid != nil && stackDirtyMid.isEmpty() {
			// Do this only for idle-compactions.
			atomic.AddUint64(&m.stats.TotMergerEmptyDirtyMid, 1)
			m.m.Lock() // Allow an empty stackDirtyMid to kick persistence.
			stackDirtyMidPrev := m.stackDirtyMid
			m.stackDirtyMid = stackDirtyMid
			m.m.Unlock()

			stackDirtyMidPrev.Close()
		}
		atomic.AddUint64(&m.stats.TotMergerInternalSkip, 1)
	}

	stackDirtyBase.Close()

	lenDirtyMid := len(stackDirtyMid.a)
	if lenDirtyMid > 0 {
		topDirtyMid := stackDirtyMid.a[lenDirtyMid-1]

		m.Logf("collection: mergerMain, dirtyMid height: %2d,"+
			" dirtyMid top # entries: %d", lenDirtyMid, topDirtyMid.Len())
	}

	stackDirtyMid.Close()

	return true
}

// ------------------------------------------------------

// mergerNotifyPersister() is a helper method that notifies the
// optional persister goroutine that there's a dirty segment stack
// that needs persistence.
func (m *collection) mergerNotifyPersister() {
	if m.options.LowerLevelUpdate == nil {
		return
	}

	m.m.Lock()

	if m.stackDirtyBase == nil && m.stackDirtyMid != nil {
		atomic.AddUint64(&m.stats.TotMergerLowerLevelNotify, 1)

		m.stackDirtyBase = m.stackDirtyMid
		m.stackDirtyMid = nil

		prevLowerLevelSnapshot := m.stackDirtyBase.lowerLevelSnapshot
		m.stackDirtyBase.lowerLevelSnapshot = m.lowerLevelSnapshot.addRef()
		if prevLowerLevelSnapshot != nil {
			prevLowerLevelSnapshot.decRef()
		}

		if m.waitDirtyOutgoingCh != nil {
			close(m.waitDirtyOutgoingCh)
		}
		m.waitDirtyOutgoingCh = make(chan struct{})

		m.stackDirtyBaseCond.Broadcast()
	} else {
		atomic.AddUint64(&m.stats.TotMergerLowerLevelNotifySkip, 1)
	}

	var waitDirtyOutgoingCh chan struct{}

	if m.options.MaxDirtyOps > 0 || m.options.MaxDirtyKeyValBytes > 0 {
		cs := CollectionStats{}

		m.statsSegmentsLOCKED(&cs)

		if cs.CurDirtyOps > m.options.MaxDirtyOps ||
			cs.CurDirtyBytes > m.options.MaxDirtyKeyValBytes {
			waitDirtyOutgoingCh = m.waitDirtyOutgoingCh
		}
	}

	m.m.Unlock()

	if waitDirtyOutgoingCh != nil {
		atomic.AddUint64(&m.stats.TotMergerWaitOutgoingBeg, 1)

		select {
		case <-m.stopCh:
			atomic.AddUint64(&m.stats.TotMergerWaitOutgoingStop, 1)
			return

		case <-waitDirtyOutgoingCh:
			// NO-OP.
		}

		atomic.AddUint64(&m.stats.TotMergerWaitOutgoingEnd, 1)
	} else {
		atomic.AddUint64(&m.stats.TotMergerWaitOutgoingSkip, 1)
	}
}
