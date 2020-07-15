package manager

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}

type RunTestSuite struct {
	suite.Suite
}

func (s *RunTestSuite) TestFold() {
	testNodeName := "TestNodeName"
	testNodeID := "TestNodeID"
	testStandardID := "TestStandardID"
	testStandardName := "TestStandardName"
	testCheckID := "TestCheckID"
	testStandard := &standards.Standard{
		Standard: metadata.Standard{
			ID:   testStandardID,
			Name: testStandardName,
		},
	}
	testDomain := framework.NewComplianceDomain(
		nil,
		[]*storage.Node{
			{
				Id:   testNodeID,
				Name: testNodeName,
			},
		},
		nil,
		nil,
	)
	testRun := createRun("testRun", testDomain, testStandard)

	complianceRunResults := &storage.ComplianceRunResults{
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"unrelated results": {},
		},
	}
	expectedNodeResults := &storage.ComplianceResultValue{
		Evidence: []*storage.ComplianceResultValue_Evidence{
			{
				State:   0,
				Message: "Joseph Rules",
			},
		},
		OverallState: 0,
	}
	testNodeResults := map[string]map[string]*compliance.ComplianceStandardResult{
		testNodeName: {
			testStandardID: {
				CheckResults: map[string]*storage.ComplianceResultValue{
					testCheckID: expectedNodeResults,
				},
			},
		},
	}
	expectedRunResults := &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			testCheckID: expectedNodeResults,
		},
	}

	testRun.foldNodeResults(complianceRunResults, testNodeResults)
	s.Contains(complianceRunResults.NodeResults, "unrelated results")
	s.Require().Contains(complianceRunResults.NodeResults, testNodeID)
	s.Equal(expectedRunResults, complianceRunResults.NodeResults[testNodeID])
}
