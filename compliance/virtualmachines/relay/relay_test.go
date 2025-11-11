package relay

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestVMRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *relayTestSuite) TestParseIndexReport() {
	data := []byte("malformed-data")
	parsedIndexReport, err := parseIndexReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedIndexReport)

	validIndexReport := &v1.IndexReport{VsockCid: "42"}
	data, err = proto.Marshal(validIndexReport)
	s.Require().NoError(err)
	parsedIndexReport, err = parseIndexReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validIndexReport, parsedIndexReport))
}

func (s *relayTestSuite) TestSendReportToSensor_HandlesContextCancellation() {
	client := newMockSensorClient().withDelay(100 * time.Millisecond)
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Millisecond)
	defer cancel()

	err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
	s.Require().Error(err)
	s.Contains(err.Error(), "context deadline exceeded")
}

func (s *relayTestSuite) TestSendReportToSensor_RetriesOnRetryableErrors() {
	cases := map[string]struct {
		err         error
		respSuccess bool
		shouldRetry bool
	}{
		"retryable error is retried": {
			err:         status.Error(codes.ResourceExhausted, "retryable error"),
			respSuccess: false,
			shouldRetry: true,
		},
		"non-retryable error is not retried": {
			err:         errox.NotImplemented,
			respSuccess: false,
			shouldRetry: false,
		},
		"Unsuccessful request is retried": {
			err:         nil,
			respSuccess: false,
			shouldRetry: true,
		},
	}
	for name, c := range cases {
		s.Run(name, func() {
			client := newMockSensorClient().withError(c.err)
			if !c.respSuccess {
				client = client.withUnsuccessfulResponse()
			}

			// The retry logic uses withExponentialBackoff, which currently has an initial delay between retries of
			// 100 ms, therefore after 500 ms the failing call has been retried already
			ctx, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
			defer cancel()

			err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
			s.Require().Error(err)

			retried := len(client.capturedRequests) > 1
			s.Equal(c.shouldRetry, retried)
		})
	}
}

func (s *relayTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	indexReport := v1.IndexReport{VsockCid: "42"}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().NoError(err)
}

func (s *relayTestSuite) TestRelay_RespectsConcurrentConnectionLimit() {
	maxConcurrent := 2
	mockServer := newMockVsockServer(maxConcurrent)
	mockSensorClient := newMockSensorClient()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	relay := &Relay{
		connectionReadTimeout: 2 * time.Second,
		ctx:                   ctx,
		sensorClient:          mockSensorClient,
		vsockServer:           mockServer,
		waitAfterFailedAccept: 10 * time.Millisecond,
	}

	// Start the relay in a goroutine
	relayStopped := make(chan error)
	go func() {
		relayStopped <- relay.Run()
	}()

	// Connection 1 - should be accepted and will block on read
	conn1 := createMockVsockConnection(100)
	mockServer.acceptChan <- conn1
	mockServer.acquireResultChan <- nil // Semaphore acquired
	<-conn1.readStarted                 // Wait until it starts reading

	// Connection 2 - should be accepted and will block on read
	conn2 := createMockVsockConnection(101)
	mockServer.acceptChan <- conn2
	mockServer.acquireResultChan <- nil // Semaphore acquired
	<-conn2.readStarted                 // Wait until it starts reading

	// Both connections should now be holding semaphore slots
	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent, mockServer.getCurrentAcquired())

	// Connection 3 - should fail to acquire semaphore (limit reached)
	conn3 := createMockVsockConnection(102)
	mockServer.acceptChan <- conn3
	mockServer.acquireResultChan <- context.DeadlineExceeded // Semaphore timeout

	// Wait for connection 3 to be closed
	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent, mockServer.getCurrentAcquired(),
		"Should still have exactly maxConcurrent connections")

	// Allow first connection to complete
	conn1.signalReadReady()
	<-mockServer.releaseCallChan
	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent-1, mockServer.getCurrentAcquired())

	// Connection 4 - should now be accepted (slot available)
	conn4 := createMockVsockConnection(103)
	mockServer.acceptChan <- conn4
	mockServer.acquireResultChan <- nil
	<-conn4.readStarted // Wait until it starts reading
	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent, mockServer.getCurrentAcquired())

	// Cleanup: complete remaining connections
	conn2.signalReadReady()
	conn4.signalReadReady()
	cancel()
	<-relayStopped
}

