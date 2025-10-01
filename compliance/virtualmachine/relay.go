package virtualmachine

import (
	"context"
	"io"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachine/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

type VsockServer struct {
	listener *vsock.Listener
	port     uint32
}

func (s *VsockServer) Start() error {
	log.Debugf("Starting vsock server on port %d", s.port)
	l, err := vsock.Listen(s.port, nil)
	if err != nil {
		return errors.Wrapf(err, "listening on port %d", s.port)
	}
	s.listener = l
	return nil
}

func (s *VsockServer) Stop() {
	log.Infof("Stopping vsock server on port %d", s.port)
	if err := s.listener.Close(); err != nil {
		log.Errorf("Error closing vsock listener: %v", err)
	}
}

type Relay struct {
	vsockServer  VsockServer
	sensorClient sensor.VirtualMachineIndexReportServiceClient
}

func NewRelay(conn grpc.ClientConnInterface) *Relay {
	port := env.VirtualMachineVsockPort.IntegerSetting()
	return &Relay{
		sensorClient: sensor.NewVirtualMachineIndexReportServiceClient(conn),
		vsockServer:  VsockServer{port: uint32(port)},
	}
}

func (r *Relay) Run(ctx context.Context) error {
	log.Info("Starting virtual machine relay")

	if err := r.vsockServer.Start(); err != nil {
		return errors.Wrap(err, "starting vsock server")
	}
	defer r.vsockServer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down virtual machine relay")
			return ctx.Err()
		default:
			conn, err := r.vsockServer.listener.Accept()
			if err != nil {
				return err
			}
			go func() {
				if err := handleVsockConnection(ctx, conn, r.sensorClient); err != nil {
					log.Errorf("Error handling vsock connection: %v", err)
				}
			}()
		}
	}
}

func extractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
	remoteAddr, ok := conn.RemoteAddr().(*vsock.Addr)
	if !ok {
		return 0, errors.New("Failed to extract remote address from vsock connection")
	}

	return remoteAddr.ContextID, nil
}

func handleVsockConnection(ctx context.Context, conn net.Conn, sensorClient sensor.VirtualMachineIndexReportServiceClient) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	metrics.IndexReportsReceived.Inc()

	log.Debugf("Handling vsock connection from %s", conn.RemoteAddr())

	vsockCID, err := extractVsockCIDFromConnection(conn)
	if err != nil {
		return errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := 10 * 1024 * 1024 // 10 MB
	data, err := readFromConn(conn, maxSizeBytes)
	if err != nil {
		return errors.Wrap(err, "reading from vsock connection")
	}

	indexReport, err := parseIndexReport(data)
	if err != nil {
		return errors.Wrap(err, "parsing index report data")
	}

	// Fill the vsock context ID - at the moment the agent does not populate this field; if that changes, this can be
	// replaced with a sanity check.
	indexReport.VsockCid = strconv.Itoa(int(vsockCID))

	if err = sendReportToSensor(ctx, indexReport, sensorClient); err != nil {
		return errors.Wrap(err, "sending report to sensor")
	}

	return nil
}

func parseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

func readFromConn(conn net.Conn, maxSize int) ([]byte, error) {
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
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: report,
	}
	log.Debugf("Sending index report to sensor: %v", req)

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
				err = retry.MakeRetryable(errors.New("Sensor failed to handle virtual machine index report"))
			}
		}

		var transportErr *transport.Error
		var urlError *url.Error
		if errors.As(err, &transportErr) && transportErr.Temporary() {
			return retry.MakeRetryable(err)
		}
		if errors.As(err, &urlError) && urlError.Temporary() {
			return retry.MakeRetryable(err)
		}
		if errors.Is(err, errox.ResourceExhausted) {
			return retry.MakeRetryable(err)
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
		metrics.IndexReportsNotRelayed.Inc()
	} else {
		metrics.IndexReportsRelayed.Inc()
	}

	return err
}
