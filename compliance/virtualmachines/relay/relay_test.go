package relay

//go:generate mockgen-wrapper Conn net
//go:generate mockgen-wrapper VirtualMachineIndexReportServiceClient github.com/stackrox/rox/generated/internalapi/sensor

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	sensormocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/github.com/stackrox/rox/generated/internalapi/sensor/mocks"
	netmocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/net/mocks"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestVMRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite

	ctx      context.Context
	mockCtrl *gomock.Controller
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *relayTestSuite) TearDownTest() {
	if s.mockCtrl != nil {
		s.mockCtrl.Finish()
	}
}

func (s *relayTestSuite) TestExtractVsockCIDFromConnection() {
	cases := map[string]struct {
		setupConn        func() net.Conn
		shouldError      bool
		expectedVsockCID uint32
	}{
		"wrong type fails": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&net.TCPAddr{}).AnyTimes()
				return conn
			},
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"reserved vsock CID fails": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: 2}).AnyTimes()
				return conn
			},
			shouldError:      true,
			expectedVsockCID: 0,
		},
		"valid vsock CID succeeds": {
			setupConn: func() net.Conn {
				conn := netmocks.NewMockConn(s.mockCtrl)
				conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: 42}).AnyTimes()
				return conn
			},
			shouldError:      false,
			expectedVsockCID: 42,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			conn := c.setupConn()
			vsockCID, err := extractVsockCIDFromConnection(conn)
			if c.shouldError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(c.expectedVsockCID, vsockCID)
			}
		})
	}
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

func (s *relayTestSuite) TestReadFromConn() {
	data := []byte("Hello, world!")

	cases := map[string]struct {
		delay       time.Duration
		maxSize     int
		readTimeout time.Duration
		shouldError bool
	}{
		"data smaller than limit succeeds": {
			maxSize:     2 * len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data of equal size as limit succeeds": {
			maxSize:     len(data),
			readTimeout: 10 * time.Second,
			shouldError: false,
		},
		"data larger than limit fails": {
			maxSize:     len(data) - 1,
			readTimeout: 10 * time.Second,
			shouldError: true,
		},
		"delay longer than timeout fails": {
			maxSize:     len(data),
			delay:       1 * time.Second,
			readTimeout: 100 * time.Millisecond,
			shouldError: true,
		},
		"delay shorter than timeout succeeds": {
			maxSize:     len(data),
			delay:       100 * time.Millisecond,
			readTimeout: 1 * time.Second,
			shouldError: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			conn := netmocks.NewMockConn(s.mockCtrl)

			var readDeadline time.Time
			conn.EXPECT().SetReadDeadline(gomock.Any()).DoAndReturn(func(t time.Time) error {
				readDeadline = t
				return nil
			})

			conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
				time.Sleep(c.delay)
				if !readDeadline.IsZero() && time.Now().After(readDeadline) {
					return 0, os.ErrDeadlineExceeded
				}
				n := copy(b, data)
				if n == len(data) {
					return n, io.EOF
				}
				return n, nil
			}).AnyTimes()

			readData, err := readFromConn(conn, c.maxSize, c.readTimeout, 12345)
			if c.shouldError {
				s.Error(err)
			} else {
				s.Require().NoError(err)
				s.Equal(data, readData)
			}
		})
	}
}

func (s *relayTestSuite) TestSemaphore() {
	vsockServer := &vsockServerImpl{
		semaphore:        semaphore.NewWeighted(1),
		semaphoreTimeout: 5 * time.Millisecond,
	}

	// First should succeed
	err := vsockServer.acquireSemaphore(s.ctx)
	s.Require().NoError(err)

	// Second should time out
	err = vsockServer.acquireSemaphore(s.ctx)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "failed to acquire semaphore")

	// After releasing once, a new acquire should succeed
	vsockServer.releaseSemaphore()
	err = vsockServer.acquireSemaphore(s.ctx)
	s.Require().NoError(err)
}

