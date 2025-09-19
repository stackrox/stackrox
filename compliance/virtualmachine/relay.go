package virtualmachine

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
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
	port := 1024 // TODO: Make configurable
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
			go handleVsockConnection(ctx, conn, r.sensorClient)
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

func handleVsockConnection(ctx context.Context, conn net.Conn, sensorClient sensor.VirtualMachineIndexReportServiceClient) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close connection: %v", err)
		}
	}()

	log.Debugf("Handling vsock connection from %s", conn.RemoteAddr())

	vsockCID, err := extractVsockCIDFromConnection(conn)
	if err != nil {
		log.Errorf("Error extracting vsock CID: %v", err)
		return
	}

	data, err := io.ReadAll(conn)
	if err != nil {
		log.Errorf("Failed to read data from vsock connection (vsock CID %d): %v", vsockCID, err)
		return
	}

	indexReport, err := parseIndexReport(data)
	if err != nil {
		log.Errorf("Failed to parse index report: %v", err)
		return
	}

	// Fill the vsock context ID - at the moment the agent does not populate this field; if that changes, this can be
	// replaced with a sanity check.
	indexReport.VsockCid = strconv.Itoa(int(vsockCID))

	err = sendReportToSensor(ctx, indexReport, sensorClient)
	if err != nil {
		log.Errorf("Failed to send report to sensor: %v", err)
	}
}

func parseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

func sendReportToSensor(ctx context.Context, report *v1.IndexReport, sensorClient sensor.VirtualMachineIndexReportServiceClient) error {
	req := &sensor.UpsertVirtualMachineIndexReportRequest{
		IndexReport: report,
	}
	resp, err := sensorClient.UpsertVirtualMachineIndexReport(ctx, req)
	if err != nil {
		return errors.Wrap(err, "calling sensor VM index client")
	}
	if !resp.Success {
		return errors.New("Sensor failed to handle virtual machine index report")
	}

	log.Debugf("Successfully sent index report to sensor")

	return nil
}
