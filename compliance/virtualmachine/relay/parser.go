package relay

import (
	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

// IndexReportParser handles parsing raw data into IndexReport
type IndexReportParser struct{}

// NewIndexReportParser creates a new index report parser
func NewIndexReportParser() *IndexReportParser {
	return &IndexReportParser{}
}

// ParseIndexReport converts raw bytes into an IndexReport
func (p *IndexReportParser) ParseIndexReport(data []byte, vsockCid string) (*v1.IndexReport, error) {
	if len(data) == 0 {
		return nil, errors.New("received empty data")
	}

	// Parse the protobuf data as a v4.IndexReport
	var scannerIndexReport v4.IndexReport
	if err := proto.Unmarshal(data, &scannerIndexReport); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal index report from data")
	}

	// Create the virtual machine IndexReport
	return &v1.IndexReport{
		VsockCid: vsockCid,
		IndexV4:  &scannerIndexReport,
	}, nil
}
