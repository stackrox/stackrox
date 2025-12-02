package provider

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
)

func TestProvider(t *testing.T) {
	suite.Run(t, new(providerTestSuite))
}

type providerTestSuite struct {
	suite.Suite
}

func (s *providerTestSuite) TestParseIndexReport() {
	data := []byte("malformed-data")
	parsedIndexReport, err := parseIndexReport(data)
	s.Require().Error(err)
	s.Require().Nil(parsedIndexReport)

	validIndexReport := &v1.IndexReport{VsockCid: "42"}
	data, err = proto.Marshal(validIndexReport)
	s.Require().NoError(err)
	parsedIndexReport, err = parseIndexReport(data)
	s.Require().NoError(err)
	s.Require().True(proto.Equal(validIndexReport, parsedIndexReport))
}

func (s *providerTestSuite) TestValidateVsockCID() {
	// Reported CID is 42
	indexReport := v1.IndexReport{VsockCid: "42"}

	// Real (connection) CID is 99 - does not match, should return error
	connVsockCID := uint32(99)
	err := validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().Error(err)

	// Real (connection) CID is 42 - matches, should return nil
	connVsockCID = uint32(42)
	err = validateReportedVsockCID(&indexReport, connVsockCID)
	s.Require().NoError(err)
}
