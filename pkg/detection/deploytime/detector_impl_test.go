package deploytime

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestDeploytimeDetector(t *testing.T) {
	suite.Run(t, new(DeploytimeDetectorTestSuite))
}

type DeploytimeDetectorTestSuite struct {
	suite.Suite
}

func (s *DeploytimeDetectorTestSuite) TestDeploytimeCVEPolicy() {
	policySet := detection.NewPolicySet()

	err := policySet.UpsertPolicy(s.getCVEPolicy())
	s.NoError(err, "upsert policy should succeed")

	d := NewDetector(policySet)

	dep := fixtures.GetDeployment()
	images := fixtures.DeploymentImages()
	alerts, err := d.Detect(DetectionContext{}, dep, images)

	s.NoError(err)
	s.NotNil(alerts)
	j, _ := json.Marshal(alerts[0])
	fmt.Printf("%+v\n", alerts[0])
	fmt.Println(string(j))
}

func (s *DeploytimeDetectorTestSuite) getCVEPolicy() *storage.Policy {
	return policyversion.MustEnsureConverted(&storage.Policy{
		Id:            "9dc8b85e-7b35-4423-847b-165cd9b92fc7",
		PolicyVersion: "1.1",
		Name:          "TEST-CVE_POLICY",
		Severity:      storage.Severity_LOW_SEVERITY,
		Categories:    []string{"test"},
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "CVE",
						Negate:    false,
						Values:    []*storage.PolicyValue{{Value: "cve"}},
					},
				},
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
	})
}
