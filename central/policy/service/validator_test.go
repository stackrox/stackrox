package service

import (
	"fmt"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestPolicyValidator(t *testing.T) {
	suite.Run(t, new(PolicyValidatorTestSuite))
}

type PolicyValidatorTestSuite struct {
	suite.Suite
	validator *policyValidator
	nStorage  *notifierMocks.Store
	cStorage  *clusterMocks.DataStore
}

func (suite *PolicyValidatorTestSuite) SetupTest() {
	suite.nStorage = &notifierMocks.Store{}
	suite.cStorage = &clusterMocks.DataStore{}
	suite.validator = newPolicyValidator(suite.nStorage, suite.cStorage)
}

func (suite *PolicyValidatorTestSuite) TestValidatesName() {
	policy := &v1.Policy{
		Name: "Robert",
	}
	err := suite.validator.validateName(policy)
	suite.NoError(err, "\"Robert\" should be a valid name")

	policy = &v1.Policy{
		Name: "Jim-Bob",
	}
	err = suite.validator.validateName(policy)
	suite.NoError(err, "\"Jim-Bob\" should be a valid name")

	policy = &v1.Policy{
		Name: "Jimmy_John",
	}
	err = suite.validator.validateName(policy)
	suite.NoError(err, "\"Jimmy_John\" should be a valid name")

	policy = &v1.Policy{
		Name: "",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "a name should be required")

	policy = &v1.Policy{
		Name: "Rob",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "names that are too short should not be supported")

	policy = &v1.Policy{
		Name: "RobertIsTheCoolestDudeEverToLiveUnlessYouCountMrTBecauseHeIsEvenDoper",
	}
	err = suite.validator.validateName(policy)
	suite.Error(err, "names that are more than 64 chars are not supported")

	policy = &v1.Policy{
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
			policy := &v1.Policy{
				Fields: &v1.PolicyFields{
					AddCapabilities:  c.adds,
					DropCapabilities: c.drops,
				},
			}
			assert.Equal(t, c.expectedError, suite.validator.validateCapabilities(policy) != nil)
		})
	}
}

func (suite *PolicyValidatorTestSuite) TestValidateDescription() {
	policy := &v1.Policy{
		Description: "",
	}
	err := suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions are not required")

	policy = &v1.Policy{
		Description: "Yo",
	}
	err = suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions can be as short as they like")

	policy = &v1.Policy{
		Description: "This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!",
	}
	err = suite.validator.validateDescription(policy)
	suite.NoError(err, "descriptions should take the form of a sentence")

	policy = &v1.Policy{
		Description: `This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!
			Oh, and I almost forgot that this is also to help the good people of nowhere-ville get back on their 
			feet after that tornado ripped their town to shreds and left them nothing but pineapple and gum.`,
	}
	err = suite.validator.validateDescription(policy)
	suite.Error(err, "descriptions should be no more than 256 chars")

	policy = &v1.Policy{
		Description: "This$Rox",
	}
	err = suite.validator.validateDescription(policy)
	suite.Error(err, "no special characters")
}

