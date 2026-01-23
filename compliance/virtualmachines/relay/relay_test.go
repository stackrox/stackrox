package relay

import (
	"context"
	"errors"
	"testing"
	"time"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
)

func TestRelay(t *testing.T) {
	suite.Run(t, new(relayTestSuite))
}

type relayTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *relayTestSuite) SetupTest() {
	s.ctx = context.Background()
}

// errTestStreamStart is returned by the mock stream's Start method to simulate
// a startup failure.
var errTestStreamStart = errors.New("test stream start failure")

// errTest is a generic test error for use in mock implementations.
var errTest = errors.New("test error")

// TestRelay_StartFailure verifies that Relay.Run propagates stream startup
// errors and does not enter the main select loop when initialization fails.
func (s *relayTestSuite) TestRelay_StartFailure() {
	// Use a bounded context to ensure the test fails if Relay.Run blocks.
	ctx, cancel := context.WithTimeout(s.ctx, 100*time.Millisecond)
	defer cancel()

	// Create a stream that fails immediately on Start.
	stream := &failingIndexReportStream{}

	// Create a dummy sender. It should never be used in this test because the
	// relay is expected to fail before entering its main loop.
	sender := &mockIndexReportSender{
		failOnIndex:   -1,
		expectedCount: 0,
	}

	// Construct the relay under test.
	relay := New(stream, sender, &mockUMH{}, 0, 4*time.Hour)

	// Run the relay in a goroutine so we can assert it returns promptly and
	// does not block in its select loop.
	errCh := make(chan error, 1)
	go func() {
		errCh <- relay.Run(ctx)
	}()

	select {
	case err := <-errCh:
		// Relay.Run should surface the stream startup error (possibly wrapped).
		s.Require().Error(err, "Relay.Run should return an error when stream Start fails")
		s.Require().ErrorIs(err, errTestStreamStart, "Relay.Run should wrap the stream startup error")
		s.Equal(1, stream.startCalled, "stream.Start should be called exactly once")
	case <-time.After(100 * time.Millisecond):
		s.Fail("Relay.Run did not return promptly on stream Start failure (likely entered select loop)")
	}
}

