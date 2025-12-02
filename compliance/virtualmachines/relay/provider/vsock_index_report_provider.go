// Package provider manages the vsock server and produces validated index reports.
// It combines connection management, parsing, and validation before forwarding
// reports to the relay for sending to Sensor.
package provider

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/connutil"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

type VsockIndexReportProvider struct {
	listener              net.Listener
	semaphore             *semaphore.Weighted
	semaphoreTimeout      time.Duration
	connectionReadTimeout time.Duration
	waitAfterFailedAccept time.Duration
	maxSizeBytes          int
	stopOnce              sync.Once
}

// New creates a VsockIndexReportProvider with a vsock listener.
// Concurrency limits are read from env vars VirtualMachinesMaxConcurrentVsockConnections
// and VirtualMachinesConcurrencyTimeout.
func New() (*VsockIndexReportProvider, error) {
	listener, err := vsock.NewListener()
	if err != nil {
		return nil, errors.Wrap(err, "creating vsock listener")
	}

	maxConcurrentConnections := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	semaphoreTimeout := env.VirtualMachinesConcurrencyTimeout.DurationSetting()
	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024

	return &VsockIndexReportProvider{
		listener:              listener,
		semaphore:             semaphore.NewWeighted(int64(maxConcurrentConnections)),
		semaphoreTimeout:      semaphoreTimeout,
		connectionReadTimeout: 10 * time.Second,
		waitAfterFailedAccept: time.Second,
		maxSizeBytes:          maxSizeBytes,
	}, nil
}

// Start begins accepting vsock connections and returns a channel of validated reports.
// The provider spawns goroutines to handle each connection concurrently (up to the
// configured limit). Reports are validated before being sent to the channel.
func (p *VsockIndexReportProvider) Start(ctx context.Context) (<-chan *v1.IndexReport, error) {
	log.Info("Starting report provider")

	if p.listener == nil {
		return nil, errors.New("listener is nil")
	}

	// Buffer size = concurrency limit to allow provider goroutines to complete
	// without blocking on sender
	bufferSize := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	reportChan := make(chan *v1.IndexReport, bufferSize)

	// Start the accept loop in a goroutine
	go p.acceptLoop(ctx, reportChan)

	return reportChan, nil
}

func (p *VsockIndexReportProvider) acceptLoop(ctx context.Context, reportChan chan<- *v1.IndexReport) {
	defer close(reportChan)
	defer p.stop()

	var wg sync.WaitGroup
	defer wg.Wait() // Wait for all handlers to finish before closing channel

	go func() {
		<-ctx.Done()
		p.stop()
	}()

	for {
		// Accept() is blocking, but it will return when ctx is cancelled and the above goroutine calls p.stop()
		conn, err := p.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping report provider")
				return
			}

			// We deliberately don't kill the listener on errors. The only way to stop that is to cancel the context.
			// If we had return here on fatal errors, then compliance would continue working without the relay
			// and that would make it an invisible problem to the user.
			log.Errorf("Error accepting connection: %v", err)

			time.Sleep(p.waitAfterFailedAccept) // Prevent a tight loop
			continue
		}
		metrics.ConnectionsAccepted.Inc()

		if err := p.acquireSemaphore(ctx); err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping report provider")
				return
			}

			// This log is rate-limited because when the concurrency limit is reached it is emitted every
			// semaphoreTimeout, which is user-configurable (min: 1 second).
			logging.GetRateLimitedLogger().WarnL(
				"relay semaphore timeout",
				"Failed to acquire semaphore to handle connection: %v",
				err,
			)

			// When the concurrency limit is reached, the semaphore cannot be acquired. We close the connection and
			// continue to listen. In this case, there is no need to add an extra wait to prevent a busy loop, because
			// we already waited semaphoreTimeout
			if err := conn.Close(); err != nil {
				log.Warnf("Failed to close connection after failing to acquire semaphore: %v", err)
			}

			continue
		}

		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			p.handleConnection(ctx, conn, reportChan)
		}(conn)
	}
}

func (p *VsockIndexReportProvider) handleConnection(ctx context.Context, conn net.Conn, reportChan chan<- *v1.IndexReport) {
	defer p.releaseSemaphore()

	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}(conn)

	log.Infof("Handling connection from %s", conn.RemoteAddr())

	indexReport, err := p.receiveAndValidateIndexReport(conn)
	if err != nil {
		log.Errorf("Error handling connection from %v: %v", conn.RemoteAddr(), err)
		return
	}

	log.Infof("Finished handling connection from %s", conn.RemoteAddr())

	// Send validated report to channel
	// WaitGroup ensures channel won't be closed while we're sending
	reportChan <- indexReport
}

func (p *VsockIndexReportProvider) receiveAndValidateIndexReport(conn net.Conn) (*v1.IndexReport, error) {
	vsockCID, err := vsock.ExtractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	data, err := connutil.ReadFromConn(conn, p.maxSizeBytes, p.connectionReadTimeout)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from connection (vsock CID: %d)", vsockCID)
	}

	log.Debugf("Parsing index report (vsock CID: %d)", vsockCID)
	indexReport, err := parseIndexReport(data)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing index report data (vsock CID: %d)", vsockCID)
	}
	metrics.IndexReportsReceived.Inc()

	err = validateReportedVsockCID(indexReport, vsockCID)
	if err != nil {
		log.Debugf("Error validating reported vsock CID: %v", err)
		return nil, errors.Wrap(err, "validating reported vsock CID")
	}

	return indexReport, nil
}

func parseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

// validateReportedVsockCID ensures the report's vsock CID matches the connection.
func validateReportedVsockCID(indexReport *v1.IndexReport, connVsockCID uint32) error {
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(connVsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), connVsockCID)
	}
	return nil
}

func (p *VsockIndexReportProvider) stop() {
	p.stopOnce.Do(func() {
		log.Info("Stopping connection server")
		if p.listener != nil {
			if err := p.listener.Close(); err != nil {
				log.Errorf("Error closing listener: %v", err)
			}
		}
	})
}

func (p *VsockIndexReportProvider) acquireSemaphore(parentCtx context.Context) error {
	semCtx, cancel := context.WithTimeout(parentCtx, p.semaphoreTimeout)
	defer cancel()

	metrics.SemaphoreQueueSize.Inc()
	defer metrics.SemaphoreQueueSize.Dec()
	if err := p.semaphore.Acquire(semCtx, 1); err != nil {
		reason := "unknown"
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug("Could not acquire semaphore, too many concurrent connections")
			reason = "concurrency_limit"
		} else if errors.Is(err, context.Canceled) {
			log.Debug("Could not acquire semaphore, the context was canceled")
			reason = "context_canceled"
		}
		metrics.SemaphoreAcquisitionFailures.With(prometheus.Labels{"reason": reason}).Inc()
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	metrics.SemaphoreHoldingSize.Inc()
	return nil
}

func (p *VsockIndexReportProvider) releaseSemaphore() {
	p.semaphore.Release(1)
	metrics.SemaphoreHoldingSize.Dec()
}
