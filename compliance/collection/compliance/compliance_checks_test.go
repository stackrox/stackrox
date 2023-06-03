package compliance

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	complianceCompress "github.com/stackrox/rox/pkg/compliance/compress"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/orchestrators"
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
	decompresedBytes, err := io.ReadAll(gr)
	s.Require().NoError(err)
	var result complianceCompress.ResultWrapper
	err = json.Unmarshal(decompresedBytes, &result)
	s.Require().NoError(err)

	// This is a test artifact.  Compression/decompression will convert empty maps to nil maps, but we're going to do a .equals with a data structure containing an empty map.
	for _, standardResult := range result.ResultMap {
		if standardResult.NodeCheckResults == nil {
			standardResult.NodeCheckResults = make(map[string]*storage.ComplianceResultValue)
		}
		if standardResult.ClusterCheckResults == nil {
			standardResult.ClusterCheckResults = make(map[string]*storage.ComplianceResultValue)
		}
	}

	return &result
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
				NodeCheckResults: map[string]*storage.ComplianceResultValue{
					checkNameOne: {
						Evidence:     evidenceOne,
						OverallState: 0,
					},
				},
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{},
			},
		},
	}

	testResults := map[string]*compliance.ComplianceStandardResult{}
	addCheckResultsToResponse(testResults, standardID, checkNameOne, framework.NodeKind, evidenceOne)
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
	mockData.ResultMap[standardID].NodeCheckResults[checkNameTwo] = &storage.ComplianceResultValue{
		Evidence:     evidenceTwo,
		OverallState: 0,
	}
	addCheckResultsToResponse(testResults, standardID, checkNameTwo, framework.NodeKind, evidenceTwo)
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
		NodeCheckResults: map[string]*storage.ComplianceResultValue{
			checkNameThree: {
				Evidence:     evidenceThree,
				OverallState: 0,
			},
		},
		ClusterCheckResults: map[string]*storage.ComplianceResultValue{},
	}
	addCheckResultsToResponse(testResults, standardIDTwo, checkNameThree, framework.NodeKind, evidenceThree)
	s.Equal(mockData.ResultMap, testResults)

	// Add a cluster-level result
	checkNameFour := "jkdfdjk"
	evidenceFour := []*storage.ComplianceResultValue_Evidence{
		{
			State:   0,
			Message: "jkl",
		},
	}
	mockData.ResultMap[standardIDTwo].ClusterCheckResults = map[string]*storage.ComplianceResultValue{
		checkNameFour: {
			Evidence:     evidenceFour,
			OverallState: 0,
		},
	}
	addCheckResultsToResponse(testResults, standardIDTwo, checkNameFour, framework.ClusterKind, evidenceFour)
	s.Equal(mockData.ResultMap, testResults)
}

func (s *ComplianceResultsBuilderTestSuite) TestSend() {
	s.T().Setenv(string(orchestrators.NodeName), "fakeName")

	testResults, client, mockData := s.getMockData()

	err := sendResults(testResults, client, "test", &dummyNodeNameProvider{})
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

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "Foo"
}
