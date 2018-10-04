package image

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())))

	err := policySet.UpsertPolicy(goodPolicy())
	suite.NoError(err, "insertion should succeed")

	hasMatch := false
	policySet.ForEach(func(p *v1.Policy, m searchbasedpolicies.Matcher) error {
		if p.GetId() == "1" {
			hasMatch = true
		}
		return nil
	})
	suite.True(hasMatch, "policy set should contain a matching policy")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())))

	err := policySet.UpsertPolicy(badPolicy())
	suite.Error(err, "insertion should not succeed since the regex in the policy is bad")

	hasMatch := false
	policySet.ForEach(func(p *v1.Policy, m searchbasedpolicies.Matcher) error {
		if p.GetId() == "1" {
			hasMatch = true
		}
		return nil
	})
	suite.False(hasMatch, "policy set should not contain a matching policy")
}

func goodPolicy() *v1.Policy {
	return &v1.Policy{
		Id:         "1",
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "latest",
			},
		},
	}
}

func badPolicy() *v1.Policy {
	return &v1.Policy{
		Id:         "2",
		Name:       "latest",
		Severity:   v1.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &v1.PolicyFields{
			ImageName: &v1.ImageNamePolicy{
				Tag: "^^[/",
			},
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
}