func (s *relayTestSuite) TestSendReportToSensor_HandlesContextCancellation() {
	client := sensormocks.NewMockVirtualMachineIndexReportServiceClient(s.mockCtrl)
	ctx, cancel := context.WithCancel(s.ctx)

	client.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, opts ...interface{}) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return &sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}).AnyTimes()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
	s.Require().Error(err)
	s.Contains(err.Error(), "context canceled")
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
			client := sensormocks.NewMockVirtualMachineIndexReportServiceClient(s.mockCtrl)

			var callCount int
			client.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, opts ...interface{}) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
					callCount++
					return &sensor.UpsertVirtualMachineIndexReportResponse{Success: c.respSuccess}, c.err
				}).AnyTimes()

			// The retry logic uses withExponentialBackoff, which currently has an initial delay between retries of
			// 100 ms, therefore after 500 ms the failing call has been retried already
			ctx, cancel := context.WithTimeout(s.ctx, 500*time.Millisecond)
			defer cancel()

			err := sendReportToSensor(ctx, &v1.IndexReport{}, client)
			s.Require().Error(err)

			retried := callCount > 1
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

// mockVsockServer is a test double for vsockServer that embeds real semaphore logic.
//
// We cannot use generated mocks (mockgen) for vsockServer because it has unexported methods,
// which cannot be implemented from outside the package due to Go's visibility rules.
// This custom mock tests the real semaphore integration while allowing precise control
// over I/O operations via channels.
type mockVsockServer struct {
	semaphore            *semaphore.Weighted
	maxSemaphoreWaitTime time.Duration
	acceptFunc           func() (net.Conn, error)
	startFunc            func() error
	stopFunc             func()
}

func (m *mockVsockServer) start() error {
	if m.startFunc != nil {
		return m.startFunc()
	}
	return nil
}

func (m *mockVsockServer) stop() {
	if m.stopFunc != nil {
		m.stopFunc()
	}
}

// acquireSemaphore uses REAL semaphore logic - this is what we're testing
func (m *mockVsockServer) acquireSemaphore(ctx context.Context) error {
	semCtx, cancel := context.WithTimeout(ctx, m.maxSemaphoreWaitTime)
	defer cancel()

	if err := m.semaphore.Acquire(semCtx, 1); err != nil {
		return err
	}
	return nil
}

// releaseSemaphore uses REAL semaphore logic - this is what we're testing
func (m *mockVsockServer) releaseSemaphore() {
	m.semaphore.Release(1)
}

func (m *mockVsockServer) accept() (net.Conn, error) {
	if m.acceptFunc != nil {
		return m.acceptFunc()
	}
	return nil, nil
}

func (s *relayTestSuite) TestRelaySemaphoreLimitsConcurrentConnections() {
	maxConcurrent := 2
	setup := s.setupSemaphoreTest(maxConcurrent)
	defer setup.cleanup()

	tracker := newConnectionTracker()

	// Setup accept function to create connections that track concurrency
	setup.vsockServer.acceptFunc = func() (net.Conn, error) {
		if setup.ctx.Err() != nil {
			return nil, setup.ctx.Err()
		}

		connIdx := tracker.incrementAttempts()
		return s.setupMockConnectionWithReadBehavior(uint32(100+connIdx), func(b []byte) (int, error) {
			tracker.started <- connIdx
			tracker.recordStart()

			allowedToFinish := <-tracker.canFinish
			tracker.recordEnd()

			report := &v1.IndexReport{VsockCid: strconv.FormatUint(uint64(100+allowedToFinish), 10)}
			return copy(b, s.marshalIndexReport(report)), io.EOF
		}), nil
	}

	setup.start()

	// Wait for maxConcurrent connections to start
	startedConns := s.waitForConnections(tracker.started, maxConcurrent)
	time.Sleep(200 * time.Millisecond) // Allow time for semaphore to block additional connections

	// Verify semaphore limits concurrent connections
	current, maxObserved := tracker.getCounts()
	s.Equal(maxConcurrent, current, "Should have exactly maxConcurrent connections running")
	s.LessOrEqual(maxObserved, maxConcurrent, "Semaphore should limit concurrent connections")

	// Release one connection and verify another can proceed
	tracker.canFinish <- startedConns[0]
	s.waitForConnections(tracker.started, 1)

	_, maxObserved = tracker.getCounts()
	s.LessOrEqual(maxObserved, maxConcurrent, "Semaphore should still limit concurrent connections")

	close(tracker.canFinish)
}

