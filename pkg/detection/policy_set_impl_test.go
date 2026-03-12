package detection

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/scopecomp/mocks"
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
	policySet := NewPolicySet(nil, nil)

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
	policySet := NewPolicySet(nil, nil)

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
	policySet := NewPolicySet(nil, nil)

	err := policySet.ForOne("1", func(compiled CompiledPolicy) error {
		return nil
	})
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(nil, nil)

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

func (suite *PolicyTestSuite) TestPolicySetWithMockProviders() {
	// Enable the feature flag for label-based policy scoping
	suite.T().Setenv("ROX_LABEL_BASED_POLICY_SCOPING", "true")

	// Test that PolicySet correctly threads providers through to policy compilation
	mockClusterProvider := mocks.NewMockClusterLabelProvider(suite.mockController)
	mockNamespaceProvider := mocks.NewMockNamespaceLabelProvider(suite.mockController)

	policySet := NewPolicySet(mockClusterProvider, mockNamespaceProvider)

	// Create a policy with cluster label matcher
	policyWithLabels := &storage.Policy{
		Id:       "label-policy",
		Name:     "Test cluster label policy",
		Severity: storage.Severity_HIGH_SEVERITY,
		Scope: []*storage.Scope{
			{
				ClusterLabel: &storage.Scope_Label{
					Key:   "env",
					Value: "prod",
				},
			},
		},
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
				},
			},
		},
		PolicyVersion:   "1.1",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
	}

	err := policySet.UpsertPolicy(policyWithLabels)
	suite.NoError(err, "insertion should succeed")

	// Verify the policy was added
	hasMatch := false
	suite.NoError(policySet.ForEach(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() == "label-policy" {
			hasMatch = true
		}
		return nil
	}))
	suite.True(hasMatch, "policy set should contain the label-based policy")
}