func (s *relayTestSuite) TestRelay_ClosesConnectionWhenSemaphoreUnavailable() {
	mockServer := newMockVsockServer(1)
	mockSensorClient := newMockSensorClient()

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	relay := &Relay{
		connectionReadTimeout: 2 * time.Second,
		ctx:                   ctx,
		sensorClient:          mockSensorClient,
		vsockServer:           mockServer,
		waitAfterFailedAccept: 10 * time.Millisecond,
	}

	relayStopped := make(chan error)
	go func() {
		relayStopped <- relay.Run()
	}()

	// Create a connection that tracks if Close() was called
	rejectedConn := newConnectionCloseTracker(
		createMockVsockConnectionImmediate(100),
	)

	// Send connection that will fail semaphore acquisition
	mockServer.acceptChan <- rejectedConn
	mockServer.acquireResultChan <- context.DeadlineExceeded

	// Verify connection was closed
	select {
	case <-rejectedConn.closed:
		// Expected - connection was closed
	case <-time.After(500 * time.Millisecond):
		s.Fail("Connection was not closed after semaphore failure")
	}

	// Wait for relay to stop
	cancel()
	<-relayStopped
}

func (s *relayTestSuite) TestRelay_HandlesMultipleWavesOfConnections() {
	maxConcurrent := 3
	mockServer := newMockVsockServer(maxConcurrent)
	mockSensorClient := newMockSensorClient()

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	relay := &Relay{
		connectionReadTimeout: 2 * time.Second,
		ctx:                   ctx,
		sensorClient:          mockSensorClient,
		vsockServer:           mockServer,
		waitAfterFailedAccept: 10 * time.Millisecond,
	}

	relayStopped := make(chan error)
	go func() {
		relayStopped <- relay.Run()
	}()

	// Wave 1: Send maxConcurrent connections that block on read
	wave1Conns := make([]*mockVsockConn, maxConcurrent)
	for i := 0; i < maxConcurrent; i++ {
		conn := createMockVsockConnection(uint32(100 + i))
		wave1Conns[i] = conn
		mockServer.acceptChan <- conn
		mockServer.acquireResultChan <- nil
		<-conn.readStarted // Wait until each starts reading
	}

	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent, mockServer.getCurrentAcquired())

	// Wave 2: Send 2 more - should be rejected
	for i := 0; i < 2; i++ {
		mockServer.acceptChan <- createMockVsockConnectionImmediate(uint32(200 + i))
		mockServer.acquireResultChan <- context.DeadlineExceeded
	}

	time.Sleep(50 * time.Millisecond)
	s.Equal(maxConcurrent, mockServer.getCurrentAcquired(),
		"Rejected connections should not increase count")

	// Complete wave 1 connections
	for _, conn := range wave1Conns {
		conn.signalReadReady()
	}

	// Wait for wave 1 to complete
	for i := 0; i < maxConcurrent; i++ {
		<-mockServer.releaseCallChan
	}

	time.Sleep(50 * time.Millisecond)
	s.Equal(0, mockServer.getCurrentAcquired())

	// Wave 3: Send another batch - should all succeed now
	for i := 0; i < maxConcurrent; i++ {
		mockServer.acceptChan <- createMockVsockConnectionImmediate(uint32(300 + i))
		mockServer.acquireResultChan <- nil
	}

	time.Sleep(100 * time.Millisecond)
	// By the time we check, wave 3 connections should have completed quickly
	// so the semaphore count might be 0 or close to 0
	s.LessOrEqual(mockServer.getCurrentAcquired(), maxConcurrent,
		"Should never exceed max concurrent")

	// Wait for relay to stop
	cancel()
	<-relayStopped
}
