package image

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	imageMatcher "github.com/stackrox/rox/pkg/compiledpolicies/image/matcher"
	"github.com/stretchr/testify/suite"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet(&mocks.DataStore{})

	err := policySet.UpsertPolicy(goodPolicy())
	suite.NoError(err, "insertion should succeed")

	hasMatch := false
	policySet.ForEach(func(p *v1.Policy, m imageMatcher.Matcher) error {
		if p.GetId() == "1" {
			hasMatch = true
		}
		return nil
	})
	suite.True(hasMatch, "policy set should contain a matching policy")
}

func (suite *PolicyTestSuite) TestForOneSucceeds() {
	policySet := NewPolicySet(&mocks.DataStore{})

	err := policySet.UpsertPolicy(goodPolicy())
	suite.NoError(err, "insertion should succeed")

	err = policySet.ForOne("1", func(p *v1.Policy, m imageMatcher.Matcher) error {
		if p.GetId() != "1" {
			return fmt.Errorf("wrong id served")
		}
		return nil
	})
	suite.NoError(err, "for one should succeed since the policy exists")
}

func (suite *PolicyTestSuite) TestForOneFails() {
	policySet := NewPolicySet(&mocks.DataStore{})

	err := policySet.ForOne("1", func(p *v1.Policy, m imageMatcher.Matcher) error {
		return nil
	})
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(&mocks.DataStore{})

	err := policySet.UpsertPolicy(badPolicy())
	suite.Error(err, "insertion should not succeed since the regex in the policy is bad")

	hasMatch := false
	policySet.ForEach(func(p *v1.Policy, m imageMatcher.Matcher) error {
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
			SetPrivileged: &v1.PolicyFields_Privileged{
				Privileged: true,
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
