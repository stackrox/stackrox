// Package handler provides handling logic for connections carrying virtual machine index reports.
// It coordinates reading, validating, and forwarding VM index reports to sensor.
// The validation is vsock-specific, since we ensure the reported vsock CID matches the real vsock CID.
package handler

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/connutil"
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

// Handler processes connections carrying virtual machine index reports.
type Handler interface {
	Handle(ctx context.Context, conn net.Conn) error
}

type handlerImpl struct {
	connectionReadTimeout time.Duration
	sensorClient          sensor.VirtualMachineIndexReportServiceClient
}

var _ Handler = (*handlerImpl)(nil)

func New(sensorClient sensor.VirtualMachineIndexReportServiceClient) Handler {
	return &handlerImpl{
		connectionReadTimeout: 10 * time.Second,
		sensorClient:          sensorClient,
	}
}

func (h *handlerImpl) Handle(ctx context.Context, conn net.Conn) error {
	log.Infof("Handling connection from %s", conn.RemoteAddr())

	indexReport, err := h.receiveAndValidateIndexReport(conn)
	if err != nil {
		return err
	}

	if err = sender.SendReportToSensor(ctx, indexReport, h.sensorClient); err != nil {
		log.Debugf("Error sending index report to sensor (vsock CID: %s): %v", indexReport.GetVsockCid(), err)
		return errors.Wrapf(err, "sending report to sensor (vsock CID: %s)", indexReport.GetVsockCid())
	}

	log.Infof("Finished handling connection from %s", conn.RemoteAddr())

	return nil
}

func (h *handlerImpl) receiveAndValidateIndexReport(conn net.Conn) (*v1.IndexReport, error) {
	vsockCID, err := vsock.ExtractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024
	data, err := connutil.ReadFromConn(conn, maxSizeBytes, h.connectionReadTimeout)
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

// validateReportedVsockCID prevents spoofing by ensuring the reported CID matches the connection's CID
func validateReportedVsockCID(indexReport *v1.IndexReport, connVsockCID uint32) error {
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(connVsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), connVsockCID)
	}
	return nil
}
