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

func (s *streamTestSuite) TestParseVsockMessage() {
	data := []byte("malformed-data")
	parsedVsockMessage, err := parseVsockMessage(data)
	s.Require().Error(err)
	s.Require().Nil(parsedVsockMessage)

	validVsockMessage := &v1.VsockMessage{
		IndexReport: &v1.IndexReport{VsockCid: "42"},
		DiscoveredData: &v1.DiscoveredData{
			DetectedOs:        "unknown",
			ActivationStatus:  v1.ActivationStatus_ACTIVATION_STATUS_UNSPECIFIED,
			DnfMetadataStatus: v1.DnfMetadataStatus_DNF_METADATA_STATUS_UNSPECIFIED,
		},
	}
	data, err = proto.Marshal(validVsockMessage)
	s.Require().NoError(err)
	parsedVsockMessage, err = parseVsockMessage(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validVsockMessage, parsedVsockMessage))
}

func (s *streamTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	vsockMessage := &v1.VsockMessage{
		IndexReport: &v1.IndexReport{VsockCid: "42"},
	}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(vsockMessage, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(vsockMessage, connVsockCID)
	s.Require().NoError(err)
}
