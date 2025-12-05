package relay

import (
	"context"
	"errors"
	"testing"
	"time"

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
	relay := New(stream, sender)

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

	// Create mock stream that produces test reports
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"},
		},
	}

	// Create relay with mock dependencies using the public constructor
	relay := New(mockIndexReportStream, mockIndexReportSender)

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

	// Verify all reports were sent
	mockIndexReportSender.mu.Lock()
	s.Require().Len(mockIndexReportSender.sentReports, 2)
	s.Equal("100", mockIndexReportSender.sentReports[0].GetVsockCid())
	s.Equal("200", mockIndexReportSender.sentReports[1].GetVsockCid())
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
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"},
			{VsockCid: "300"},
		},
	}

	relay := New(mockIndexReportStream, mockIndexReportSender)

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

	// All three reports should have been attempted
	mockIndexReportSender.mu.Lock()
	s.Require().Len(mockIndexReportSender.sentReports, 3)
	mockIndexReportSender.mu.Unlock()

	err := <-errChan
	s.ErrorIs(err, context.Canceled)
}

// TestRelay_ContextCancellation verifies relay stops on context cancellation
func (s *relayTestSuite) TestRelay_ContextCancellation() {
	// The mocked stream signals when first report is sent
	started := concurrency.NewSignal()
	mockIndexReportStream := &mockIndexReportStream{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"}, // Second report will never be processed
		},
		started: &started,
	}

	mockIndexReportSender := &mockIndexReportSender{
		failOnIndex: -1, // never fail
	}

	relay := New(mockIndexReportStream, mockIndexReportSender)

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

// Mock implementations

// failingIndexReportStream is a mock IndexReportStream whose Start method
// always fails. It tracks how many times Start is called so tests can assert
// correct behavior.
type failingIndexReportStream struct {
	startCalled int
}

// Start implements IndexReportStream.Start. It always returns a nil channel
// and errTestStreamStart to simulate a stream startup failure.
func (f *failingIndexReportStream) Start(ctx context.Context) (<-chan *v1.IndexReport, error) {
	f.startCalled++
	return nil, errTestStreamStart
}

type mockIndexReportStream struct {
	reports []*v1.IndexReport
	started *concurrency.Signal // signals when first report is streamed
}

func (m *mockIndexReportStream) Start(ctx context.Context) (<-chan *v1.IndexReport, error) {
	reportChan := make(chan *v1.IndexReport, len(m.reports))

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
	sentReports   []*v1.IndexReport
	failOnIndex   int                 // Index to fail on (0-based), use -1 to never fail
	done          *concurrency.Signal // signals when expectedCount reports are sent
	expectedCount int                 // number of reports expected before signaling done
}

func (m *mockIndexReportSender) Send(_ context.Context, report *v1.IndexReport) error {
	m.mu.Lock()
	currentIndex := len(m.sentReports)
	m.sentReports = append(m.sentReports, report)

	// Signal done when we've sent expected count
	if m.done != nil && len(m.sentReports) == m.expectedCount {
		m.done.Signal()
	}
	m.mu.Unlock()

	// Fail on the specified index
	if currentIndex == m.failOnIndex {
		return errTest
	}
	return nil
}
