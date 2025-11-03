package relay

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type mockSensorClient struct {
	capturedRequests []*sensor.UpsertVirtualMachineIndexReportRequest
	delay            time.Duration
	err              error
	response         *sensor.UpsertVirtualMachineIndexReportResponse
}

func newMockSensorClient() *mockSensorClient {
	return &mockSensorClient{
		response: &sensor.UpsertVirtualMachineIndexReportResponse{Success: true},
	}
}

func (c *mockSensorClient) UpsertVirtualMachineIndexReport(ctx context.Context, req *sensor.UpsertVirtualMachineIndexReportRequest, _ ...grpc.CallOption) (*sensor.UpsertVirtualMachineIndexReportResponse, error) {
	select {
	case <-time.After(c.delay):
		c.capturedRequests = append(c.capturedRequests, req)
		return c.response, c.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *mockSensorClient) withDelay(delay time.Duration) *mockSensorClient {
	c.delay = delay
	return c
}

func (c *mockSensorClient) withError(err error) *mockSensorClient {
	c.err = err
	return c
}

func (c *mockSensorClient) withUnsuccessfulResponse() *mockSensorClient {
	c.response = &sensor.UpsertVirtualMachineIndexReportResponse{Success: false}
	return c
}

// mockVsockServer implements vsock.Server interface for testing concurrent connection limiting
type mockVsockServer struct {
	// Control channels for test orchestration
	acceptChan        chan net.Conn // Send connections to accept
	acceptErrChan     chan error    // Send errors from Accept()
	acquireResultChan chan error    // Control semaphore acquisition results
	releaseCallChan   chan struct{} // Track Release() calls

	// State tracking
	semaphoreAcquired int
	maxConcurrent     int

	// Synchronization
	mu sync.Mutex
}

func newMockVsockServer(maxConcurrent int) *mockVsockServer {
	return &mockVsockServer{
		acceptChan:        make(chan net.Conn, 10),
		acceptErrChan:     make(chan error, 1),
		acquireResultChan: make(chan error, 10),
		releaseCallChan:   make(chan struct{}, 10),
		maxConcurrent:     maxConcurrent,
	}
}

func (m *mockVsockServer) Accept() (net.Conn, error) {
	select {
	case conn := <-m.acceptChan:
		return conn, nil
	case err := <-m.acceptErrChan:
		return nil, err
	}
}

func (m *mockVsockServer) AcquireSemaphore(ctx context.Context) error {
	select {
	case err := <-m.acquireResultChan:
		if err == nil {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.semaphoreAcquired++
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mockVsockServer) ReleaseSemaphore() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.semaphoreAcquired--
	m.releaseCallChan <- struct{}{}
}

func (m *mockVsockServer) Start() error {
	return nil
}

func (m *mockVsockServer) Stop() {
	close(m.acceptChan)
	close(m.acceptErrChan)
}

// getCurrentAcquired returns the current number of acquired semaphore slots
func (m *mockVsockServer) getCurrentAcquired() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.semaphoreAcquired
}

// mockVsockConn is a mock implementation of net.Conn for testing
type mockVsockConn struct {
	closed       bool
	data         []byte
	readStarted  chan struct{} // Signals when Read() is called
	readReady    chan struct{} // Blocks Read() until signaled
	remoteAddr   net.Addr
	readDeadline time.Time
	closedMu     sync.Mutex
}

func newMockVsockConn(vsockCID uint32, data []byte) *mockVsockConn {
	return &mockVsockConn{
		data:        data,
		readStarted: make(chan struct{}),
		readReady:   make(chan struct{}),
		remoteAddr:  &vsock.Addr{ContextID: vsockCID},
	}
}

func (c *mockVsockConn) withImmediateRead() *mockVsockConn {
	// For immediate reads, close the readReady channel so Read() doesn't block
	close(c.readReady)
	return c
}

func (c *mockVsockConn) signalReadReady() {
	select {
	case <-c.readReady:
		// Already closed
	default:
		close(c.readReady)
	}
}

func (c *mockVsockConn) Read(b []byte) (n int, err error) {
	// Signal that Read() has been called
	select {
	case <-c.readStarted:
		// Already signaled
	default:
		close(c.readStarted)
	}

	// Wait until we're signaled to proceed or deadline expires
	select {
	case <-c.readReady:
		// Proceed with read
	case <-time.After(time.Until(c.readDeadline)):
		if !c.readDeadline.IsZero() {
			return 0, os.ErrDeadlineExceeded
		}
	}

	n = copy(b, c.data)
	if n == len(c.data) {
		return n, io.EOF
	}
	return n, nil
}

func (c *mockVsockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *mockVsockConn) Close() error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	c.closed = true
	return nil
}

func (c *mockVsockConn) IsClosed() bool {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()
	return c.closed
}

func (c *mockVsockConn) Write([]byte) (int, error)   { return 0, nil }
func (c *mockVsockConn) LocalAddr() net.Addr         { return nil }
func (c *mockVsockConn) SetDeadline(time.Time) error { return nil }
func (c *mockVsockConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = t
	return nil
}
func (c *mockVsockConn) SetWriteDeadline(time.Time) error { return nil }

// createMockVsockConnection creates a mock connection that returns a valid index report
// The connection will block in Read() until signalReadReady() is called
func createMockVsockConnection(vsockCID uint32) *mockVsockConn {
	indexReport := &v1.IndexReport{
		VsockCid: strconv.FormatUint(uint64(vsockCID), 10),
	}
	data, _ := proto.Marshal(indexReport)
	return newMockVsockConn(vsockCID, data)
}

// createMockVsockConnectionImmediate creates a mock connection that reads immediately without blocking
func createMockVsockConnectionImmediate(vsockCID uint32) *mockVsockConn {
	return createMockVsockConnection(vsockCID).withImmediateRead()
}

// connectionCloseTracker wraps a connection to track Close() calls
type connectionCloseTracker struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
}

func newConnectionCloseTracker(conn net.Conn) *connectionCloseTracker {
	return &connectionCloseTracker{
		Conn:   conn,
		closed: make(chan struct{}),
	}
}

func (c *connectionCloseTracker) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return c.Conn.Close()
}
