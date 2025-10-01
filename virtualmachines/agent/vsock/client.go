package vsock

import (
	"fmt"
	"net"
	"time"

	"github.com/mdlayher/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/encoding/protojson"
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
	// This is currently needed because claircore 1.5.40 enumerates
	// repositories as integers, but we reference repos by name in the
	// environments.
	report = c.fixupIndexReport(report)
	conn.SetDeadline(time.Now().Add(c.Timeout))
	return c.writeIndexReport(conn, report)
}

func (c *Client) fixupIndexReport(report *v4.IndexReport) *v4.IndexReport {
	for _, repo := range report.GetContents().GetRepositories() {
		repo.Id = repo.GetName()
	}
	return report
}

func (c *Client) writeIndexReport(conn net.Conn, report *v4.IndexReport) error {
	jsonBytes, err := protojson.Marshal(report)
	if err != nil {
		return err
	}

	log.Infof(string(jsonBytes))

	reportBytes, err := protocompat.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshalling index report: %w", err)
	}
	if _, err := conn.Write(reportBytes); err != nil {
		return fmt.Errorf("writing index report: %w", err)
	}
	log.Infof("Sent index report %q to host", report.GetHashId())
	return nil
}
