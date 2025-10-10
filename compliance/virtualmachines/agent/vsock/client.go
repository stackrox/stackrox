package vsock

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
)

var log = logging.LoggerForModule()

type Client struct {
	Port    uint32
	Timeout time.Duration
}

func (c *Client) SendIndexReport(report *v4.IndexReport) error {
	conn, err := vsock.Dial(vsock.Host, c.Port, &vsock.Config{})
	if err != nil {
		return fmt.Errorf("dialing vsock connection: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Failed to close vsock connection: %v", err)
		}
	}()
	if err := conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		return fmt.Errorf("setting connection deadline: %w", err)
	}
	vsockCid, err := vsock.ContextID()
	if err != nil {
		return fmt.Errorf("getting vsock context id: %w", err)
	}
	indexReport := &v1.IndexReport{}
	indexReport.SetVsockCid(strconv.Itoa(int(vsockCid)))
	indexReport.SetIndexV4(report)

	return c.writeIndexReport(conn, indexReport)
}

func (c *Client) writeIndexReport(conn net.Conn, report *v1.IndexReport) error {
	reportBytes, err := protocompat.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshalling index report: %w", err)
	}
	if _, err := conn.Write(reportBytes); err != nil {
		return fmt.Errorf("writing index report: %w", err)
	}
	log.Infof("Sent index report %q to host", report.GetIndexV4().GetHashId())
	return nil
}
