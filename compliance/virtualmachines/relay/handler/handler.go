package handler

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/connection"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/sender"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

// Handler processes connections by reading index reports, validating them, and sending them to sensor.
type Handler struct {
	connectionReadTimeout time.Duration
	sensorClient          sensor.VirtualMachineIndexReportServiceClient
}

func New(sensorClient sensor.VirtualMachineIndexReportServiceClient) *Handler {
	return &Handler{
		connectionReadTimeout: 10 * time.Second,
		sensorClient:          sensorClient,
	}
}

func (h *Handler) Handle(ctx context.Context, conn net.Conn) error {
	log.Infof("Handling connection from %s", conn.RemoteAddr())

	indexReport, err := h.receiveAndValidateIndexReport(conn)
	if err != nil {
		return err
	}

	if err = sender.SendReportToSensor(ctx, indexReport, h.sensorClient); err != nil {
		log.Debugf("Error sending index report to sensor (vsock CID: %s): %v", indexReport.GetVsockCid(), err)
		return errors.Wrapf(err, "sending report to sensor (vsock CID: %s)", indexReport.GetVsockCid())
	}

	log.Debugf("Finished handling connection from %s", conn.RemoteAddr())

	return nil
}

func (h *Handler) receiveAndValidateIndexReport(conn net.Conn) (*v1.IndexReport, error) {
	vsockCID, err := vsock.ExtractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024
	data, err := connection.ReadFromConn(conn, maxSizeBytes, h.connectionReadTimeout)
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

// validateReportedVsockCID checks the vsock CID in the indexReport against the one extracted from the vsock connection
func validateReportedVsockCID(indexReport *v1.IndexReport, connVsockCID uint32) error {
	// Ensure the reported vsock CID is correct, to prevent spoofing
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(connVsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), connVsockCID)
	}
	return nil
}
