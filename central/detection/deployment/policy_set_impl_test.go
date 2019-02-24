package deployment

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/deployment/predicate"
	"github.com/stretchr/testify/suite"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())), nil)

	err := policySet.UpsertPolicy(goodPolicy())
	suite.NoError(err, "insertion should succeed")

	hasMatch := false
	suite.NoError(policySet.ForEach(func(p *storage.Policy, matcher searchbasedpolicies.Matcher, pred predicate.Predicate) error {
		if p.GetId() == "1" {
			hasMatch = true
		}
		return nil
	}))
	suite.True(hasMatch, "policy set should contain a matching policy")
}

func (suite *PolicyTestSuite) TestForOneSucceeds() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())), nil)

	err := policySet.UpsertPolicy(goodPolicy())
	suite.NoError(err, "insertion should succeed")

	err = policySet.ForOne("1", func(p *storage.Policy, m searchbasedpolicies.Matcher, pred predicate.Predicate) error {
		if p.GetId() != "1" {
			return fmt.Errorf("wrong id served")
		}
		return nil
	})
	suite.NoError(err, "for one should succeed since the policy exists")
}

func (suite *PolicyTestSuite) TestForOneFails() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())), nil)

	err := policySet.ForOne("1", func(p *storage.Policy, m searchbasedpolicies.Matcher, pred predicate.Predicate) error {
		return nil
	})
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(mocks.NewMockDataStore(gomock.NewController(suite.T())), nil)

	err := policySet.UpsertPolicy(badPolicy())
	suite.Error(err, "insertion should not succeed since the regex in the policy is bad")

	hasMatch := false
	suite.NoError(policySet.ForEach(func(p *storage.Policy, m searchbasedpolicies.Matcher, pred predicate.Predicate) error {
		if p.GetId() == "1" {
			hasMatch = true
		}
		return nil
	}))
	suite.False(hasMatch, "policy set should not contain a matching policy")
}

func goodPolicy() *storage.Policy {
	return &storage.Policy{
		Id:         "1",
		Name:       "latest",
		Severity:   storage.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Tag: "latest",
			},
			SetPrivileged: &storage.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
}

func badPolicy() *storage.Policy {
	return &storage.Policy{
		Id:         "2",
		Name:       "latest",
		Severity:   storage.Severity_LOW_SEVERITY,
		Categories: []string{"Image Assurance", "Privileges Capabilities"},
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Tag: "^^[/",
			},
			SetPrivileged: &storage.PolicyFields_Privileged{
				Privileged: true,
			},
		},
	}
}
