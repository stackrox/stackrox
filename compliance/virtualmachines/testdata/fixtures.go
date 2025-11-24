package testdata

import (
	"embed"
	"fmt"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

//go:embed indexreport_small.pb indexreport_avg.pb indexreport_large.pb
var embeddedFixtures embed.FS

// EmbeddedFixture returns the embedded protobuf payload for the named size ("small", "avg", "large").
func EmbeddedFixture(size string) ([]byte, error) {
	filename, ok := map[string]string{
		"small": "indexreport_small.pb",
		"avg":   "indexreport_avg.pb",
		"large": "indexreport_large.pb",
	}[size]
	if !ok {
		return nil, fmt.Errorf("unknown embedded fixture size %q", size)
	}
	return embeddedFixtures.ReadFile(filename)
}

// LoadReportFromBytes parses a protobuf IndexReport from bytes.
func LoadReportFromBytes(data []byte) (*v1.IndexReport, error) {
	report := &v1.IndexReport{}
	if err := proto.Unmarshal(data, report); err != nil {
		return nil, err
	}
	return report, nil
}
