package detection

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	matcherMocks "github.com/stackrox/rox/pkg/searchbasedpolicies/matcher/mocks"
	"github.com/stretchr/testify/suite"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite

	mockController     *gomock.Controller
	mockMatcherBuilder *matcherMocks.MockBuilder
}

func (suite *PolicyTestSuite) SetupTest() {
	suite.mockController = gomock.NewController(suite.T())
	suite.mockMatcherBuilder = matcherMocks.NewMockBuilder(suite.mockController)
}

func (suite *PolicyTestSuite) TearDownTest() {
	suite.mockController.Finish()
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet(NewLegacyPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(goodPolicy).Return(nil, nil)

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
	policySet := NewPolicySet(NewLegacyPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(goodPolicy).Return(nil, nil)

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
	policySet := NewPolicySet(NewLegacyPolicyCompiler(suite.mockMatcherBuilder))

	err := policySet.ForOne("1", func(compiled CompiledPolicy) error {
		return nil
	})
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(NewLegacyPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(badPolicy).Return(nil, errors.New("cant create legacySearchBasedMatcher"))

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
	Fields: &storage.PolicyFields{
		ImageName: &storage.ImageNamePolicy{
			Tag: "latest",
		},
		SetPrivileged: &storage.PolicyFields_Privileged{
			Privileged: true,
		},
	},
}

var badPolicy = &storage.Policy{
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
