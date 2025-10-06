package virtualmachine

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachine/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

type VsockServer struct {
	listener *vsock.Listener
	port     uint32
}

func (s *VsockServer) Start() error {
	log.Debugf("Starting vsock server on port %d", s.port)
	l, err := vsock.ListenContextID(vsock.Host, s.port, nil)
	if err != nil {
		return errors.Wrapf(err, "listening on port %d", s.port)
	}
	s.listener = l
	return nil
}

func (s *VsockServer) Stop() {
	log.Infof("Stopping vsock server on port %d", s.port)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Errorf("Error closing vsock listener: %v", err)
		}
	}
}

type Relay struct {
	connectionReadTimeout time.Duration
	sensorClient          sensor.VirtualMachineIndexReportServiceClient
	vsockServer           VsockServer
	waitAfterFailedAccept time.Duration
}

func NewRelay(conn grpc.ClientConnInterface) *Relay {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
	return &Relay{
		connectionReadTimeout: 10 * time.Second,
		sensorClient:          sensor.NewVirtualMachineIndexReportServiceClient(conn),
		vsockServer:           VsockServer{port: uint32(port)},
		waitAfterFailedAccept: time.Second,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")

	if err := r.vsockServer.Start(); err != nil {
		return errors.Wrap(err, "starting vsock server")
	}

	go func() {
		<-ctx.Done()
		r.vsockServer.Stop()
	}()

	for {
		conn, err := r.vsockServer.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				log.Info("Stopping virtual machine relay")
				return ctx.Err()
			}
			log.Errorf("Error accepting connection: %v", err)
			time.Sleep(r.waitAfterFailedAccept) // Prevent a tight loop
			continue
		}

		go func() {
			defer func() {
				if err := conn.Close(); err != nil {
					log.Errorf("Failed to close connection: %v", err)
				}
			}()

			if err := handleVsockConnection(ctx, conn, r.sensorClient, r.connectionReadTimeout); err != nil {
				log.Errorf("Error handling vsock connection: %v", err)
			}
		}()
	}
}

func extractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
	remoteAddr, ok := conn.RemoteAddr().(*vsock.Addr)
	if !ok {
		return 0, fmt.Errorf("failed to extract remote address from vsock connection: unexpected type %T, value: %v",
			conn.RemoteAddr(), conn.RemoteAddr())
	}

	if remoteAddr.ContextID <= 2 {
		return 0, fmt.Errorf("received an invalid vsock context ID: %d (values <=2 are reserved)", remoteAddr.ContextID)
	}

	return remoteAddr.ContextID, nil
}

func handleVsockConnection(ctx context.Context, conn net.Conn, sensorClient sensor.VirtualMachineIndexReportServiceClient, connectionReadTimeout time.Duration) error {
	metrics.VsockConnectionsAccepted.Inc()

	log.Infof("Handling vsock connection from %s", conn.RemoteAddr())

	vsockCID, err := extractVsockCIDFromConnection(conn)
	if err != nil {
		return errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024
	log.Debugf("Reading from connection (max bytes: %d, timeout: %s", maxSizeBytes, connectionReadTimeout)
	data, err := readFromConn(conn, maxSizeBytes, connectionReadTimeout)
	if err != nil {
		return errors.Wrapf(err, "reading from connection (vsock CID: %d)", vsockCID)
	}

	log.Debugf("Parsing index report (vsock CID: %d)", vsockCID)
	indexReport, err := parseIndexReport(data)
	if err != nil {
		return errors.Wrapf(err, "parsing index report data (vsock CID: %d)", vsockCID)
	}
	metrics.IndexReportsReceived.Inc()

	// Ensure the reported vsock CID is correct, to prevent spoofing
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(vsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		log.Errorf("Received index report with invalid vsock CID: %d", vsockCID)
		return fmt.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), vsockCID)
	}

	if err = sendReportToSensor(ctx, indexReport, sensorClient); err != nil {
		return errors.Wrapf(err, "sending report to sensor (vsock CID: %d)", vsockCID)
	}

	log.Debugf("Finished handling vsock connection from %s", conn.RemoteAddr())

	return nil
}

func isRetryableGRPCError(err error) bool {
	grpcErr, ok := status.FromError(err)
	if !ok {
		return false
	}
	code := grpcErr.Code()
	switch code {
	case codes.DeadlineExceeded:
		return !errors.Is(err, context.Canceled)
	case codes.Unavailable, codes.ResourceExhausted, codes.Internal:
		return true
	default:
		return false
	}
}

func parseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

func readFromConn(conn net.Conn, maxSize int, timeout time.Duration) ([]byte, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, errors.Wrap(err, "setting read deadline on connection")
	}

	// Add 1 to the limit so we can detect oversized data. If we used exactly maxSize, we couldn't tell the difference
	// between a valid message of exactly maxSize bytes and an invalid message that's larger than maxSize (both would
	// read maxSize bytes). With maxSize+1, reading more than maxSize bytes means the original data was too large.
	limitedReader := io.LimitReader(conn, int64(maxSize+1))
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, errors.Wrap(err, "reading data from vsock connection")
	}

	if len(data) > maxSize {
		return nil, errors.Errorf("data size exceeds the limit (%d bytes)", maxSize)
	}

	return data, nil
}

func sendReportToSensor(ctx context.Context, report *v1.IndexReport, sensorClient sensor.VirtualMachineIndexReportServiceClient) error {
	log.Infof("Sending index report to sensor (vsockCID: %s)", report.VsockCid)

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

		if resp != nil && !resp.Success {
			// This can't happen as of this writing (Success is only false when an error is returned) but is
			// theoretically possible, let's add retries too.
			if err == nil {
				log.Errorf("Sending index report didn't return an error but response indicated failure: %v", resp)
				err = retry.MakeRetryable(errors.New("sensor failed to handle virtual machine index report"))
			}
		}

		if isRetryableGRPCError(err) {
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

	if err != nil {
		metrics.IndexReportsSentToSensor.With(metrics.StatusFailLabels).Inc()
		log.Errorf("Error sending index report to sensor (vsock CID: %d): %v", report.VsockCid, err)
	} else {
		metrics.IndexReportsSentToSensor.With(metrics.StatusSuccessLabels).Inc()
		log.Debugf("Successfully sent index report to sensor (vsock CID: %d)", report.VsockCid)
	}

	return err
}
