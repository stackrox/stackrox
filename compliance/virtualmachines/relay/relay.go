package relay

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

type vsockServer interface {
	start() error
	stop()
	acquireSemaphore(ctx context.Context) error
	releaseSemaphore()
	accept() (net.Conn, error)
}

type vsockServerImpl struct {
	listener         *vsock.Listener
	port             uint32
	semaphore        *semaphore.Weighted
	semaphoreTimeout time.Duration
}

var _ vsockServer = (*vsockServerImpl)(nil)

func newVsockServer() *vsockServerImpl {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	maxConcurrentConnections := env.VirtualMachinesMaxConcurrentVsockConnections.IntegerSetting()
	semaphoreTimeout := env.VirtualMachinesConcurrencyTimeout.DurationSetting()
	return &vsockServerImpl{
		port:             uint32(port),
		semaphore:        semaphore.NewWeighted(int64(maxConcurrentConnections)),
		semaphoreTimeout: semaphoreTimeout,
	}
}

func (s *vsockServerImpl) start() error {
	log.Debugf("Starting vsock server on port %d", s.port)
	l, err := vsock.ListenContextID(vsock.Host, s.port, nil)
	if err != nil {
		return errors.Wrapf(err, "listening on port %d", s.port)
	}
	s.listener = l
	return nil
}

func (s *vsockServerImpl) stop() {
	log.Infof("Stopping vsock server on port %d", s.port)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Errorf("Error closing vsock listener: %v", err)
		}
	}
}

func (s *vsockServerImpl) accept() (net.Conn, error) {
	if s.listener == nil {
		return nil, fmt.Errorf("vsock server has not been started on port %d", s.port)
	}
	return s.listener.Accept()
}

func (s *vsockServerImpl) acquireSemaphore(parentCtx context.Context) error {
	semCtx, cancel := context.WithTimeout(parentCtx, s.semaphoreTimeout)
	defer cancel()

	metrics.VsockSemaphoreQueueSize.Inc()
	defer metrics.VsockSemaphoreQueueSize.Dec()
	if err := s.semaphore.Acquire(semCtx, 1); err != nil {
		reason := "unknown"
		if errors.Is(err, context.DeadlineExceeded) {
			log.Debug("Could not acquire semaphore, too many concurrent vsock connections")
			reason = "concurrency_limit"
		} else if errors.Is(err, context.Canceled) {
			log.Debug("Could not acquire semaphore, the context was canceled")
			reason = "context_canceled"
		}
		metrics.VsockSemaphoreAcquisitionFailures.With(prometheus.Labels{"reason": reason}).Inc()
		return errors.Wrap(err, "failed to acquire semaphore")
	}
	metrics.VsockSemaphoreHoldingSize.Inc()
	return nil
}

func (s *vsockServerImpl) releaseSemaphore() {
	s.semaphore.Release(1)
	metrics.VsockSemaphoreHoldingSize.Dec()
}

type Relay struct {
	connectionReadTimeout time.Duration
	ctx                   context.Context
	sensorClient          sensor.VirtualMachineIndexReportServiceClient
	vsockServer           vsockServer
	waitAfterFailedAccept time.Duration
}

func NewRelay(ctx context.Context, conn grpc.ClientConnInterface) *Relay {
	return &Relay{
		connectionReadTimeout: 10 * time.Second,
		ctx:                   ctx,
		sensorClient:          sensor.NewVirtualMachineIndexReportServiceClient(conn),
		vsockServer:           newVsockServer(),
		waitAfterFailedAccept: time.Second,
	}
}

