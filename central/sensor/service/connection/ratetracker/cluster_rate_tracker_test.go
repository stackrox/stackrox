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

type testClusterWithMessages struct {
	name    string
	numMsgs int
}

func getTestClusters() []testClusterWithMessages {
	return []testClusterWithMessages{
		{"c1", 1},
		{"c2", 8},
		{"c3", 9},
		{"c4", 5},
		{"c5", 3},
	}
}

func TestMessageRateTrackerHeap(t *testing.T) {
	clusters := getTestClusters()
	tracker := newMessageRateTracker(time.Minute, 3)
	for i := 0; i < 10; i++ {
		for c := 0; c < len(clusters); c++ {
			if clusters[c].numMsgs > 0 {
				tracker.ReceiveMsg(clusters[c].name)
				clusters[c].numMsgs--
			}
		}
	}

	// Validate candidate clusters.
	assert.Equal(t, tracker.maxClusters, len(tracker.clusterLimitCandidates))
	for _, clusterID := range []string{"c4", "c3", "c2"} {
		assert.True(t, tracker.IsTopCluster(clusterID))
	}

	for _, clusterID := range []string{"c1", "c5"} {
		assert.False(t, tracker.IsTopCluster(clusterID))
	}

	// Validate heap is correct.
	assert.Equal(t, tracker.maxClusters, len(tracker.clusterLimitCandidates))
	var throttleCandidates []string
	for len(*tracker.clusterRatesHeap) > 0 {
		throttleCandidates = append(throttleCandidates, heap.Pop(tracker.clusterRatesHeap).(*clusterRate).clusterID)
	}
	assert.ElementsMatch(t, []string{"c4", "c3", "c2"}, throttleCandidates)
}

func TestMessageRateTrackerRemove(t *testing.T) {
	clusters := getTestClusters()

	tracker := newMessageRateTracker(time.Minute, 3)
	for i := 0; i < 10; i++ {
		for c := 0; c < len(clusters); c++ {
			if clusters[c].numMsgs > 0 {
				tracker.ReceiveMsg(clusters[c].name)
				clusters[c].numMsgs--
			}
		}
	}

	// Validate full list of clusters.
	assert.True(t, tracker.IsTopCluster("c3"))
	assert.Equal(t, 5, len(tracker.clusterRates))
	assert.Equal(t, 3, tracker.clusterRatesHeap.Len())
	assert.Equal(t, 3, len(tracker.clusterLimitCandidates))

	tracker.Remove("c3")

	assert.False(t, tracker.IsTopCluster("c3"))
	assert.Equal(t, 4, len(tracker.clusterRates))
	assert.Equal(t, 2, tracker.clusterRatesHeap.Len())
	assert.Equal(t, 2, len(tracker.clusterLimitCandidates))

	assert.NotPanics(t, func() { tracker.Remove("c3") })
}
