package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	matcherMocks "github.com/stackrox/rox/central/searchbasedpolicies/matcher/mocks"
	sbpMocks "github.com/stackrox/rox/central/searchbasedpolicies/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestPolicyValidator(t *testing.T) {
	suite.Run(t, new(PolicyValidatorTestSuite))
}

type PolicyValidatorTestSuite struct {
	suite.Suite
	requestContext context.Context
	validator      *policyValidator
	nStorage       *notifierMocks.MockDataStore
	cStorage       *clusterMocks.MockDataStore
	matcherBuilder *matcherMocks.MockBuilder

	mockCtrl *gomock.Controller
}

func (suite *PolicyValidatorTestSuite) SetupTest() {
	// Since all the datastores underneath are mocked, the context of the request doesns't need any permissions.
	suite.requestContext = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.nStorage = notifierMocks.NewMockDataStore(suite.mockCtrl)
	suite.cStorage = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.matcherBuilder = matcherMocks.NewMockBuilder(suite.mockCtrl)

	suite.validator = newPolicyValidator(suite.nStorage, suite.cStorage, suite.matcherBuilder, suite.matcherBuilder)
}

func (suite *PolicyValidatorTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PolicyValidatorTestSuite) TestValidatesName() {
	policy := &storage.Policy{
		Name: "Robert",
	}
	err := suite.validator.validateName(policy)
	suite.NoError(err, "\"Robert\" should be a valid name")

	policy = &storage.Policy{
		Name: "Jim-Bob",
	}
	err = suite.validator.validateName(policy)
	suite.NoError(err, "\"Jim-Bob\" should be a valid name")

	policy = &storage.Policy{
		Name: "Jimmy_John",
	}
	err = suite.validator.validateName(policy)
	suite.NoError(err, "\"Jimmy_John\" should be a valid name")

	policy = &storage.Policy{
		Name: "",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "a name should be required")

	policy = &storage.Policy{
		Name: "Rob",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "names that are too short should not be supported")

	policy = &storage.Policy{
		Name: "RobertIsTheCoolestDudeEverToLiveUnlessYouCountMrTBecauseHeIsEvenDoper",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "names that are more than 64 chars are not supported")

	policy = &storage.Policy{
		Name: "Rob$",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "special characters should not be supported")
}

func (suite *PolicyValidatorTestSuite) TestsValidateCapabilities() {

	cases := []struct {
		name          string
		adds          []string
		drops         []string
		expectedError bool
	}{
		{
			name:          "no values",
			expectedError: false,
		},
		{
			name:          "adds only",
			adds:          []string{"hi"},
			expectedError: false,
		},
		{
			name:          "drops only",
			drops:         []string{"hi"},
			expectedError: false,
		},
		{
			name:          "different adds and drops",
			adds:          []string{"hello"},
			drops:         []string{"hey"},
			expectedError: false,
		},
		{
			name:          "same adds and drops",
			adds:          []string{"hello"},
			drops:         []string{"hello"},
			expectedError: true,
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			policy := &storage.Policy{
				Fields: &storage.PolicyFields{
					AddCapabilities:  c.adds,
					DropCapabilities: c.drops,
				},
			}
			assert.Equal(t, c.expectedError, suite.validator.validateCapabilities(policy) != nil)
		})
	}
}

func (suite *PolicyValidatorTestSuite) TestValidateDescription() {
	policy := &storage.Policy{
		Description: "",
	}
	err := suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions are not required")

	policy = &storage.Policy{
		Description: "Yo",
	}
	err = suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions can be as short as they like")

	policy = &storage.Policy{
		Description: "This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!",
	}
	err = suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions should take the form of a sentence")

	policy = &storage.Policy{
		Description: `This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!
			Oh, and I almost forgot that this is also to help the good people of nowhere-ville get back on their 
			feet after that tornado ripped their town to shreds and left them nothing but pineapple and gum.`,
	}
	err = suite.validator.validateDescription(policy)
	suite.Error(err, "descriptions should be no more than 256 chars")

	policy = &storage.Policy{
		Description: "This$Rox",
	}
	err = suite.validator.validateDescription(policy)
	suite.Error(err, "no special characters")
}

