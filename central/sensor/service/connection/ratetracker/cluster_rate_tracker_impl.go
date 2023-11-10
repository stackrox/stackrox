package ratetracker

import (
	"container/heap"
	"time"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// Tracker
type messageRateTracker struct {
	mutex sync.Mutex

	ratePeriod  time.Duration
	maxClusters int

	clusterRates           map[string]*clusterRate
	clusterRatesHeap       *clusterRatesHeap
	clusterLimitCandidates set.StringSet
}

func (t *messageRateTracker) getClusterRate(clusterID string) *clusterRate {
	clRate, found := t.clusterRates[clusterID]
	if !found {
		clRate = newClusterRate(clusterID, t.ratePeriod)
		t.clusterRates[clusterID] = clRate
	}

	return clRate
}

func (t *messageRateTracker) updateTopClusters(clRate *clusterRate) {
	if t.clusterLimitCandidates.Contains(clRate.clusterID) {
		heap.Fix(t.clusterRatesHeap, clRate.index)

		return
	}

	heap.Push(t.clusterRatesHeap, clRate)
	t.clusterLimitCandidates.Add(clRate.clusterID)

	if t.clusterRatesHeap.Len() > t.maxClusters {
		droppedCandidate := heap.Pop(t.clusterRatesHeap).(*clusterRate)
		t.clusterLimitCandidates.Remove(droppedCandidate.clusterID)
	}
}

func newMessageRateTracker(period time.Duration, maxClusters int) *messageRateTracker {
	return &messageRateTracker{
		ratePeriod:  period,
		maxClusters: maxClusters,

		clusterRates:           make(map[string]*clusterRate),
		clusterRatesHeap:       &clusterRatesHeap{},
		clusterLimitCandidates: set.NewStringSet(),
	}
}

func (t *messageRateTracker) ReceiveMsg(clusterID string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	clRate := t.getClusterRate(clusterID)
	clRate.receiveMsg()
	t.updateTopClusters(clRate)
}

func (t *messageRateTracker) IsTopCluster(clusterID string) bool {
	if t == nil {
		return false
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.clusterLimitCandidates.Contains(clusterID)
}

func (t *messageRateTracker) Remove(clusterID string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	clRate, found := t.clusterRates[clusterID]
	if !found {
		return
	}

	delete(t.clusterRates, clusterID)
	if t.clusterLimitCandidates.Contains(clusterID) {
		heap.Remove(t.clusterRatesHeap, clRate.index)
		t.clusterLimitCandidates.Remove(clusterID)
	}
}

func NewClusterRateTracker(period time.Duration, maxClusters int) ClusterRateTracker {
	return newMessageRateTracker(period, maxClusters)
}
