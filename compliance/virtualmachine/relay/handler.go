package relay

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/compliance/virtualmachine/vsock"
)

// IndexReportHandler handles connections and parses index reports
type IndexReportHandler struct {
	parser *IndexReportParser
}

// NewIndexReportHandler creates a new index report handler
func NewIndexReportHandler() *IndexReportHandler {
	return &IndexReportHandler{
		parser: NewIndexReportParser(),
	}
}

// HandleConnection processes a single connection and returns an IndexReport
func (h *IndexReportHandler) HandleConnection(ctx context.Context, conn net.Conn) (interface{}, error) {
	log.Debugf("Handling vsock connection from %s", conn.RemoteAddr())

	// Read the data from the connection
	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data from connection")
	}
	log.Debugf("Read %d bytes from connection", len(data))

	// Extract ContextID from vsock address
	vsockCid := "unknown"
	if a, ok := conn.RemoteAddr().(*vsock.Addr); ok {
		vsockCid = strconv.Itoa(int(a.CID()))
		log.Debugf("Extracted ContextID %d from vsock address", a.CID())
	}

	// Parse the data into an IndexReport
	report, err := h.parser.ParseIndexReport(data, vsockCid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse index report")
	}

	log.Debugf("Successfully parsed index report with hash_id: %s", report.IndexV4.HashId)
	return report, nil
}
