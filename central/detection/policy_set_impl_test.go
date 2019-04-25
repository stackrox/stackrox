package detection

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	datastoreMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	matcherMocks "github.com/stackrox/rox/central/searchbasedpolicies/matcher/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestPolicySet(t *testing.T) {
	suite.Run(t, new(PolicyTestSuite))
}

type PolicyTestSuite struct {
	suite.Suite

	mockController     *gomock.Controller
	mockDataStore      *datastoreMocks.MockDataStore
	mockMatcherBuilder *matcherMocks.MockBuilder
}

func (suite *PolicyTestSuite) SetupTest() {
	suite.mockController = gomock.NewController(suite.T())
	suite.mockDataStore = datastoreMocks.NewMockDataStore(suite.mockController)
	suite.mockMatcherBuilder = matcherMocks.NewMockBuilder(suite.mockController)
}

func (suite *PolicyTestSuite) TearDownTest() {
	suite.mockController.Finish()
}

func (suite *PolicyTestSuite) TestAddsCompilable() {
	policySet := NewPolicySet(suite.mockDataStore, NewPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(goodPolicy).Return(nil, nil)

	err := policySet.UpsertPolicy(goodPolicy)
	suite.NoError(err, "insertion should succeed")

	hasMatch := false
	suite.NoError(policySet.ForEach(FunctionAsExecutor(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() == "1" {
			hasMatch = true
		}
		return nil
	})))
	suite.True(hasMatch, "policy set should contain a matching policy")
}

func (suite *PolicyTestSuite) TestForOneSucceeds() {
	policySet := NewPolicySet(suite.mockDataStore, NewPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(goodPolicy).Return(nil, nil)

	err := policySet.UpsertPolicy(goodPolicy)
	suite.NoError(err, "insertion should succeed")

	err = policySet.ForOne("1", FunctionAsExecutor(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() != "1" {
			return errors.New("wrong id served")
		}
		return nil
	}))
	suite.NoError(err, "for one should succeed since the policy exists")
}

func (suite *PolicyTestSuite) TestForOneFails() {
	policySet := NewPolicySet(suite.mockDataStore, NewPolicyCompiler(suite.mockMatcherBuilder))

	err := policySet.ForOne("1", FunctionAsExecutor(func(compiled CompiledPolicy) error {
		return nil
	}))
	suite.Error(err, "for one should fail since no policies exist")
}

func (suite *PolicyTestSuite) TestThrowsErrorForNotCompilable() {
	policySet := NewPolicySet(suite.mockDataStore, NewPolicyCompiler(suite.mockMatcherBuilder))

	suite.mockMatcherBuilder.EXPECT().ForPolicy(badPolicy).Return(nil, errors.New("cant create matcher"))

	err := policySet.UpsertPolicy(badPolicy)
	suite.Error(err, "insertion should not succeed since the compile is set to fail")

	hasMatch := false
	suite.NoError(policySet.ForEach(FunctionAsExecutor(func(compiled CompiledPolicy) error {
		if compiled.Policy().GetId() == "1" {
			hasMatch = true
		}
		return nil
	})))
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