// waitForConnections waits for n connections to be signaled on the channel, returning their IDs.
func (s *relayTestSuite) waitForConnections(ch chan int, count int) []int {
	result := make([]int, 0, count)
	for i := 0; i < count; i++ {
		select {
		case connIdx := <-ch:
			result = append(result, connIdx)
		case <-time.After(1 * time.Second):
			s.Fail("Timed out waiting for connection")
		}
	}
	return result
}

func (s *relayTestSuite) TestRelaySemaphoreReleasedOnConnectionHandlingError() {
	setup := s.setupSemaphoreTest(1)
	defer setup.cleanup()

	tracker := newConnectionTracker()
	acceptSignal := make(chan struct{})

	// Setup accept to wait for signal, then create connection (first fails, second succeeds)
	setup.vsockServer.acceptFunc = func() (net.Conn, error) {
		select {
		case <-acceptSignal:
		case <-setup.ctx.Done():
			return nil, setup.ctx.Err()
		}

		connIdx := tracker.incrementAttempts()
		tracker.attempted <- struct{}{}

		conn := netmocks.NewMockConn(s.mockCtrl)
		conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: uint32(100 + connIdx)}).AnyTimes()

		conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
		conn.EXPECT().Close().Return(nil).AnyTimes()

		if connIdx == 1 {
			// First connection fails to read
			conn.EXPECT().Read(gomock.Any()).Return(0, io.ErrUnexpectedEOF)
		} else {
			// Second connection succeeds
			conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
				report := &v1.IndexReport{VsockCid: strconv.FormatUint(uint64(100+connIdx), 10)}
				return copy(b, s.marshalIndexReport(report)), io.EOF
			}).AnyTimes()
		}
		return conn, nil
	}

	setup.start()

	// First connection (will fail)
	acceptSignal <- struct{}{}
	s.waitForSignal(tracker.attempted, "first connection")

	// Second connection (should succeed, proving semaphore was released)
	acceptSignal <- struct{}{}
	s.waitForSignal(tracker.attempted, "second connection")

	s.GreaterOrEqual(tracker.getAttemptCount(), 2, "Semaphore should be released after connection error")
}

// waitForSignal waits for a signal on the channel with a timeout.
func (s *relayTestSuite) waitForSignal(ch chan struct{}, description string) {
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		s.Fail("Timed out waiting for " + description)
	}
}

func (s *relayTestSuite) TestRelaySemaphoreReleasedOnAcceptFailure() {
	setup := s.setupSemaphoreTest(1)
	defer setup.cleanup()

	setup.relay.waitAfterFailedAccept = 100 * time.Millisecond

	tracker := newConnectionTracker()
	acceptSignal := make(chan struct{})

	// Setup accept to fail first time, succeed second time
	setup.vsockServer.acceptFunc = func() (net.Conn, error) {
		select {
		case <-acceptSignal:
		case <-setup.ctx.Done():
			return nil, setup.ctx.Err()
		}

		connIdx := tracker.incrementAttempts()
		tracker.attempted <- struct{}{}

		if connIdx == 1 {
			return nil, errors.New("accept failed")
		}

		// Second accept succeeds
		return s.setupMockConnectionWithReadBehavior(42, func(b []byte) (int, error) {
			report := &v1.IndexReport{VsockCid: "42"}
			return copy(b, s.marshalIndexReport(report)), io.EOF
		}), nil
	}

	setup.start()

	// First accept (will fail)
	acceptSignal <- struct{}{}
	s.waitForSignal(tracker.attempted, "first accept")

	// Second accept (should succeed, proving semaphore was released)
	acceptSignal <- struct{}{}
	s.waitForSignal(tracker.attempted, "second accept")

	s.GreaterOrEqual(tracker.getAttemptCount(), 2, "Semaphore should be released after accept failure")
}

