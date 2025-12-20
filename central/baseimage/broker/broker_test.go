package broker

import (
	"context"
	"testing"
	"time"

	connMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBroker(t *testing.T) {
	suite.Run(t, &BrokerTestSuite{})
}

type BrokerTestSuite struct {
	suite.Suite
}

// TestOnClusterDisconnect verifies that OnClusterDisconnect properly cleans up
// all pending requests for a cluster.
func (s *BrokerTestSuite) TestOnClusterDisconnect() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	mockConnMgr := connMocks.NewMockManager(mockCtrl)
	stopSig := concurrency.NewErrorSignal()

	broker := New(mockConnMgr, &stopSig)

	// Simulate two requests from cluster1 and one from cluster2.
	cluster1ID := "cluster-1"
	cluster2ID := "cluster-2"

	// Manually set up state as if StreamRepoScan had been called.
	broker.chansMutex.Lock()
	broker.chans[cluster1ID] = map[string]chan *central.RepoScanResponse{
		"req1": make(chan *central.RepoScanResponse, 10),
		"req2": make(chan *central.RepoScanResponse, 10),
	}
	broker.chans[cluster2ID] = map[string]chan *central.RepoScanResponse{
		"req3": make(chan *central.RepoScanResponse, 10),
	}
	broker.chansMutex.Unlock()

	// Verify initial state.
	s.Len(broker.chans, 2)
	s.Len(broker.chans[cluster1ID], 2)
	s.Len(broker.chans[cluster2ID], 1)

	// Disconnect cluster1.
	broker.OnClusterDisconnect(cluster1ID)

	// Verify cluster1's requests are cleaned up.
	broker.chansMutex.Lock()
	defer broker.chansMutex.Unlock()

	s.Len(broker.chans, 1, "should only have cluster2 left")
	_, ok := broker.chans[cluster1ID]
	s.False(ok, "cluster1 should be completely removed")

	s.Len(broker.chans[cluster2ID], 1)
	_, ok = broker.chans[cluster2ID]["req3"]
	s.True(ok, "cluster2's request should still exist")
}

// TestOnClusterDisconnectClosesChannels verifies that channels are closed
// when a cluster disconnects, allowing waiting goroutines to unblock.
func (s *BrokerTestSuite) TestOnClusterDisconnectClosesChannels() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	mockConnMgr := connMocks.NewMockManager(mockCtrl)
	stopSig := concurrency.NewErrorSignal()

	broker := New(mockConnMgr, &stopSig)

	clusterID := "cluster-1"
	ch := make(chan *central.RepoScanResponse, 10)

	broker.chansMutex.Lock()
	broker.chans[clusterID] = map[string]chan *central.RepoScanResponse{
		"req1": ch,
	}
	broker.chansMutex.Unlock()

	// Goroutine simulating StreamRepoScan waiting for response.
	done := make(chan bool)
	go func() {
		_, ok := <-ch
		s.False(ok, "channel should be closed")
		done <- true
	}()

	// Disconnect cluster - should close the channel.
	broker.OnClusterDisconnect(clusterID)

	// Verify the goroutine unblocked.
	select {
	case <-done:
		// Success - goroutine unblocked
	case <-time.After(100 * time.Millisecond):
		s.Fail("goroutine should have unblocked when channel closed")
	}
}

// TestNotifyRepoScanReceivedWithNoChannel verifies that notifications
// for unknown request IDs are handled gracefully.
func (s *BrokerTestSuite) TestNotifyRepoScanReceivedWithNoChannel() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	mockConnMgr := connMocks.NewMockManager(mockCtrl)
	stopSig := concurrency.NewErrorSignal()

	broker := New(mockConnMgr, &stopSig)

	clusterID := "cluster-1"
	resp := &central.RepoScanResponse{
		RequestId: "unknown-request",
		Payload: &central.RepoScanResponse_Start_{
			Start: &central.RepoScanResponse_Start{},
		},
	}

	// Should not panic or deadlock.
	// This logs a warning but doesn't crash.
	broker.OnScanResponse(clusterID, resp)

	// Verify state - since cluster doesn't exist, nothing should be created.
	broker.chansMutex.Lock()
	defer broker.chansMutex.Unlock()

	_, ok := broker.chans[clusterID]
	s.False(ok, "no cluster entry should be created for unknown cluster")
}

// TestStreamRepoScanCleansUpOnReturn verifies that StreamRepoScan
// properly cleans up its state when it returns.
func (s *BrokerTestSuite) TestStreamRepoScanCleansUpOnReturn() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	mockConnMgr := connMocks.NewMockManager(mockCtrl)
	stopSig := concurrency.NewErrorSignal()

	broker := New(mockConnMgr, &stopSig)

	clusterID := "cluster-1"
	req := &central.RepoScanRequest{
		Repository: "registry.example.com/repo",
		TagPattern: "*",
	}

	// Mock SendMessage to succeed but never send responses.
	// Expect: 1) initial request, 2) cancellation on context timeout, 3) cancellation in defer.
	mockConnMgr.EXPECT().SendMessage(clusterID, gomock.Any()).Return(nil).AnyTimes()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Call StreamRepoScan - should timeout via context.
	var iterErr error
	for _, err := range broker.StreamRepoScan(ctx, clusterID, req) {
		if err != nil {
			iterErr = err
			break
		}
	}

	s.Error(iterErr, "should timeout")
	s.Contains(iterErr.Error(), "context done")

	// Verify state is cleaned up.
	// Wait a bit for the goroutine cleanup to complete.
	time.Sleep(10 * time.Millisecond)

	broker.chansMutex.Lock()
	defer broker.chansMutex.Unlock()

	// The defer in StreamRepoScan should have deleted the entry.
	_, ok := broker.chans[clusterID]
	s.False(ok, "cluster entry should be deleted after cleanup")
}
