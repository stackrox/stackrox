package stream

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestStream(t *testing.T) {
	suite.Run(t, new(streamTestSuite))
}

type streamTestSuite struct {
	suite.Suite
}

func (s *streamTestSuite) TestParseVMReport() {
	data := []byte("malformed-data")
	parsedVMReport, err := parseVMReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedVMReport)

	s.Run("should parse message without discovered data", func() {
		indexReportOnly := &v1.VMReport{
			IndexReport: &v1.IndexReport{},
		}
		marshaledIndexReportOnly, err := proto.Marshal(indexReportOnly)
		s.Require().NoError(err)

		parsedVMReport, err := parseVMReport(marshaledIndexReportOnly)
		s.Require().NoError(err)
		s.Require().NotNil(parsedVMReport)
		s.Require().Nil(parsedVMReport.GetDiscoveredData())
	})

	validVMReport := &v1.VMReport{
		IndexReport: &v1.IndexReport{VsockCid: "42"},
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        v1.DetectedOS_UNKNOWN,
			OsVersion:         "",
			ActivationStatus:  v1.ActivationStatus_ACTIVATION_UNSPECIFIED,
			DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_UNSPECIFIED,
		},
	}
	data, err = proto.Marshal(validVMReport)
	s.Require().NoError(err)
	parsedVMReport, err = parseVMReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validVMReport, parsedVMReport))
}

func (s *streamTestSuite) TestValidateVsockCID() {
	s.Run("missing index report does not panic", func() {
		vmReport := &v1.VMReport{}
		connVsockCID := uint32(42)
		s.NotPanics(func() {
			err := validateReportedVsockCID(vmReport, connVsockCID)
			s.Require().Error(err)
		})
	})

	// Reported CID is 42
	vmReport := &v1.VMReport{
		IndexReport: &v1.IndexReport{VsockCid: "42"},
	}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(vmReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(vmReport, connVsockCID)
	s.Require().NoError(err)
}
