package vsock

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/mdlayher/vsock"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
)

var log = logging.LoggerForModule()

type Client struct {
	Port    uint32
	Timeout time.Duration
	Verbose bool
}

func (c *Client) SendIndexReport(report *v4.IndexReport) error {
	vsockCid, err := vsock.ContextID()
	if err != nil {
		return fmt.Errorf("getting vsock context id: %w", err)
	}
	wrappedReport := &v1.IndexReport{
		VsockCid: strconv.FormatUint(uint64(vsockCid), 10),
		IndexV4:  report,
	}

	// Create VsockMessage with placeholder discovered data values.
	vsockMsg := &v1.VsockMessage{
		IndexReport: wrappedReport,
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        "unknown", // TODO: get proper values from VM.
			ActivationStatus:  v1.ActivationStatus_ACTIVATION_STATUS_UNSPECIFIED,
			DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_STATUS_UNSPECIFIED,
		},
	}

	if c.Verbose {
		reportJson, err := jsonutil.ProtoToJSON(vsockMsg)
		if err != nil {
			log.Errorf("Failed to convert vsock message to JSON (vsockCid=%s): %v", wrappedReport.GetVsockCid(), err)
		} else {
			fmt.Println(reportJson)
		}
	}

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
	return c.writeVsockMessage(conn, vsockMsg)
}

func (c *Client) writeVsockMessage(conn net.Conn, msg *v1.VsockMessage) error {
	msgBytes, err := protocompat.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshalling vsock message: %w", err)
	}
	if _, err := conn.Write(msgBytes); err != nil {
		return fmt.Errorf("writing vsock message: %w", err)
	}

	// Safely count packages, handling potential nil values
	numPackages := 0
	if indexReport := msg.GetIndexReport(); indexReport != nil {
		if indexV4 := indexReport.GetIndexV4(); indexV4 != nil {
			if contents := indexV4.GetContents(); contents != nil {
				numPackages = len(contents.GetPackages())
			}
		}
	}
	log.Infof("Sent vsock message with index report (%d packages) to host", numPackages)
	return nil
}
