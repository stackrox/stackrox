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
	"sync"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	sensormocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/github.com/stackrox/rox/generated/internalapi/sensor/mocks"
	netmocks "github.com/stackrox/rox/compliance/virtualmachines/relay/mocks/net/mocks"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/errox"
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
}

func (s *relayTestSuite) SetupSubTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *relayTestSuite) TearDownSubTest() {
	s.mockCtrl.Finish()
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

			readData, err := readFromConn(conn, c.maxSize, c.readTimeout)
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
		semaphore:            semaphore.NewWeighted(1),
		maxSemaphoreWaitTime: 5 * time.Millisecond,
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
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	maxConcurrent := 2
	mockVsock := &mockVsockServer{
		semaphore:            semaphore.NewWeighted(int64(maxConcurrent)),
		maxSemaphoreWaitTime: 50 * time.Millisecond,
	}

	// Control when accept() returns a connection
	acceptSignal := make(chan struct{})
	// Track concurrent connections
	connectionStarted := make(chan struct{}, 10)
	connectionDone := make(chan struct{}, 10)

	// Track the maximum concurrent connections observed
	var maxObservedConcurrent int
	var currentConcurrent int
	var mu sync.Mutex

	connIndex := 0
	mockVsock.acceptFunc = func() (net.Conn, error) {
		// Block until we're told to accept a connection or context is canceled
		select {
		case <-acceptSignal:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		connIndex++
		currentIndex := connIndex

		conn := netmocks.NewMockConn(mockCtrl)
		conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: uint32(100 + currentIndex)}).AnyTimes()
		conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
			// Signal that this connection has started being handled
			connectionStarted <- struct{}{}

			mu.Lock()
			currentConcurrent++
			if currentConcurrent > maxObservedConcurrent {
				maxObservedConcurrent = currentConcurrent
			}
			mu.Unlock()

			// Wait for signal to complete
			<-connectionDone

			mu.Lock()
			currentConcurrent--
			mu.Unlock()

			// Return valid index report data
			report := &v1.IndexReport{VsockCid: strconv.FormatUint(uint64(100+currentIndex), 10)}
			data, _ := proto.Marshal(report)
			n := copy(b, data)
			return n, io.EOF
		}).AnyTimes()
		conn.EXPECT().Close().Return(nil).AnyTimes()

		return conn, nil
	}

	mockClient := sensormocks.NewMockVirtualMachineIndexReportServiceClient(mockCtrl)
	mockClient.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil).AnyTimes()

	relay := &Relay{
		connectionReadTimeout: 1 * time.Second,
		ctx:                   ctx,
		sensorClient:          mockClient,
		vsockServer:           mockVsock,
		waitAfterFailedAccept: 1 * time.Millisecond,
	}

	// Start relay in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = relay.Run()
	}()

	// Signal maxConcurrent connections to be accepted
	for i := 0; i < maxConcurrent; i++ {
		acceptSignal <- struct{}{}
	}

	// Wait for them to start being handled
	for i := 0; i < maxConcurrent; i++ {
		select {
		case <-connectionStarted:
		case <-time.After(1 * time.Second):
			s.Fail("Timed out waiting for connection to start")
		}
	}

	// Now try to accept one more connection - it should be blocked by the semaphore
	go func() {
		acceptSignal <- struct{}{}
	}()
	time.Sleep(100 * time.Millisecond) // Give some time for the third connection to try to acquire semaphore

	// Verify that we haven't exceeded maxConcurrent (the third hasn't started yet)
	mu.Lock()
	current := currentConcurrent
	max := maxObservedConcurrent
	mu.Unlock()
	s.Equal(maxConcurrent, current, "Should have exactly maxConcurrent connections running")
	s.LessOrEqual(max, maxConcurrent, "Semaphore should limit concurrent connections to %d", maxConcurrent)

	// Now release one connection to allow the third to proceed
	connectionDone <- struct{}{}

	// Wait for the third connection to start
	select {
	case <-connectionStarted:
	case <-time.After(1 * time.Second):
		s.Fail("Timed out waiting for next connection after releasing one")
	}

	// Verify that we still haven't exceeded maxConcurrent
	mu.Lock()
	max = maxObservedConcurrent
	mu.Unlock()
	s.LessOrEqual(max, maxConcurrent, "Semaphore should limit concurrent connections to %d even after releasing one", maxConcurrent)

	// Clean up - release remaining connections and cancel context
	close(connectionDone)
	cancel()
	wg.Wait()
}

