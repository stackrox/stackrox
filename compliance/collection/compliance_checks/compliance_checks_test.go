package compliance_checks

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
	"github.com/stackrox/rox/pkg/protoassert"
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
	gr, err := gzip.NewReader(bytes.NewBuffer(chunk.GetGzip()))
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
		if standardResult.GetNodeCheckResults() == nil {
			standardResult.SetNodeCheckResults(make(map[string]*storage.ComplianceResultValue))
		}
		if standardResult.GetClusterCheckResults() == nil {
			standardResult.SetClusterCheckResults(make(map[string]*storage.ComplianceResultValue))
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
	ce := &storage.ComplianceResultValue_Evidence{}
	ce.SetState(0)
	ce.SetMessage("abc")
	evidenceOne := []*storage.ComplianceResultValue_Evidence{
		ce,
	}
	mockData := &complianceCompress.ResultWrapper{
		ResultMap: map[string]*compliance.ComplianceStandardResult{
			standardID: compliance.ComplianceStandardResult_builder{
				NodeCheckResults: map[string]*storage.ComplianceResultValue{
					checkNameOne: storage.ComplianceResultValue_builder{
						Evidence:     evidenceOne,
						OverallState: 0,
					}.Build(),
				},
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{},
			}.Build(),
		},
	}

	testResults := map[string]*compliance.ComplianceStandardResult{}
	addCheckResultsToResponse(testResults, standardID, checkNameOne, framework.NodeKind, evidenceOne)
	protoassert.MapEqual(s.T(), mockData.ResultMap, testResults)

	return testResults, client, mockData
}

func (s *ComplianceResultsBuilderTestSuite) TestAddEvidence() {
	// Try one result
	testResults, _, mockData := s.getMockData()

	// Add another result from the same standard
	standardID := "Joseph"
	checkNameTwo := "is really great"
	ce := &storage.ComplianceResultValue_Evidence{}
	ce.SetState(0)
	ce.SetMessage("def")
	evidenceTwo := []*storage.ComplianceResultValue_Evidence{
		ce,
	}
	crv := &storage.ComplianceResultValue{}
	crv.SetEvidence(evidenceTwo)
	crv.SetOverallState(0)
	mockData.ResultMap[standardID].GetNodeCheckResults()[checkNameTwo] = crv
	addCheckResultsToResponse(testResults, standardID, checkNameTwo, framework.NodeKind, evidenceTwo)
	protoassert.MapEqual(s.T(), mockData.ResultMap, testResults)

	// Add a result from a different standard
	standardIDTwo := "abababab"
	checkNameThree := "bababababa"
	ce2 := &storage.ComplianceResultValue_Evidence{}
	ce2.SetState(0)
	ce2.SetMessage("ghi")
	evidenceThree := []*storage.ComplianceResultValue_Evidence{
		ce2,
	}
	crv2 := &storage.ComplianceResultValue{}
	crv2.SetEvidence(evidenceThree)
	crv2.SetOverallState(0)
	csr := &compliance.ComplianceStandardResult{}
	csr.SetNodeCheckResults(map[string]*storage.ComplianceResultValue{
		checkNameThree: crv2,
	})
	csr.SetClusterCheckResults(map[string]*storage.ComplianceResultValue{})
	mockData.ResultMap[standardIDTwo] = csr
	addCheckResultsToResponse(testResults, standardIDTwo, checkNameThree, framework.NodeKind, evidenceThree)
	protoassert.MapEqual(s.T(), mockData.ResultMap, testResults)

	// Add a cluster-level result
	checkNameFour := "jkdfdjk"
	ce3 := &storage.ComplianceResultValue_Evidence{}
	ce3.SetState(0)
	ce3.SetMessage("jkl")
	evidenceFour := []*storage.ComplianceResultValue_Evidence{
		ce3,
	}
	crv3 := &storage.ComplianceResultValue{}
	crv3.SetEvidence(evidenceFour)
	crv3.SetOverallState(0)
	mockData.ResultMap[standardIDTwo].SetClusterCheckResults(map[string]*storage.ComplianceResultValue{
		checkNameFour: crv3,
	})
	addCheckResultsToResponse(testResults, standardIDTwo, checkNameFour, framework.ClusterKind, evidenceFour)
	protoassert.MapEqual(s.T(), mockData.ResultMap, testResults)
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
	protoassert.MapEqual(s.T(), mockData.ResultMap, unzippedEvidence.ResultMap)
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "Foo"
}