func (suite *PolicyValidatorTestSuite) TestValidateLifeCycle() {
	testCases := []struct {
		description string
		p           *v1.Policy
		errExpected bool
	}{
		{
			description: "Build time policy with non-image fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_BUILD_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{Remote: "blah"},
					ContainerResourcePolicy: &v1.ResourcePolicy{
						CpuResourceLimit: &v1.NumericalPolicy{
							Value: 1.0,
						},
					},
				},
			},
			errExpected: true,
		},
		{
			description: "Build time policy with no image fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_BUILD_TIME,
				},
			},
			errExpected: true,
		},
		{
			description: "valid build time",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_BUILD_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
			},
		},
		{
			description: "deploy time with no fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_DEPLOY_TIME,
				},
			},
			errExpected: true,
		},
		{
			description: "deploy time with runtime fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_DEPLOY_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					ProcessPolicy: &v1.ProcessPolicy{Name: "BLAH"},
				},
			},
			errExpected: true,
		},
		{
			description: "Valid deploy time",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_DEPLOY_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &v1.VolumePolicy{
						Name: "Asfasf",
					},
				},
			},
		},
		{
			description: "Run time with no fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_RUN_TIME,
				},
			},
			errExpected: true,
		},
		{
			description: "Run time with only deploy-time fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_RUN_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &v1.VolumePolicy{
						Name: "Asfasf",
					},
				},
			},
			errExpected: true,
		},
		{
			description: "Valid Run time with just process fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_RUN_TIME,
				},
				Fields: &v1.PolicyFields{
					ProcessPolicy: &v1.ProcessPolicy{Name: "asfasfaa"},
				},
			},
		},
		{
			description: "Valid Run time with all sorts of fields",
			p: &v1.Policy{
				LifecycleStages: []v1.LifecycleStage{
					v1.LifecycleStage_RUN_TIME,
				},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					VolumePolicy: &v1.VolumePolicy{
						Name: "Asfasf",
					},
					ProcessPolicy: &v1.ProcessPolicy{Name: "asfasfaa"},
				},
			},
		},
	}

	for _, c := range testCases {
		suite.T().Run(c.description, func(t *testing.T) {
			c.p.Name = "BLAHBLAH"
			err := suite.validator.validateCompilableForLifecycle(c.p)
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (suite *PolicyValidatorTestSuite) TestValidateSeverity() {
	policy := &v1.Policy{
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := suite.validator.validateSeverity(policy)
	suite.NoError(err, "severity should pass when set")

	policy = &v1.Policy{
		Severity: v1.Severity_UNSET_SEVERITY,
	}
	err = suite.validator.validateSeverity(policy)
	suite.Error(err, "severity should fail when not set")
}

func (suite *PolicyValidatorTestSuite) TestValidateCategories() {
	policy := &v1.Policy{}
	err := suite.validator.validateCategories(policy)
	suite.Error(err, "at least one category should be required")

	policy = &v1.Policy{
		Categories: []string{
			"cat1",
			"cat2",
			"cat1",
		},
	}
	err = suite.validator.validateCategories(policy)
	suite.Error(err, "duplicate categories should fail")

	policy = &v1.Policy{
		Categories: []string{
			"cat1",
			"cat2",
		},
	}
	err = suite.validator.validateCategories(policy)
	suite.NoError(err, "valid categories should not fail")
}

func (suite *PolicyValidatorTestSuite) TestValidateNotifiers() {
	policy := &v1.Policy{
		Notifiers: []string{
			"id1",
		},
	}
	suite.nStorage.On("GetNotifier", "id1").Return((*v1.Notifier)(nil), true, nil)
	err := suite.validator.validateNotifiers(policy)
	suite.NoError(err, "severity should pass when set")
	suite.nStorage.AssertExpectations(suite.T())

	policy = &v1.Policy{
		Notifiers: []string{
			"id2",
		},
	}
	suite.nStorage.On("GetNotifier", "id2").Return((*v1.Notifier)(nil), false, nil)
	err = suite.validator.validateNotifiers(policy)
	suite.Error(err, "should fail when it does not exist")
	suite.nStorage.AssertExpectations(suite.T())

	policy = &v1.Policy{
		Notifiers: []string{
			"id3",
		},
	}
	suite.nStorage.On("GetNotifier", "id3").Return((*v1.Notifier)(nil), true, fmt.Errorf("oh noes"))
	err = suite.validator.validateNotifiers(policy)
	suite.Error(err, "should fail when an error is thrown")
	suite.nStorage.AssertExpectations(suite.T())
}

func (suite *PolicyValidatorTestSuite) TestValidateScopes() {
	policy := &v1.Policy{}
	err := suite.validator.validateScopes(policy)
	suite.NoError(err, "scopes should not be required")

	scope := &v1.Scope{
		Cluster: "cluster1",
	}
	policy = &v1.Policy{
		Scope: []*v1.Scope{
			scope,
		},
	}
	suite.cStorage.On("GetCluster", "cluster1").Return((*v1.Cluster)(nil), true, nil)
	err = suite.validator.validateScopes(policy)
	suite.NoError(err, "valid scope definition")
	suite.cStorage.AssertExpectations(suite.T())

	scope = &v1.Scope{}
	policy = &v1.Policy{
		Scope: []*v1.Scope{
			scope,
		},
	}
	err = suite.validator.validateScopes(policy)
	suite.NoError(err, "scopes with no cluster should be allowed")

	scope = &v1.Scope{
		Cluster: "cluster2",
	}
	policy = &v1.Policy{
		Scope: []*v1.Scope{
			scope,
		},
	}
	suite.cStorage.On("GetCluster", "cluster2").Return((*v1.Cluster)(nil), false, nil)
	err = suite.validator.validateScopes(policy)
	suite.Error(err, "scopes whose clusters can't be found should fail")
	suite.cStorage.AssertExpectations(suite.T())

	scope = &v1.Scope{
		Cluster: "cluster3",
	}
	policy = &v1.Policy{
		Scope: []*v1.Scope{
			scope,
		},
	}
	suite.cStorage.On("GetCluster", "cluster3").Return((*v1.Cluster)(nil), true, fmt.Errorf("dang boi"))
	err = suite.validator.validateScopes(policy)
	suite.Error(err, "scopes whose clusters fail to be found should fail")
	suite.cStorage.AssertExpectations(suite.T())
}

func (suite *PolicyValidatorTestSuite) TestValidateWhitelists() {
	policy := &v1.Policy{}
	err := suite.validator.validateWhitelists(policy)
	suite.NoError(err, "whitelists should not be required")

	deployment := &v1.Whitelist_Deployment{
		Name: "that phat cluster",
	}
	deploymentWhitelist := &v1.Whitelist{
		Deployment: deployment,
	}
	policy = &v1.Policy{
		Whitelists: []*v1.Whitelist{
			deploymentWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.NoError(err, "valid to whitelist by deployment name")

	emptyWhitelist := &v1.Whitelist{}
	policy = &v1.Policy{
		Whitelists: []*v1.Whitelist{
			emptyWhitelist,
		},
	}
	err = suite.validator.validateWhitelists(policy)
	suite.Error(err, "whitelist requires either container or deployment configuration")

}