func (s *relayTestSuite) TestRelaySemaphoreReleasedOnConnectionHandlingError() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	maxConcurrent := 1
	mockVsock := &mockVsockServer{
		semaphore:            semaphore.NewWeighted(int64(maxConcurrent)),
		maxSemaphoreWaitTime: 50 * time.Millisecond,
	}

	acceptSignal := make(chan struct{})
	connectionAttempts := 0
	var attemptsMu sync.Mutex

	mockVsock.acceptFunc = func() (net.Conn, error) {
		select {
		case <-acceptSignal:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		attemptsMu.Lock()
		connectionAttempts++
		currentAttempt := connectionAttempts
		attemptsMu.Unlock()

		conn := netmocks.NewMockConn(mockCtrl)
		conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: uint32(100 + currentAttempt)}).AnyTimes()

		if currentAttempt == 1 {
			// First connection will fail to read (simulating an error during handling)
			conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil)
			conn.EXPECT().Read(gomock.Any()).Return(0, io.ErrUnexpectedEOF)
			conn.EXPECT().Close().Return(nil)
		} else {
			// Second connection should succeed, proving the semaphore was released
			conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
			conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
				report := &v1.IndexReport{VsockCid: strconv.FormatUint(uint64(100+currentAttempt), 10)}
				data, _ := proto.Marshal(report)
				n := copy(b, data)
				return n, io.EOF
			}).AnyTimes()
			conn.EXPECT().Close().Return(nil)
		}

		return conn, nil
	}

	mockClient := sensormocks.NewMockVirtualMachineIndexReportServiceClient(mockCtrl)
	mockClient.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil).AnyTimes()

	relay := &Relay{
		connectionReadTimeout: 100 * time.Millisecond,
		ctx:                   ctx,
		sensorClient:          mockClient,
		vsockServer:           mockVsock,
		waitAfterFailedAccept: 1 * time.Millisecond,
	}

	// Start relay in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = relay.Run()
	}()

	// Signal first connection to be accepted (which will error during handling)
	acceptSignal <- struct{}{}
	time.Sleep(200 * time.Millisecond) // Wait for it to fail

	// Signal second connection to be accepted (which should succeed, proving semaphore was released)
	acceptSignal <- struct{}{}
	time.Sleep(200 * time.Millisecond) // Wait for it to complete

	// Verify that both connections were attempted, proving the semaphore was released after the first error
	attemptsMu.Lock()
	attempts := connectionAttempts
	attemptsMu.Unlock()
	s.GreaterOrEqual(attempts, 2, "Second connection should have been attempted after first failed, proving semaphore was released")

	cancel()
	wg.Wait()
}

func (s *relayTestSuite) TestRelaySemaphoreReleasedOnAcceptFailure() {
	mockCtrl := gomock.NewController(s.T())
	defer mockCtrl.Finish()

	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	maxConcurrent := 1
	mockVsock := &mockVsockServer{
		semaphore:            semaphore.NewWeighted(int64(maxConcurrent)),
		maxSemaphoreWaitTime: 50 * time.Millisecond,
	}

	acceptSignal := make(chan struct{})
	acceptAttempts := 0
	var attemptsMu sync.Mutex

	mockVsock.acceptFunc = func() (net.Conn, error) {
		select {
		case <-acceptSignal:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		attemptsMu.Lock()
		acceptAttempts++
		currentAttempt := acceptAttempts
		attemptsMu.Unlock()

		if currentAttempt == 1 {
			// First accept fails
			return nil, errors.New("accept failed")
		}

		// Second accept succeeds, proving semaphore was released
		conn := netmocks.NewMockConn(mockCtrl)
		conn.EXPECT().RemoteAddr().Return(&vsock.Addr{ContextID: 42}).AnyTimes()
		conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
			report := &v1.IndexReport{VsockCid: "42"}
			data, _ := proto.Marshal(report)
			n := copy(b, data)
			return n, io.EOF
		}).AnyTimes()
		conn.EXPECT().Close().Return(nil)

		return conn, nil
	}

	mockClient := sensormocks.NewMockVirtualMachineIndexReportServiceClient(mockCtrl)
	mockClient.EXPECT().UpsertVirtualMachineIndexReport(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&sensor.UpsertVirtualMachineIndexReportResponse{Success: true}, nil).AnyTimes()

	relay := &Relay{
		connectionReadTimeout: 100 * time.Millisecond,
		ctx:                   ctx,
		sensorClient:          mockClient,
		vsockServer:           mockVsock,
		waitAfterFailedAccept: 100 * time.Millisecond, // Increased to avoid tight loops
	}

	// Start relay in background
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = relay.Run()
	}()

	// Signal first accept (which will fail)
	acceptSignal <- struct{}{}
	time.Sleep(150 * time.Millisecond) // Wait for it to process and wait

	// Signal second accept (which should succeed, proving semaphore was released)
	acceptSignal <- struct{}{}
	time.Sleep(200 * time.Millisecond) // Wait for it to complete

	// Verify that both accepts were attempted, proving the semaphore was released after the first failed
	attemptsMu.Lock()
	attempts := acceptAttempts
	attemptsMu.Unlock()
	s.GreaterOrEqual(attempts, 2, "Second accept should have been attempted after first failed, proving semaphore was released")

	cancel()
	wg.Wait()
}
