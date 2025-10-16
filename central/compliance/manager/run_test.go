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
	"github.com/stackrox/rox/pkg/protoassert"
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
	node := &storage.Node{}
	node.SetId(testNodeID)
	node.SetName(testNodeName)
	testNodes := []*storage.Node{
		node,
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

	ce := &storage.ComplianceResultValue_Evidence{}
	ce.SetState(1)
	ce.SetMessage("Joseph Rules")
	expectedNodeResults := &storage.ComplianceResultValue{}
	expectedNodeResults.SetEvidence([]*storage.ComplianceResultValue_Evidence{
		ce,
	})
	expectedNodeResults.SetOverallState(1)
	ce2 := &storage.ComplianceResultValue_Evidence{}
	ce2.SetState(1)
	ce2.SetMessage("Joseph is the best")
	expectedClusterResults := &storage.ComplianceResultValue{}
	expectedClusterResults.SetEvidence([]*storage.ComplianceResultValue_Evidence{
		ce2,
	})
	expectedClusterResults.SetOverallState(1)
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
	expectedNodeRunResults := &storage.ComplianceRunResults_EntityResults{}
	expectedNodeRunResults.SetControlResults(map[string]*storage.ComplianceResultValue{
		testNodeCheckID: expectedNodeResults,
	})
	expectedClusterRunResults := &storage.ComplianceRunResults_EntityResults{}
	expectedClusterRunResults.SetControlResults(map[string]*storage.ComplianceResultValue{
		testClusterCheckID: expectedClusterResults,
	})

	complianceRunResults := testRun.collectResults(testRunData, testNodeResults)
	s.Require().Contains(complianceRunResults.GetNodeResults(), testNodeID)
	protoassert.Equal(s.T(), expectedNodeRunResults, complianceRunResults.GetNodeResults()[testNodeID])

	protoassert.Equal(s.T(), expectedClusterRunResults, complianceRunResults.GetClusterResults())
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
	ce := &storage.ComplianceResultValue_Evidence{}
	ce.SetState(storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
	ce.SetMessage("Some successful test")
	existingResult := &storage.ComplianceResultValue{}
	existingResult.SetEvidence([]*storage.ComplianceResultValue_Evidence{
		ce,
	})
	existingResult.SetOverallState(storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
	clusterResults := map[string]*storage.ComplianceResultValue{
		existingResultName: existingResult,
	}
	testRun := makeTestRun("testRun", pkgStandards.CISKubernetes, pkgStandards.CISKubernetes, nil)

	testRun.noteMissingNodeClusterChecks(clusterResults)

	// The existing result must not have changed
	s.Require().Contains(clusterResults, existingResultName)
	returnedResult := clusterResults[existingResultName]
	protoassert.Equal(s.T(), existingResult, returnedResult)
}

func (s *RunTestSuite) TestMergesMultipleClusterResults() {
	testNodeOne := "TestNodeOne"
	testNodeTwo := "TestNodeTwo"
	node := &storage.Node{}
	node.SetId(testNodeOne)
	node.SetName(testNodeOne)
	node2 := &storage.Node{}
	node2.SetId(testNodeTwo)
	node2.SetName(testNodeTwo)
	testNodes := []*storage.Node{
		node,
		node2,
	}
	evidenceOne := &storage.ComplianceResultValue_Evidence{}
	evidenceOne.SetState(storage.ComplianceState_COMPLIANCE_STATE_NOTE)
	evidenceOne.SetMessage("Test One")
	evidenceTwo := &storage.ComplianceResultValue_Evidence{}
	evidenceTwo.SetState(storage.ComplianceState_COMPLIANCE_STATE_SUCCESS)
	evidenceTwo.SetMessage("Test Two")
	testName := "test"
	testNodeResults := map[string]map[string]*compliance.ComplianceStandardResult{
		testNodeOne: {
			pkgStandards.CISKubernetes: compliance.ComplianceStandardResult_builder{
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{
					testName: storage.ComplianceResultValue_builder{
						Evidence: []*storage.ComplianceResultValue_Evidence{
							evidenceOne,
						},
						OverallState: evidenceOne.GetState(),
					}.Build(),
				},
			}.Build(),
		},
		testNodeTwo: {
			pkgStandards.CISKubernetes: compliance.ComplianceStandardResult_builder{
				ClusterCheckResults: map[string]*storage.ComplianceResultValue{
					testName: storage.ComplianceResultValue_builder{
						Evidence: []*storage.ComplianceResultValue_Evidence{
							evidenceTwo,
						},
						OverallState: evidenceTwo.GetState(),
					}.Build(),
				},
			}.Build(),
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
	protoassert.SliceContains(s.T(), testResult.GetEvidence(), evidenceOne)
	protoassert.SliceContains(s.T(), testResult.GetEvidence(), evidenceTwo)
}