// TestRelay_Integration tests the interaction between stream, relay, and sender.
// This uses mock implementations to verify the full data flow without real vsock/sensor.
func (s *relayTestSuite) TestRelay_Integration() {
	// Create mock sender that signals when reports are received
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1, // never fail
		done:          &done,
		expectedCount: 2,
	}

	// Create mock stream that produces test messages with discovered data
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			{
				IndexReport: &v1.IndexReport{VsockCid: "100"},
				DiscoveredData: &v1.DiscoveredData{
					DetectedOs:        v1.DetectedOS_RHEL,
					OsVersion:         "9.3",
					ActivationStatus:  v1.ActivationStatus_ACTIVE,
					DnfMetadataStatus: v1.DnfMetadataStatus_AVAILABLE,
				},
			},
			{
				IndexReport: &v1.IndexReport{VsockCid: "200"},
				DiscoveredData: &v1.DiscoveredData{
					DetectedOs:        v1.DetectedOS_UNKNOWN,
					OsVersion:         "unknown",
					ActivationStatus:  v1.ActivationStatus_INACTIVE,
					DnfMetadataStatus: v1.DnfMetadataStatus_UNAVAILABLE,
				},
			},
		},
	}

	// Create relay with mock dependencies using the public constructor
	// Rate limiting disabled (0)
	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 0, 4*time.Hour)

	// Run relay in background
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be processed (or timeout)
	select {
	case <-done.Done():
		// All reports processed
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// Verify all messages were sent with discovered data preserved
	mockIndexReportSender.mu.Lock()
	s.Require().Len(mockIndexReportSender.sentMessages, 2)

	first := mockIndexReportSender.sentMessages[0]
	second := mockIndexReportSender.sentMessages[1]

	// IndexReport fields are preserved
	s.Equal("100", first.GetIndexReport().GetVsockCid())
	s.Equal("200", second.GetIndexReport().GetVsockCid())

	// VM discovered data is preserved
	s.Equal(v1.DetectedOS_RHEL, first.GetDiscoveredData().GetDetectedOs())
	s.Equal("9.3", first.GetDiscoveredData().GetOsVersion())
	s.Equal(v1.ActivationStatus_ACTIVE, first.GetDiscoveredData().GetActivationStatus())
	s.Equal(v1.DnfMetadataStatus_AVAILABLE, first.GetDiscoveredData().GetDnfMetadataStatus())

	s.Equal(v1.DetectedOS_UNKNOWN, second.GetDiscoveredData().GetDetectedOs())
	s.Equal("unknown", second.GetDiscoveredData().GetOsVersion())
	s.Equal(v1.ActivationStatus_INACTIVE, second.GetDiscoveredData().GetActivationStatus())
	s.Equal(v1.DnfMetadataStatus_UNAVAILABLE, second.GetDiscoveredData().GetDnfMetadataStatus())

	mockIndexReportSender.mu.Unlock()

	// Verify relay exited cleanly
	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_SenderErrorsDoNotStopProcessing verifies that sender errors don't halt the relay
func (s *relayTestSuite) TestRelay_SenderErrorsDoNotStopProcessing() {
	// Sender fails on second report but signals completion
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   1, // fail on second report
		done:          &done,
		expectedCount: 3,
	}

	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
			relaytest.NewTestVMReport("200"),
			relaytest.NewTestVMReport("300"),
		},
	}

	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 0, 4*time.Hour)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be attempted
	select {
	case <-done.Done():
		// All reports attempted
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// All three messages should have been attempted
	mockIndexReportSender.mu.Lock()
	s.Require().Len(mockIndexReportSender.sentMessages, 3)
	mockIndexReportSender.mu.Unlock()

	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_ContextCancellation verifies relay stops on context cancellation
func (s *relayTestSuite) TestRelay_ContextCancellation() {
	// The mocked stream signals when first message is sent
	started := concurrency.NewSignal()
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.VMReport{
			relaytest.NewTestVMReport("100"),
			relaytest.NewTestVMReport("200"), // Second message will never be processed
		},
		started: &started,
	}

	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex: -1, // never fail
	}

	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 0, 4*time.Hour)

	ctx, cancel := context.WithCancel(s.ctx)

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for stream to start sending reports
	<-started.Done()

	// Cancel immediately
	cancel()

	// Should exit quickly
	select {
	case err := <-errChan:
		s.ErrorIs(err, context.Canceled)
	case <-time.After(100 * time.Millisecond):
		s.Fail("Relay did not exit after context cancellation")
	}
}

// TestRelay_RateLimiting verifies that rate limiting works
func (s *relayTestSuite) TestRelay_RateLimiting() {
	// Create mock sender
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1, // never fail
		done:          &done,
		expectedCount: 1, // only first should pass
	}

	// Send 3 reports from same VSOCK ID
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "100"}, // Same ID - should be rate limited
			{VsockCid: "100"}, // Same ID - should be rate limited
		},
	}

	// Rate limit: 1 per minute (effectively blocks after first)
	relay := New(mockIndexReportStream, mockIndexReportSender, &mockUMH{}, 1, 4*time.Hour)

	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for first report
	select {
	case <-done.Done():
		// First report processed
	case <-time.After(500 * time.Millisecond):
		s.Fail("Timeout waiting for first report")
	}

	// Give time for other reports to be processed (they should be rate limited)
	time.Sleep(100 * time.Millisecond)

	cancel()

	// Only 1 report should have been sent (others rate limited)
	mockIndexReportSender.mu.Lock()
	s.Len(mockIndexReportSender.sentReports, 1)
	mockIndexReportSender.mu.Unlock()

	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_UMHInteraction verifies the relay observes UMH signals and marks ACKs.