func (r *Relay) Run() error {
	log.Info("Starting virtual machine relay")

	if err := r.vsockServer.start(); err != nil {
		return errors.Wrap(err, "starting vsock server")
	}

	go func() {
		<-r.ctx.Done()
		r.vsockServer.stop()
	}()

	for {
		// Accept() is blocking, but it will return when ctx is cancelled and the above goroutine calls r.vsockServer.Stop()
		conn, err := r.vsockServer.accept()
		if err != nil {
			if r.ctx.Err() != nil {
				log.Info("Stopping virtual machine relay")
				return r.ctx.Err()
			}

			// We deliberately don't kill the listener on errors. The only way to stop that is to cancel the context.
			// If we had return here on fatal errors, then compliance would continue working without the relay
			// and that would make it an invisible problem to the user.
			log.Errorf("Error accepting connection: %v", err)

			time.Sleep(r.waitAfterFailedAccept) // Prevent a tight loop
			continue
		}
		metrics.VsockConnectionsAccepted.Inc()

		if err := r.vsockServer.acquireSemaphore(r.ctx); err != nil {
			if r.ctx.Err() != nil {
				log.Info("Stopping virtual machine relay")
				return r.ctx.Err()
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

		go func(conn net.Conn) {
			defer r.vsockServer.releaseSemaphore()

			defer func(conn net.Conn) {
				if err := conn.Close(); err != nil {
					log.Errorf("Failed to close connection: %v", err)
				}
			}(conn)

			if err := r.handleVsockConnection(conn); err != nil {
				log.Errorf("Error handling vsock connection from %v: %v", conn.RemoteAddr(), err)
			}
		}(conn)
	}
}

func (r *Relay) handleVsockConnection(conn net.Conn) error {
	log.Infof("Handling vsock connection from %s", conn.RemoteAddr())

	indexReport, err := r.receiveAndValidateIndexReport(conn)
	if err != nil {
		return err
	}

	if err = sendReportToSensor(r.ctx, indexReport, r.sensorClient); err != nil {
		log.Debugf("Error sending index report to sensor (vsock CID: %s): %v", indexReport.GetVsockCid(), err)
		return errors.Wrapf(err, "sending report to sensor (vsock CID: %s)", indexReport.GetVsockCid())
	}

	log.Infof("Finished handling vsock connection from %s", conn.RemoteAddr())

	return nil
}

func (r *Relay) receiveAndValidateIndexReport(conn net.Conn) (*v1.IndexReport, error) {
	vsockCID, err := extractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024
	data, err := readFromConn(conn, maxSizeBytes, r.connectionReadTimeout, vsockCID)
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

func extractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
	remoteAddr, ok := conn.RemoteAddr().(*vsock.Addr)
	if !ok {
		return 0, fmt.Errorf("failed to extract remote address from vsock connection: unexpected type %T, value: %v",
			conn.RemoteAddr(), conn.RemoteAddr())
	}

	// Reject invalid values according to the vsock spec (https://www.man7.org/linux/man-pages/man7/vsock.7.html)
	if remoteAddr.ContextID <= 2 {
		return 0, fmt.Errorf("received an invalid vsock context ID: %d (values <=2 are reserved)", remoteAddr.ContextID)
	}

	return remoteAddr.ContextID, nil
}

func parseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

func readFromConn(conn net.Conn, maxSize int, timeout time.Duration, vsockCID uint32) ([]byte, error) {
	log.Debugf("Reading from connection (max bytes: %d, timeout: %s, vsockCID: %d)", maxSize, timeout, vsockCID)

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, errors.Wrapf(err, "setting read deadline on connection (vsockCID: %d)", vsockCID)
	}

	// Even if not strictly required, we limit the amount of data to be read to protect Sensor against large workloads.
	// Add 1 to the limit so we can detect oversized data. If we used exactly maxSize, we couldn't tell the difference
	// between a valid message of exactly maxSize bytes and an invalid message that's larger than maxSize (both would
	// read maxSize bytes). With maxSize+1, reading more than maxSize bytes means the original data was too large.
	limitedReader := io.LimitReader(conn, int64(maxSize+1))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, errors.Wrapf(err, "reading data from vsock connection (vsockCID: %d)", vsockCID)
	}

	if len(data) > maxSize {
		return nil, errors.Errorf("data size exceeds the limit (%d bytes, vsockCID: %d)", maxSize, vsockCID)
	}

	return data, nil
}

func sendReportToSensor(ctx context.Context, report *v1.IndexReport, sensorClient sensor.VirtualMachineIndexReportServiceClient) error {
	log.Infof("Sending index report to sensor (vsockCID: %s)", report.GetVsockCid())

	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: report,
	}

	// Considering a timeout of 5 seconds and 10 tries with exponential backoff, the maximum time spent in this function
	// is around 1 min 40 s. Given that each virtual machine sends an index report every 4 hours, these retries seem
	// reasonable and are unlikely to cause issues.
	err := retry.WithRetry(func() error {
		sendToSensorCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		resp, err := sensorClient.UpsertVirtualMachineIndexReport(sendToSensorCtx, req)

		if resp != nil && !resp.GetSuccess() {
			// This can't happen as of this writing (Success is only false when an error is returned) but is
			// theoretically possible, let's add retries too.
			if err == nil {
				log.Errorf("Sending index report didn't return an error but response indicated failure: %v", resp)
				err = retry.MakeRetryable(errors.New("sensor failed to handle virtual machine index report"))
			}
		}

		// Retry transient gRPC errors: unavailable, resource exhausted, internal, and deadline exceeded
		// (unless deadline exceeded is due to context cancellation).
		policy := retry.NoGrpcCodesPolicy().WithRetryableCodes(
			codes.Unavailable, codes.ResourceExhausted, codes.Internal, codes.DeadlineExceeded)

		// Don't retry if the context was canceled
		if errors.Is(err, context.Canceled) {
			return err
		}

		if policy.ShouldRetry(err) {
			err = retry.MakeRetryable(err)
		}

		return err
	},
		retry.WithContext(ctx),
		retry.OnFailedAttempts(func(e error) {
			log.Warnf("Error sending index report to sensor, retrying. Error was: %v", e)
		}),
		retry.Tries(10), // With current wait values in exponential backoff logic, this takes around 50 s
		retry.OnlyRetryableErrors(),
		retry.WithExponentialBackoff())

	metrics.IndexReportsSentToSensor.With(prometheus.Labels{"failed": strconv.FormatBool(err != nil)}).Inc()

	return err
}

// validateReportedVsockCID checks the vsock CID in the indexReport against the one extracted from the vsock connection
func validateReportedVsockCID(indexReport *v1.IndexReport, connVsockCID uint32) error {
	// Ensure the reported vsock CID is correct, to prevent spoofing
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(connVsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), connVsockCID)
	}
	return nil
}
