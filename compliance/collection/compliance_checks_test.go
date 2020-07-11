package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	complianceCompress "github.com/stackrox/rox/pkg/compliance/compress"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type mockClient struct {
	grpc.ClientStream

	sendList []*sensor.MsgFromCompliance
}

func (c *mockClient) Send(msg *sensor.MsgFromCompliance) error {
	c.sendList = append(c.sendList, msg)
	return nil
}

func (c *mockClient) Recv() (*sensor.MsgToCompliance, error) {
	return nil, nil
}

func TestComplianceResultsBuilder(t *testing.T) {
	suite.Run(t, new(ComplianceResultsBuilderTestSuite))
}

type ComplianceResultsBuilderTestSuite struct {
	suite.Suite
}

func (s *ComplianceResultsBuilderTestSuite) decompressEvidence(chunk *compliance.GZIPDataChunk) *complianceCompress.ResultWrapper {
	gr, err := gzip.NewReader(bytes.NewBuffer(chunk.Gzip))
	s.Require().NoError(err)
	defer func() {
		err := gr.Close()
		s.Require().NoError(err)
	}()
	decompresedBytes, err := ioutil.ReadAll(gr)
	s.Require().NoError(err)
	var result *complianceCompress.ResultWrapper
	err = json.Unmarshal(decompresedBytes, &result)
	s.Require().NoError(err)
	return result
}

func (s *ComplianceResultsBuilderTestSuite) getMockData() (map[string]*compliance.ComplianceStandardResult, *mockClient, *complianceCompress.ResultWrapper) {
	client := &mockClient{
		sendList: []*sensor.MsgFromCompliance{},
	}

	standardID := "Joseph"
	checkNameOne := "rules"
	evidenceOne := []*storage.ComplianceResultValue_Evidence{
		{
			State:   0,
			Message: "abc",
		},
	}
	mockData := &complianceCompress.ResultWrapper{
		ResultMap: map[string]*compliance.ComplianceStandardResult{
			standardID: {
				CheckResults: map[string]*storage.ComplianceResultValue{
					checkNameOne: {
						Evidence:     evidenceOne,
						OverallState: 0,
					},
				},
			},
		},
	}

	testResults := map[string]*compliance.ComplianceStandardResult{}
	addCheckResultsToResponse(testResults, standardID, checkNameOne, evidenceOne)
	s.Equal(mockData.ResultMap, testResults)

	return testResults, client, mockData
}

func (s *ComplianceResultsBuilderTestSuite) TestAddEvidence() {
	// Try one result
	testResults, _, mockData := s.getMockData()

	// Add another result from the same standard
	standardID := "Joseph"
	checkNameTwo := "is really great"
	evidenceTwo := []*storage.ComplianceResultValue_Evidence{
		{
			State:   0,
			Message: "def",
		},
	}
	mockData.ResultMap[standardID].CheckResults[checkNameTwo] = &storage.ComplianceResultValue{
		Evidence:     evidenceTwo,
		OverallState: 0,
	}
	addCheckResultsToResponse(testResults, standardID, checkNameTwo, evidenceTwo)
	s.Equal(mockData.ResultMap, testResults)

	// Add a result from a different standard
	standardIDTwo := "abababab"
	checkNameThree := "bababababa"
	evidenceThree := []*storage.ComplianceResultValue_Evidence{
		{
			State:   0,
			Message: "ghi",
		},
	}
	mockData.ResultMap[standardIDTwo] = &compliance.ComplianceStandardResult{
		CheckResults: map[string]*storage.ComplianceResultValue{
			checkNameThree: {
				Evidence:     evidenceThree,
				OverallState: 0,
			},
		},
	}
	addCheckResultsToResponse(testResults, standardIDTwo, checkNameThree, evidenceThree)
	s.Equal(mockData.ResultMap, testResults)
}

func (s *ComplianceResultsBuilderTestSuite) TestSend() {
	envIsolator := testutils.NewEnvIsolator(s.T())
	envIsolator.Setenv(string(orchestrators.NodeName), "fakeName")
	defer envIsolator.RestoreAll()

	testResults, client, mockData := s.getMockData()

	err := sendResults(testResults, client, "test")
	s.NoError(err)
	s.Require().Len(client.sendList, 1)
	msg := client.sendList[0]
	complianceReturn := msg.GetReturn()
	s.Require().NotNil(complianceReturn)
	zippedEvidence := complianceReturn.GetEvidence()
	s.Require().NotNil(zippedEvidence)
	unzippedEvidence := s.decompressEvidence(zippedEvidence)
	s.Equal(mockData, unzippedEvidence)
}
