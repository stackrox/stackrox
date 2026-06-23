package vsockclient

import (
	"fmt"
	"io"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

const maxReportSize = 10 * 1024 * 1024 // 10 MiB safety limit

// DefaultVSOCKPort is the port roxagent listens on inside the VM.
const DefaultVSOCKPort uint32 = 818

// StreamReader is the minimal interface we need from KubeVirt's StreamInterface.
type StreamReader interface {
	io.Reader
	Close() error
}

// ReadVMReport reads a protobuf VMReport from the stream and closes it.
// The stream comes from virtClient.VirtualMachineInstance(ns).VSOCK(name, opts).
func ReadVMReport(stream StreamReader) (*v1.VMReport, error) {
	defer func() { _ = stream.Close() }()

	data, err := io.ReadAll(io.LimitReader(stream, maxReportSize))
	if err != nil {
		return nil, fmt.Errorf("reading from VSOCK stream: %w", err)
	}

	report := &v1.VMReport{}
	if err := proto.Unmarshal(data, report); err != nil {
		return nil, fmt.Errorf("unmarshalling VMReport: %w", err)
	}

	log.Infof("Received VM report via VSOCK pull (%d bytes, vsock_cid=%s)",
		len(data), report.GetIndexReport().GetVsockCid())
	return report, nil
}