func (suite *PolicyValidatorTestSuite) TestValidateLifeCycle() {
	testCases := []struct {
		description string
		p           *storage.Policy
		errExpected bool
	}{
		{
			description: "Build time policy with non-image fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{Remote: "blah"},
					ContainerResourcePolicy: &storage.ResourcePolicy{
						CpuResourceLimit: &storage.NumericalPolicy{
							Value: 1.0,
						},
					},
				},
			},
			errExpected: true,
		},
		{
			description: "Build time policy with no image fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
				},
			},
			errExpected: true,
		},
		{
			description: "valid build time",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
				},
			},
		},
		{
			description: "deploy time with no fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_DEPLOY,
				},
			},
			errExpected: true,
		},
		{
			description: "deploy time with runtime fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_DEPLOY,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					ProcessPolicy: &storage.ProcessPolicy{Name: "BLAH"},
				},
			},
			errExpected: true,
		},
		{
			description: "Valid deploy time",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_DEPLOY,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
				},
			},
		},
		{
			description: "Run time with no fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
			},
			errExpected: true,
		},
		{
			description: "Run time with only deploy-time fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
				},
			},
			errExpected: true,
		},
		{
			description: "Valid Run time with just process fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
				Fields: &storage.PolicyFields{
					ProcessPolicy: &storage.ProcessPolicy{Name: "asfasfaa"},
				},
			},
		},
		{
			description: "Valid Run time with all sorts of fields",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
					ProcessPolicy: &storage.ProcessPolicy{Name: "asfasfaa"},
				},
			},
		},
	}

	for _, c := range testCases {
		suite.T().Run(c.description, func(t *testing.T) {
			c.p.Name = "BLAHBLAH"
			if c.errExpected {
				suite.matcherBuilder.EXPECT().ForPolicy(c.p).Return(nil, errors.New("fail to build matcher"))
			} else {
				suite.matcherBuilder.EXPECT().ForPolicy(c.p).Return(sbpMocks.NewMockMatcher(suite.mockCtrl), nil)
			}

			err := suite.validator.validateCompilableForLifecycle(c.p)
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (suite *PolicyValidatorTestSuite) TestValidateLifeCycleEnforcementCombination() {
	testCases := []struct {
		description  string
		p            *storage.Policy
		expectedSize int
	}{
		{
			description: "Remove invalid enforcement with runtime lifecycle",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
					ProcessPolicy: &storage.ProcessPolicy{Name: "asfasfaa"},
				},
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			},
			expectedSize: 1,
		},
		{
			description: "Remove invalid enforcement with build lifecycle",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
					ProcessPolicy: &storage.ProcessPolicy{Name: "asfasfaa"},
				},
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			},
			expectedSize: 1,
		},
		{
			description: "Remove invalid enforcement with deployment lifecycle",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_DEPLOY,
				},
				Fields: &storage.PolicyFields{
					ImageName: &storage.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &storage.VolumePolicy{
						Name: "Asfasf",
					},
					ProcessPolicy: &storage.ProcessPolicy{Name: "asfasfaa"},
				},
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			},
			expectedSize: 2,
		},
	}

	for _, c := range testCases {
		suite.T().Run(c.description, func(t *testing.T) {
			c.p.Name = "BLAHBLAH"
			suite.validator.removeEnforcementsForMissingLifecycles(c.p)
			assert.Equal(t, len(c.p.EnforcementActions), c.expectedSize, "enforcement size does not match")
		})
	}
}

func (suite *PolicyValidatorTestSuite) TestValidateSeverity() {
	policy := &storage.Policy{
		Severity: storage.Severity_LOW_SEVERITY,
	}
	err := suite.validator.validateSeverity(policy)
	suite.NoError(err, "severity should pass when set")

	policy = &storage.Policy{
		Severity: storage.Severity_UNSET_SEVERITY,
	}
	err = suite.validator.validateSeverity(policy)
	suite.Error(err, "severity should fail when not set")
}

func (suite *PolicyValidatorTestSuite) TestValidateCategories() {
	policy := &storage.Policy{}
	err := suite.validator.validateCategories(policy)
	suite.Error(err, "at least one category should be required")

	policy = &storage.Policy{
		Categories: []string{
			"cat1",
			"cat2",
			"cat1",
		},
	}
	err = suite.validator.validateCategories(policy)
	suite.Error(err, "duplicate categories should fail")

	policy = &storage.Policy{
		Categories: []string{
			"cat1",
			"cat2",
		},
	}
	err = suite.validator.validateCategories(policy)
	suite.NoError(err, "valid categories should not fail")
}

func (suite *PolicyValidatorTestSuite) TestValidateNotifiers() {
	policy := &storage.Policy{
		Notifiers: []string{
			"id1",
		},
	}
	suite.nStorage.EXPECT().GetNotifier(suite.requestContext, "id1").Return((*storage.Notifier)(nil), true, nil)
	err := suite.validator.validateNotifiers(suite.requestContext, policy)
	suite.NoError(err, "severity should pass when set")

	policy = &storage.Policy{
		Notifiers: []string{
			"id2",
		},
	}
	suite.nStorage.EXPECT().GetNotifier(suite.requestContext, "id2").Return((*storage.Notifier)(nil), false, nil)
	err = suite.validator.validateNotifiers(suite.requestContext, policy)
	suite.Error(err, "should fail when it does not exist")

	policy = &storage.Policy{
		Notifiers: []string{
			"id3",
		},
	}
	suite.nStorage.EXPECT().GetNotifier(suite.requestContext, "id3").Return((*storage.Notifier)(nil), true, fmt.Errorf("oh noes"))
	err = suite.validator.validateNotifiers(suite.requestContext, policy)
	suite.Error(err, "should fail when an error is thrown")
}

func (suite *PolicyValidatorTestSuite) TestValidateWhitelists() {
	policy := &storage.Policy{}
	err := suite.validator.validateWhitelists(policy)
	suite.NoError(err, "whitelists should not be required")

	deployment := &storage.Whitelist_Deployment{
		Name: "that phat cluster",
	}
	deploymentWhitelist := &storage.Whitelist{
		Deployment: deployment,
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
		},
		Whitelists: []*storage.Whitelist{
			deploymentWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.NoError(err, "valid to whitelist by deployment name")

	imageWhitelist := &storage.Whitelist{
		Image: &storage.Whitelist_Image{
			Name: "stackrox.io",
		},
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_BUILD,
		},
		Whitelists: []*storage.Whitelist{
			imageWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.NoError(err, "valid to whitelist by image registry")

	policy = &storage.Policy{
		Whitelists: []*storage.Whitelist{
			imageWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.Error(err, "not valid to whitelist by image registry since build time lifecycle isn't present")

	emptyWhitelist := &storage.Whitelist{}
	policy = &storage.Policy{
		Whitelists: []*storage.Whitelist{
			emptyWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.Error(err, "whitelist requires either container or deployment configuration")
}
