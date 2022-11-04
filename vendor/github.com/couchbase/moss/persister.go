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
	"sync/atomic"
	"time"
)

// runPersister() implements the persister task.
func (m *collection) runPersister() {
	defer func() {
		close(m.donePersisterCh)

		atomic.AddUint64(&m.stats.TotPersisterEnd, 1)
	}()

	if m.options.LowerLevelUpdate == nil {
		return
	}

OUTER:
	for {
		atomic.AddUint64(&m.stats.TotPersisterLoop, 1)

		m.m.Lock()

		for m.stackDirtyBase == nil && !m.isClosed() {
			// There's a concurrency scenario where imagine that
			// persistence takes a long time.  Also, imagine that
			// there are no more incoming batches (so, stackDirtyTop
			// is empty).
			//
			// That allows the merger to complete a merging cycle (so,
			// stackDirtyMid is non-empty with unpersisted data) and
			// the merger is now just waiting for either more incoming
			// batches or waiting to be awoken.
			//
			// So, we notify/awake the merger here so that it can feed
			// stackDirtyMid down to the persister as stackDirtyBase.
			if m.waitDirtyIncomingCh != nil && // Merger is indeed asleep.
				(m.stackDirtyMid != nil && len(m.stackDirtyMid.a) > 0) &&
				(m.stackDirtyTop == nil || len(m.stackDirtyTop.a) <= 0) {
				m.NotifyMerger("from-persister", false)
			}

			atomic.AddUint64(&m.stats.TotPersisterWaitBeg, 1)
			m.stackDirtyBaseCond.Wait()
			atomic.AddUint64(&m.stats.TotPersisterWaitEnd, 1)
		}

		stackDirtyBase := m.stackDirtyBase

		m.m.Unlock()

		if m.isClosed() {
			return
		}

		startTime := time.Now()

		atomic.AddUint64(&m.stats.TotPersisterLowerLevelUpdateBeg, 1)

		llssNext, err := m.options.LowerLevelUpdate(stackDirtyBase)
		if err != nil {
			atomic.AddUint64(&m.stats.TotPersisterLowerLevelUpdateErr, 1)

			m.Logf("collection: runPersister, LowerLevelUpdate, err: %v", err)

			m.OnError(err)

			continue OUTER
		}

		atomic.AddUint64(&m.stats.TotPersisterLowerLevelUpdateEnd, 1)

		var stackDirtyBasePrev *segmentStack
		var stackCleanPrev *segmentStack

		m.m.Lock()

		m.invalidateLatestSnapshotLOCKED()

		stackCleanPrev = m.stackClean
		if m.options.CachePersisted {
			m.stackClean = m.stackDirtyBase
		} else {
			m.stackClean = nil

			stackDirtyBasePrev = m.stackDirtyBase
		}
		m.stackDirtyBase = nil

		waitDirtyOutgoingCh := m.waitDirtyOutgoingCh
		m.waitDirtyOutgoingCh = nil

		llssPrev := m.lowerLevelSnapshot
		m.lowerLevelSnapshot = NewSnapshotWrapper(llssNext, nil)

		m.m.Unlock()

		if stackDirtyBasePrev != nil {
			stackDirtyBasePrev.Close()
		}

		if stackCleanPrev != nil {
			stackCleanPrev.Close()
		}

		if llssPrev != nil {
			llssPrev.Close()
		}

		if waitDirtyOutgoingCh != nil {
			close(waitDirtyOutgoingCh)
		}

		// ---------------------------------------------

		atomic.AddUint64(&m.stats.TotPersisterLoopRepeat, 1)

		m.fireEvent(EventKindPersisterProgress, time.Now().Sub(startTime))
	}

	// TODO: More advanced eviction of stackClean.
	// TODO: Timer based eviction of stackClean?
	// TODO: Randomized eviction?
	// TODO: Merging of stackClean to 1 level?
	// TODO: WaitForMerger() also considers stackClean?
	// TODO: Track popular Get() keys?
	// TODO: Track shadowing during merges for writes.
}
