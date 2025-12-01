package relay

import (
	"context"
	"sync"
	"testing"
	"time"

	relaytest "github.com/stackrox/rox/compliance/virtualmachines/relay/testutils"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
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

// TestRelay_Integration tests the interaction between provider, relay, and sender.
// This uses mock implementations to verify the full data flow without real vsock/sensor.
func (s *relayTestSuite) TestRelay_Integration() {
	// Create mock sender that signals when reports are received
	mockReportSender := &mockReportSender{
		failOnIndex: -1, // never fail
		doneChan:    make(chan struct{}),
		expectedCount: 2,
	}

	// Create mock provider that produces test reports
	mockReportProvider := &mockReportProvider{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"},
		},
	}

	// Create relay with mock dependencies using the public constructor
	relay := NewRelay(mockReportProvider, mockReportSender)

	// Run relay in background
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be processed (or timeout)
	select {
	case <-mockReportSender.doneChan:
		// All reports processed
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// Verify all reports were sent
	mockReportSender.mu.Lock()
	s.Require().Len(mockReportSender.sentReports, 2)
	s.Equal("100", mockReportSender.sentReports[0].VsockCid)
	s.Equal("200", mockReportSender.sentReports[1].VsockCid)
	mockReportSender.mu.Unlock()

	// Verify relay exited cleanly
	err := <-errChan
	s.Equal(context.Canceled, err)
}

// TestRelay_SenderErrorsDoNotStopProcessing verifies that sender errors don't halt the relay
func (s *relayTestSuite) TestRelay_SenderErrorsDoNotStopProcessing() {
	// Sender fails on second report but signals completion
	mockReportSender := &mockReportSender{
		failOnIndex:   1, // fail on second report
		doneChan:      make(chan struct{}),
		expectedCount: 3,
	}

	mockReportProvider := &mockReportProvider{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"},
			{VsockCid: "300"},
		},
	}

	relay := NewRelay(mockReportProvider, mockReportSender)

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for all reports to be attempted
	select {
	case <-mockReportSender.doneChan:
		// All reports attempted
	case <-time.After(1 * time.Second):
		s.Fail("Timeout waiting for reports to be processed")
	}

	cancel()

	// All three reports should have been attempted
	mockReportSender.mu.Lock()
	s.Require().Len(mockReportSender.sentReports, 3)
	mockReportSender.mu.Unlock()

	err := <-errChan
	s.Equal(context.Canceled, err)
}

// TestRelay_ContextCancellation verifies relay stops on context cancellation
func (s *relayTestSuite) TestRelay_ContextCancellation() {
	// Provider signals when first report is sent
	startedChan := make(chan struct{})
	mockReportProvider := &mockReportProvider{
		reports: []*v1.IndexReport{
			{VsockCid: "100"},
			{VsockCid: "200"}, // Second report will never be processed
		},
		startedChan: startedChan,
	}

	mockReportSender := &mockReportSender{
		failOnIndex: -1, // never fail
	}

	relay := NewRelay(mockReportProvider, mockReportSender)

	ctx, cancel := context.WithCancel(s.ctx)

	errChan := make(chan error, 1)
	go func() {
		errChan <- relay.Run(ctx)
	}()

	// Wait for provider to start sending reports
	<-startedChan

	// Cancel immediately
	cancel()

	// Should exit quickly
	select {
	case err := <-errChan:
		s.Equal(context.Canceled, err)
	case <-time.After(100 * time.Millisecond):
		s.Fail("Relay did not exit after context cancellation")
	}
}

// Mock implementations

type mockReportProvider struct {
	reports     []*v1.IndexReport
	startedChan chan struct{} // signals when first report is sent
}

func (m *mockReportProvider) Start(ctx context.Context) (<-chan *v1.IndexReport, error) {
	reportChan := make(chan *v1.IndexReport, len(m.reports))

	go func() {
		for i, report := range m.reports {
			select {
			case <-ctx.Done():
				return
			case reportChan <- report:
				// Signal when first report is sent
				if i == 0 && m.startedChan != nil {
					close(m.startedChan)
				}
			}
		}
	}()

	return reportChan, nil
}

type mockReportSender struct {
	mu            sync.Mutex
	sentReports   []*v1.IndexReport
	failOnIndex   int           // Index to fail on (0-based), use -1 to never fail
	doneChan      chan struct{} // signals when expectedCount reports are received
	expectedCount int            // number of reports expected before signaling done
}

func (m *mockReportSender) Send(_ context.Context, report *v1.IndexReport) error {
	m.mu.Lock()
	currentIndex := len(m.sentReports)
	m.sentReports = append(m.sentReports, report)

	// Signal done when we've received expected count
	if m.doneChan != nil && len(m.sentReports) == m.expectedCount {
		close(m.doneChan)
	}
	m.mu.Unlock()

	// Fail on the specified index
	if currentIndex == m.failOnIndex {
		return relaytest.ErrTest
	}
	return nil
}
