package manager

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	pkgStandards "github.com/stackrox/rox/pkg/compliance/checks/standards"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stretchr/testify/suite"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunTestSuite))
}

type RunTestSuite struct {
	suite.Suite
}

func makeTestRun(testRunID, testStandardID, testStandardName string, testNodes []*storage.Node) *runInstance {
	testStandard := &standards.Standard{
		Standard: metadata.Standard{
			ID:   testStandardID,
			Name: testStandardName,
		},
	}
	testDomain := framework.NewComplianceDomain(
		nil,
		testNodes,
		nil,
		nil,
		nil,
	)
	return createRun(testRunID, testDomain, testStandard)
}

func getCheckNamesForStandardAndTarget(standardID string, target pkgFramework.TargetKind) []string {
	var checkNames []string

	standardChecks := pkgStandards.NodeChecks[standardID]
	for checkName, checkAndMetadata := range standardChecks {
		if checkAndMetadata.Metadata.TargetKind != target {
			continue
		}
		checkNames = append(checkNames, checkName)
	}

	return checkNames
}

func (s *RunTestSuite) TestFoldNodeResults() {
	testNodeName := "TestNodeName"
	testNodeID := "TestNodeID"
	testStandardID := "TestStandardID"
	testStandardName := "TestStandardName"
	testNodeCheckID := "TestCheckID"
	testClusterCheckID := "TestClusterCheckID"
	testNodes := []*storage.Node{
		{
			Id:   testNodeID,
			Name: testNodeName,
		},
	}

	testRun := makeTestRun("testRun", testStandardID, testStandardName, testNodes)

	testRunData, err := framework.NewComplianceRun(
		framework.NewCheckFromFunc(framework.CheckMetadata{
			ID:    testNodeCheckID,
			Scope: pkgFramework.NodeKind,
		}, nil),
		framework.NewCheckFromFunc(framework.CheckMetadata{
			ID:    testClusterCheckID,
			Scope: pkgFramework.ClusterKind,
		}, nil),
	)
	s.Require().NoError(err)

	expectedNodeResults := &storage.ComplianceResultValue{
		Evidence: []*storage.ComplianceResultValue_Evidence{
			{
				State:   1,
				Message: "Joseph Rules",
			},
		},
		OverallState: 1,
	}
	expectedClusterResults := &storage.ComplianceResultValue{
		Evidence: []*storage.ComplianceResultValue_Evidence{
			{
				State:   1,
				Message: "Joseph is the best",
			},
		},
		OverallState: 1,
	}
	testNodeResults := map[string]map[string]*compliance.ComplianceStandardResult{
		testNodeName: {
			testStandardID: {
				NodeCheckResults: map[string]*storage.ComplianceResultValue{
					testNodeCheckID: expectedNodeResults,
				},
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{
					testClusterCheckID: expectedClusterResults,
				},
			},
		},
	}
	expectedNodeRunResults := &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			testNodeCheckID: expectedNodeResults,
		},
	}
	expectedClusterRunResults := &storage.ComplianceRunResults_EntityResults{
		ControlResults: map[string]*storage.ComplianceResultValue{
			testClusterCheckID: expectedClusterResults,
		},
	}

	complianceRunResults := testRun.collectResults(testRunData, testNodeResults)
	s.Require().Contains(complianceRunResults.NodeResults, testNodeID)
	s.Equal(expectedNodeRunResults, complianceRunResults.NodeResults[testNodeID])

	s.Equal(expectedClusterRunResults, complianceRunResults.GetClusterResults())
}

