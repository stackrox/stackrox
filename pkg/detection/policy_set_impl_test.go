package detection

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite

	mockController *gomock.Controller
}

func (suite *PolicyTestSuite) SetupTest() {
	suite.mockController = gomock.NewController(suite.T())
}

func (suite *PolicyTestSuite) TearDownTest() {
	suite.mockController.Finish()
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet()

	err := policySet.UpsertPolicy(goodPolicy)
	suite.NoError(err, "insertion should succeed")

	hasMatch := false
	suite.NoError(policySet.ForEach(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() == "1" {
			hasMatch = true
		}
		return nil
	}))
	suite.True(hasMatch, "policy set should contain a matching policy")
}

func (suite *PolicyTestSuite) TestForOneSucceeds() {
	policySet := NewPolicySet()

	err := policySet.UpsertPolicy(goodPolicy)
	suite.NoError(err, "insertion should succeed")

	err = policySet.ForOne("1", func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() != "1" {
			return errors.New("wrong id served")
		}
		return nil
	})
	suite.NoError(err, "for one should succeed since the policy exists")
}

func (suite *PolicyTestSuite) TestForOneFails() {
	policySet := NewPolicySet()

	err := policySet.ForOne("1", func(compiled CompiledPolicy) error {
		return nil
	})
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet()

	err := policySet.UpsertPolicy(badPolicy)
	suite.Error(err, "insertion should not succeed since the compile is set to fail")

	hasMatch := false
	suite.NoError(policySet.ForEach(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() == "1" {
			hasMatch = true
		}
		return nil
	}))
	suite.False(hasMatch, "policy set should not contain a matching policy")
}

var goodPolicy = &storage.Policy{
	Id:         "1",
	Name:       "latest",
	Severity:   storage.Severity_LOW_SEVERITY,
	Categories: []string{"Image Assurance", "Privileges Capabilities"},
	PolicySections: []*storage.PolicySection{
		{
			SectionName: "section-1",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: fieldnames.ImageTag,
					Values: []*storage.PolicyValue{
						{
							Value: "latest",
						},
					},
				},
				{
					FieldName: fieldnames.PrivilegedContainer,
					Values: []*storage.PolicyValue{
						{
							Value: "true",
						},
					},
				},
			},
		},
	},
	PolicyVersion:   "1.1",
	LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
}

var badPolicy = &storage.Policy{
	Id:         "2",
	Name:       "latest",
	Severity:   storage.Severity_LOW_SEVERITY,
	Categories: []string{"Image Assurance", "Privileges Capabilities"},
	PolicySections: []*storage.PolicySection{
		{
			SectionName: "section-1",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: fieldnames.ImageTag,
					Values: []*storage.PolicyValue{
						{
							Value: "^^[/",
						},
					},
				},
				{
					FieldName: fieldnames.PrivilegedContainer,
					Values: []*storage.PolicyValue{
						{
							Value: "true",
						},
					},
				},
			},
		},
	},
	PolicyVersion:   "1.1",
	LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
}