// Helper functions for tests

// newMockVsockServer creates a mock vsock server with the specified max concurrent connections.
func (s *relayTestSuite) newMockVsockServer(maxConcurrent int) *mockVsockServer {
	return &mockVsockServer{
		semaphore:            semaphore.NewWeighted(int64(maxConcurrent)),
		maxSemaphoreWaitTime: 50 * time.Millisecond,
	}
}

// marshalIndexReport marshals an index report to bytes, failing the test if marshaling fails.
func (s *relayTestSuite) marshalIndexReport(report *v1.IndexReport) []byte {
	data, err := proto.Marshal(report)
	if err != nil {
		s.T().Fatalf("Failed to marshal index report in test setup: %v", err)
	}
	return data
}

// setupMockConnectionWithReadBehavior creates a mock connection with a custom read behavior.
// This helper reduces duplication in tests that need to customize how the connection responds to reads.
func (s *relayTestSuite) setupMockConnectionWithReadBehavior(vsockCID uint32, readFunc func([]byte) (int, error)) net.Conn {
	conn := netmocks.NewMockConn(s.mockCtrl)
	conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: vsockCID}).AnyTimes()
	conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
	conn.EXPECT().Read(gomock.Any()).DoAndReturn(readFunc).AnyTimes()
	conn.EXPECT().Close().Return(nil).AnyTimes()
	return conn
}

// semaphoreTestSetup contains common infrastructure for semaphore integration tests.
type semaphoreTestSetup struct {
	ctx         context.Context
	cancel      context.CancelFunc
	vsockServer *mockVsockServer
	mockClient  *sensormocks.MockVirtualMachineIndexReportServiceClient
	relay       *Relay
	wg          *sync.WaitGroup
}

// setupSemaphoreTest creates common test infrastructure for semaphore tests.
func (s *relayTestSuite) setupSemaphoreTest(maxConcurrent int) *semaphoreTestSetup {
	ctx, cancel := context.WithCancel(s.ctx)
	vsockServer := s.newMockVsockServer(maxConcurrent)

	mockClient := sensormocks.NewMockVirtualMachineIndexReportServiceClient(s.mockCtrl)
	mockClient.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil).AnyTimes()

	relay := &Relay{
		connectionReadTimeout: 100 * time.Millisecond,
		ctx:                   ctx,
		sensorClient:          mockClient,
		vsockServer:           vsockServer,
		waitAfterFailedAccept: 1 * time.Millisecond,
	}

	return &semaphoreTestSetup{
		ctx:         ctx,
		cancel:      cancel,
		vsockServer: vsockServer,
		mockClient:  mockClient,
		relay:       relay,
		wg:          &sync.WaitGroup{},
	}
}

// start begins running the relay in a background goroutine.
func (setup *semaphoreTestSetup) start() {
	setup.wg.Add(1)
	go func() {
		defer setup.wg.Done()
		_ = setup.relay.Run()
	}()
}

// cleanup cancels the context and waits for the relay to stop.
func (setup *semaphoreTestSetup) cleanup() {
	setup.cancel()
	setup.wg.Wait()
}

// connectionTracker helps track connection attempts and control their lifecycle.
type connectionTracker struct {
	started       chan int
	canFinish     chan int
	attempted     chan struct{}
	count         int
	mu            sync.Mutex
	maxObserved   int
	currentActive int
}

func newConnectionTracker() *connectionTracker {
	return &connectionTracker{
		started:   make(chan int, 10),
		canFinish: make(chan int, 10),
		attempted: make(chan struct{}, 10),
	}
}

func (ct *connectionTracker) recordStart() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.currentActive++
	if ct.currentActive > ct.maxObserved {
		ct.maxObserved = ct.currentActive
	}
}

func (ct *connectionTracker) recordEnd() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.currentActive--
}

func (ct *connectionTracker) getCounts() (current, maxObserved int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.currentActive, ct.maxObserved
}

func (ct *connectionTracker) incrementAttempts() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.count++
	return ct.count
}

func (ct *connectionTracker) getAttemptCount() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.count
}
