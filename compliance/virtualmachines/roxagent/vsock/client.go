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
	Port     uint32
	HostPath string
	Timeout  time.Duration
	Verbose  bool
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

	// Discover VM data from host system
	discovered := DiscoverVMData(c.HostPath)

	// Create VMReport with discovered data values.
	vmReport := &v1.VMReport{
		IndexReport: wrappedReport,
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        discovered.DetectedOS,
			OsVersion:         discovered.OSVersion,
			ActivationStatus:  discovered.ActivationStatus,
			DnfMetadataStatus: discovered.DnfMetadataStatus,
		},
	}

	if c.Verbose {
		reportJson, err := jsonutil.ProtoToJSON(vmReport)
		if err != nil {
			log.Errorf("Failed to convert VM report to JSON (vsockCid=%s): %v", wrappedReport.GetVsockCid(), err)
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
	return c.writeVMReport(conn, vmReport)
}

func (c *Client) writeVMReport(conn net.Conn, report *v1.VMReport) error {
	reportBytes, err := protocompat.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshalling VM report: %w", err)
	}
	if _, err := conn.Write(reportBytes); err != nil {
		return fmt.Errorf("writing VM report: %w", err)
	}

	numPackages := len(report.GetIndexReport().GetIndexV4().GetContents().GetPackages())
	log.Infof("Sent message with index report containing %d packages to host", numPackages)
	return nil
}