func (s *RunTestSuite) TestNoteMissing() {
	clusterResults := make(map[string]*storage.ComplianceResultValue)
	testRun := makeTestRun("testRun", pkgStandards.CISKubernetes, pkgStandards.CISKubernetes, nil)

	testRun.noteMissingNodeClusterChecks(clusterResults)

	clusterCheckNames := getCheckNamesForStandardAndTarget(pkgStandards.CISKubernetes, pkgFramework.ClusterKind)

	// We must have a result for each cluster-level check
	s.Require().Len(clusterResults, len(clusterCheckNames))
	// Each cluster-level check must have a result and that result must be a note
	for _, checkName := range clusterCheckNames {
		s.Require().Contains(clusterResults, checkName)
		checkResult := clusterResults[checkName]
		s.Equal(storage.ComplianceState_COMPLIANCE_STATE_NOTE, checkResult.GetOverallState())
		s.Len(checkResult.GetEvidence(), 1)
		for _, evidence := range checkResult.GetEvidence() {
			s.Equal(storage.ComplianceState_COMPLIANCE_STATE_NOTE, evidence.GetState())
		}
	}
}

func (s *RunTestSuite) TestNoteDoesNotReplace() {
	existingResultName := pkgStandards.CISKubernetes + ":1_2_32"
	existingResult := &storage.ComplianceResultValue{
		Evidence: []*storage.ComplianceResultValue_Evidence{
			{
				State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
				Message: "Some successful test",
			},
		},
		OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
	}
	clusterResults := map[string]*storage.ComplianceResultValue{
		existingResultName: existingResult,
	}
	testRun := makeTestRun("testRun", pkgStandards.CISKubernetes, pkgStandards.CISKubernetes, nil)

	testRun.noteMissingNodeClusterChecks(clusterResults)

	// The existing result must not have changed
	s.Require().Contains(clusterResults, existingResultName)
	returnedResult := clusterResults[existingResultName]
	s.Equal(existingResult, returnedResult)
}

func (s *RunTestSuite) TestMergesMultipleClusterResults() {
	testNodeOne := "TestNodeOne"
	testNodeTwo := "TestNodeTwo"
	testNodes := []*storage.Node{
		{
			Id:   testNodeOne,
			Name: testNodeOne,
		},
		{
			Id:   testNodeTwo,
			Name: testNodeTwo,
		},
	}
	evidenceOne := &storage.ComplianceResultValue_Evidence{
		State:   storage.ComplianceState_COMPLIANCE_STATE_NOTE,
		Message: "Test One",
	}
	evidenceTwo := &storage.ComplianceResultValue_Evidence{
		State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
		Message: "Test Two",
	}
	testName := "test"
	testNodeResults := map[string]map[string]*compliance.ComplianceStandardResult{
		testNodeOne: {
			pkgStandards.CISKubernetes: &compliance.ComplianceStandardResult{
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{
					testName: {
						Evidence: []*storage.ComplianceResultValue_Evidence{
							evidenceOne,
						},
						OverallState: evidenceOne.State,
					},
				},
			},
		},
		testNodeTwo: {
			pkgStandards.CISKubernetes: &compliance.ComplianceStandardResult{
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{
					testName: {
						Evidence: []*storage.ComplianceResultValue_Evidence{
							evidenceTwo,
						},
						OverallState: evidenceTwo.State,
					},
				},
			},
		},
	}
	testRunData, err := framework.NewComplianceRun(
		framework.NewCheckFromFunc(framework.CheckMetadata{
			ID:    testName,
			Scope: pkgFramework.ClusterKind,
		}, nil),
	)
	s.Require().NoError(err)

	testRun := makeTestRun("testRun", pkgStandards.CISKubernetes, pkgStandards.CISKubernetes, testNodes)
	testResults := testRun.collectResults(testRunData, testNodeResults)

	clusterResults := testResults.GetClusterResults().GetControlResults()
	s.Require().NotNil(clusterResults)
	s.Require().Contains(clusterResults, testName)
	testResult := clusterResults[testName]
	s.Equal(storage.ComplianceState_COMPLIANCE_STATE_SUCCESS, testResult.GetOverallState())
	s.Len(testResult.GetEvidence(), 2)
	s.Contains(testResult.GetEvidence(), evidenceOne)
	s.Contains(testResult.GetEvidence(), evidenceTwo)
}
