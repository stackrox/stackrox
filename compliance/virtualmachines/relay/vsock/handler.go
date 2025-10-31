package vsock

import (
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/env"
)

// HandleConnection processes a vsock connection, receiving and validating the index report.
// Returns the validated index report or an error.
func HandleConnection(conn net.Conn, connectionReadTimeout time.Duration) (*v1.IndexReport, error) {
	log.Infof("Handling vsock connection from %s", conn.RemoteAddr())

	indexReport, err := receiveAndValidateIndexReport(conn, connectionReadTimeout)
	if err != nil {
		return nil, err
	}

	log.Debugf("Finished handling vsock connection from %s", conn.RemoteAddr())
	return indexReport, nil
}

func receiveAndValidateIndexReport(conn net.Conn, connectionReadTimeout time.Duration) (*v1.IndexReport, error) {
	vsockCID, err := ExtractVsockCIDFromConnection(conn)
	if err != nil {
		return nil, errors.Wrap(err, "extracting vsock CID")
	}

	maxSizeBytes := env.VirtualMachinesVsockConnMaxSizeKB.IntegerSetting() * 1024
	data, err := ReadFromConn(conn, maxSizeBytes, connectionReadTimeout, vsockCID)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from connection (vsock CID: %d)", vsockCID)
	}

	log.Debugf("Parsing index report (vsock CID: %d)", vsockCID)
	indexReport, err := ParseIndexReport(data)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing index report data (vsock CID: %d)", vsockCID)
	}
	metrics.IndexReportsReceived.Inc()

	err = ValidateReportedVsockCID(indexReport, vsockCID)
	if err != nil {
		log.Debugf("Error validating reported vsock CID: %v", err)
		return nil, errors.Wrap(err, "validating reported vsock CID")
	}

	return indexReport, nil
}
