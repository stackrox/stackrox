package ratetracker

import (
	"container/heap"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTrackerNilGuards(t *testing.T) {
	var tracker *messageRateTracker

	assert.Nil(t, tracker)
	assert.NotPanics(t, func() { tracker.ReceiveMsg("test-1") })
	assert.NotPanics(t, func() { tracker.IsTopCluster("test-1") })
	assert.NotPanics(t, func() { tracker.Remove("test-1") })
}

func TestHeapWorks(t *testing.T) {
	clusters := []struct {
		name    string
		numMsgs int
	}{
		{"c1", 1},
		{"c2", 8},
		{"c3", 9},
		{"c4", 5},
		{"c5", 3},
	}

	tracker := newMessageRateTracker(time.Minute, 3)
	for i := 0; i < 10; i++ {
		for c := 0; c < len(clusters); c++ {
			if clusters[c].numMsgs > 0 {
				tracker.ReceiveMsg(clusters[c].name)
				clusters[c].numMsgs--
			}
		}
	}

	var throttleCandidates []string
	for len(*tracker.clusterRatesHeap) > 0 {
		throttleCandidates = append(throttleCandidates, heap.Pop(tracker.clusterRatesHeap).(*clusterRate).clusterID)
	}

	assert.ElementsMatch(t, []string{"c4", "c3", "c2"}, throttleCandidates)
}