func (s *relayTestSuite) TestRelay_UMHInteraction() {
	done := concurrency.NewSignal()
	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex:   -1,
		done:          &done,
		expectedCount: 1,
	}

	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
		},
	}

	umh := &mockUMH{
		retryCh: make(chan string, 1),
	}

	relay := New(mockIndexReportStream, mockIndexReportSender, umh, 0, 4*time.Hour)

	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for the report to be sent.
	select {
	case <-done.Done():
	case <-time.After(time.Second):
		s.Fail("timeout waiting for report send")
	}

	// Verify ObserveSending recorded the VSOCK ID.
	umh.mu.Lock()
	s.Contains(umh.sends, "100")
	umh.mu.Unlock()

	// Send an ACK via UMH and ensure relay records it in cache.
	umh.HandleACK("100")
	time.Sleep(50 * time.Millisecond)

	relay.mu.Lock()
	cached := relay.cache["100"]
	relay.mu.Unlock()
	s.Require().NotNil(cached, "cached report should exist after ACK")
	s.False(cached.lastAckedAt.IsZero(), "lastAckedAt should be recorded")

	cancel()
	<-errChan
}

// Mock implementations

// failingIndexReportStream is a mock IndexReportStream whose Start method
// always fails. It tracks how many times Start is called so tests can assert
// correct behavior.
type failingIndexReportStream struct {
	startCalled int
}

// Start implements IndexReportStream.Start. It always returns a nil channel
// and errTestStreamStart to simulate a stream startup failure.
func (f *failingIndexReportStream) Start(ctx context.Context) (<-chan *v1.VMReport, error) {
	f.startCalled++
	return nil, errTestStreamStart
}

type mockIndexReportStream struct {
	reports []*v1.VMReport
	started *concurrency.Signal // signals when first report is streamed
}

func (m *mockIndexReportStream) Start(ctx context.Context) (<-chan *v1.VMReport, error) {
	reportChan := make(chan *v1.VMReport, len(m.reports))

	go func() {
		for i, report := range m.reports {
			select {
			case <-ctx.Done():
				return
			case reportChan <- report:
				// Signal when first report is streamed
				if i == 0 && m.started != nil {
					m.started.Signal()
				}
			}
		}
	}()

	return reportChan, nil
}

type mockIndexReportSender struct {
	mu            sync.Mutex
	sentMessages  []*v1.VMReport
	failOnIndex   int                 // Index to fail on (0-based), use -1 to never fail
	done          *concurrency.Signal // signals when expectedCount reports are sent
	expectedCount int                 // number of reports expected before signaling done
}

func (m *mockIndexReportSender) Send(_ context.Context, vmReport *v1.VMReport) error {
	m.mu.Lock()
	currentIndex := len(m.sentMessages)
	m.sentMessages = append(m.sentMessages, vmReport)

	// Signal done when we've sent expected count
	if m.done != nil && len(m.sentMessages) == m.expectedCount {
		m.done.Signal()
	}
	m.mu.Unlock()

	// Fail on the specified index
	if currentIndex == m.failOnIndex {
		return errTest
	}
	return nil
}

// mockUMH is a mock UnconfirmedMessageHandler for testing
type mockUMH struct {
	mu      sync.Mutex
	acks    []string
	nacks   []string
	sends   []string
	retryCh chan string
	onACKCb func(resourceID string)
}

func (m *mockUMH) HandleACK(resourceID string) {
	m.mu.Lock()
	m.acks = append(m.acks, resourceID)
	cb := m.onACKCb
	m.mu.Unlock()
	if cb != nil {
		cb(resourceID)
	}
}

func (m *mockUMH) HandleNACK(resourceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nacks = append(m.nacks, resourceID)
}

func (m *mockUMH) ObserveSending(resourceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, resourceID)
}

func (m *mockUMH) RetryCommand() <-chan string {
	if m.retryCh == nil {
		m.retryCh = make(chan string)
	}
	return m.retryCh
}

func (m *mockUMH) OnACK(callback func(resourceID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onACKCb = callback
}
