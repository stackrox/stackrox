package vsock

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

func ReadFromConn(conn net.Conn, maxSize int, timeout time.Duration, vsockCID uint32) ([]byte, error) {
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

func ParseIndexReport(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}

	if err := proto.Unmarshal(data, report); err != nil {
		return nil, errors.Wrap(err, "unmarshalling data")
	}
	return report, nil
}

func ExtractVsockCIDFromConnection(conn net.Conn) (uint32, error) {
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

// ValidateReportedVsockCID checks the vsock CID in the indexReport against the one extracted from the vsock connection
func ValidateReportedVsockCID(indexReport *v1.IndexReport, connVsockCID uint32) error {
	// Ensure the reported vsock CID is correct, to prevent spoofing
	if indexReport.GetVsockCid() != strconv.FormatUint(uint64(connVsockCID), 10) {
		metrics.IndexReportsMismatchingVsockCID.Inc()
		return errors.Errorf("mismatch between reported (%s) and real (%d) vsock CIDs", indexReport.GetVsockCid(), connVsockCID)
	}
	return nil
}
