package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

	mockCtrl *gomock.Controller
}

func (s *PolicyValidatorTestSuite) SetupTest() {
	// Since all the datastores underneath are mocked, the context of the request doesns't need any permissions.
	s.requestContext = context.Background()

	s.mockCtrl = gomock.NewController(s.T())
	s.nStorage = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.cStorage = clusterMocks.NewMockDataStore(s.mockCtrl)
	s.validator = newPolicyValidator(s.nStorage)
}

func (s *PolicyValidatorTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PolicyValidatorTestSuite) TestValidatesName() {
	policy := &storage.Policy{}
	policy.SetName("Robert")
	err := s.validator.validateName(policy)
	s.NoError(err, "\"Robert\" should be a valid name")

	policy = &storage.Policy{}
	policy.SetName("Jim-Bob")
	err = s.validator.validateName(policy)
	s.NoError(err, "\"Jim-Bob\" should be a valid name")

	policy = &storage.Policy{}
	policy.SetName("Jimmy_John")
	err = s.validator.validateName(policy)
	s.NoError(err, "\"Jimmy_John\" should be a valid name")

	policy = &storage.Policy{}
	policy.SetName("")
	err = s.validator.validateName(policy)
	s.Error(err, "a name should be required")

	policy = &storage.Policy{}
	policy.SetName("Rob")
	err = s.validator.validateName(policy)
	s.Error(err, "names that are too short should not be supported")

	policy = &storage.Policy{}
	policy.SetName("RobertIsTheCoolestDudeEverToLiveUnlessYouCountMrTBecauseHeIsEvenDoperHisVanIsSweetAndHisHairIsCoolAndIReallyLikeAllTheGoldChainsHeWears")
	err = s.validator.validateName(policy)
	s.Error(err, "names that are more than 128 chars are not supported")

	policy = &storage.Policy{}
	policy.SetName("Rob$")
	err = s.validator.validateName(policy)
	s.Error(err, "special characters should not be supported")

	policy = &storage.Policy{}
	policy.SetName("  Boo's policy  ")
	err = s.validator.validateName(policy)
	s.NoError(err, "leading and trailing spaces should be trimmed")
	s.Equal("Boo's policy", policy.GetName())
}

func (s *PolicyValidatorTestSuite) TestValidateVersion() {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{
			"Current version should be valid",
			policyversion.CurrentVersion().String(),
			true,
		},
		{
			"No version is no longer valid",
			"",
			false,
		},
		{
			"Version 1 is no longer valid",
			"1",
			false,
		},
		{
			"Invalid version string is not valid",
			"x.y.z",
			false,
		},
		{
			"Non-existent version is not valid",
			"2.0",
			false,
		},
	}
	for _, c := range tests {
		s.T().Run(c.name, func(t *testing.T) {
			policy := &storage.Policy{}
			policy.SetPolicyVersion(c.version)
			err := s.validator.validateVersion(policy)
			if c.valid {
				s.NoError(err, "Version should be valid")
			} else {
				s.Error(err, "Version should be invalid")
			}
		})
	}
}

func (s *PolicyValidatorTestSuite) TestsValidateCapabilities() {

	cases := []struct {
		name          string
		adds          []*storage.PolicyValue
		drops         []*storage.PolicyValue
		expectedError bool
	}{
		{
			name:          "no values",
			expectedError: false,
		},
		{
			name: "adds only",
			adds: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hi",
				}.Build(),
			},
			expectedError: false,
		},
		{
			name: "drops only",
			drops: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hi",
				}.Build(),
			},
			expectedError: false,
		},
		{
			name: "different adds and drops",
			adds: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hey",
				}.Build(),
			},
			drops: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hello",
				}.Build(),
			},
			expectedError: false,
		},
		{
			name: "same adds and drops",
			adds: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hello",
				}.Build(),
			},
			drops: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "hello",
				}.Build(),
			},
			expectedError: true,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			policy := storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						SectionName: "section-1",
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.AddCaps,
								Values:    c.adds,
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.DropCaps,
								Values:    c.drops,
							}.Build(),
						},
					}.Build(),
				},
				PolicyVersion: "1.1",
			}.Build()
			assert.Equal(t, c.expectedError, s.validator.validateCapabilities(policy) != nil)
		})
	}
}

func (s *PolicyValidatorTestSuite) TestValidateDescription() {
	policy := &storage.Policy{}
	policy.SetDescription("")
	err := s.validator.validateDescription(policy)
	s.NoError(err, "descriptions are not required")

	policy = &storage.Policy{}
	policy.SetDescription("Yo")
	err = s.validator.validateDescription(policy)
	s.NoError(err, "descriptions can be as short as they like")

	policy = &storage.Policy{}
	policy.SetDescription("This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!")
	err = s.validator.validateDescription(policy)
	s.NoError(err, "descriptions should take the form of a sentence")

	policy = &storage.Policy{}
	policy.SetDescription(`This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!
			Oh, and I almost forgot that this is also to help the good people of nowhere-ville get back on their
			feet after that tornado ripped their town to shreds and left them nothing but pineapple and gum.  It was the It was
			the best of times, it was the worst of times, it was the age of wisdom, it was the age of foolishness, it was the
			epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of Darkness, it was
			the spring of hope, it was the winter of despair, we had everything before us, we had nothing before us, we were all
			going direct to Heaven, we were all going direct the other way--in short, the period was so far like the present
			period that some of its noisiest authorities insisted on its being received, for good or for evil, in the superlative
			degree of comparison only.`)
	err = s.validator.validateDescription(policy)
	s.Error(err, "descriptions should be no more than 800 chars")

	policy = &storage.Policy{}
	policy.SetDescription("This$Rox")
	err = s.validator.validateDescription(policy)
	s.Error(err, "no special characters")
}

func booleanPolicyWithFields(lifecycleStage storage.LifecycleStage, eventSource storage.EventSource, fieldsToVals map[string]string) *storage.Policy {
	groups := make([]*storage.PolicyGroup, 0, len(fieldsToVals))
	for k, v := range fieldsToVals {
		pv := &storage.PolicyValue{}
		pv.SetValue(v)
		pg := &storage.PolicyGroup{}
		pg.SetFieldName(k)
		pg.SetValues([]*storage.PolicyValue{pv})
		groups = append(groups, pg)
	}
	ps := &storage.PolicySection{}
	ps.SetPolicyGroups(groups)
	policy := &storage.Policy{}
	policy.SetPolicyVersion(policyversion.CurrentVersion().String())
	policy.SetLifecycleStages([]storage.LifecycleStage{lifecycleStage})
	policy.SetEventSource(eventSource)
	policy.SetPolicySections([]*storage.PolicySection{ps})
	return policy
}

func (s *PolicyValidatorTestSuite) TestValidateLifeCycle() {
	testCases := []struct {
		description string
		p           *storage.Policy
		errExpected bool
	}{
		{
			description: "Build time policy with non-image fields",
			p: booleanPolicyWithFields(storage.LifecycleStage_BUILD, storage.EventSource_NOT_APPLICABLE,
				map[string]string{
					fieldnames.ImageRemote:       "blah",
					fieldnames.ContainerCPULimit: "1.0",
				}),
			errExpected: true,
		},
		{
			description: "Build time policy with no image fields",
			p:           booleanPolicyWithFields(storage.LifecycleStage_BUILD, storage.EventSource_NOT_APPLICABLE, nil),
			errExpected: true,
		},
		{
			description: "valid build time",
			p: booleanPolicyWithFields(storage.LifecycleStage_BUILD, storage.EventSource_NOT_APPLICABLE, map[string]string{
				fieldnames.ImageTag: "latest",
			}),
		},
		{
			description: "deploy time with no fields",
			p:           booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE, nil),
			errExpected: true,
		},
		{
			description: "deploy time with runtime fields",
			p: booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE,
				map[string]string{
					fieldnames.ImageTag:    "latest",
					fieldnames.ProcessName: "BLAH",
				}),
			errExpected: true,
		},

		{
			description: "Valid deploy time",
			p: booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE,
				map[string]string{
					fieldnames.ImageTag:   "latest",
					fieldnames.VolumeName: "BLAH",
				}),
		},
		{
			description: "Run time with no fields",
			p:           booleanPolicyWithFields(storage.LifecycleStage_RUNTIME, storage.EventSource_DEPLOYMENT_EVENT, nil),
			errExpected: true,
		},
		{
			description: "Run time with only deploy-time fields",
			p: booleanPolicyWithFields(storage.LifecycleStage_RUNTIME, storage.EventSource_DEPLOYMENT_EVENT,
				map[string]string{
					fieldnames.ImageTag:   "latest",
					fieldnames.VolumeName: "BLAH",
				}),
			errExpected: true,
		},
		{
			description: "Valid Run time with just process fields",
			p: booleanPolicyWithFields(storage.LifecycleStage_RUNTIME, storage.EventSource_DEPLOYMENT_EVENT,
				map[string]string{
					fieldnames.ProcessName: "BLAH",
				}),
		},
		{
			description: "Valid Run time with all sorts of fields",
			p: booleanPolicyWithFields(storage.LifecycleStage_RUNTIME, storage.EventSource_DEPLOYMENT_EVENT,
				map[string]string{
					fieldnames.ProcessName: "PROCESS",
				}),
		},
	}

	for _, c := range testCases {
		s.T().Run(c.description, func(t *testing.T) {
			c.p.SetName("BLAHBLAH")

			err := s.validator.validateCompilableForLifecycle(c.p)
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func (s *PolicyValidatorTestSuite) TestValidateLifeCycleEnforcementCombination() {
	testCases := []struct {
		description  string
		p            *storage.Policy
		expectedSize int
	}{
		{
			description: "Remove invalid enforcement with runtime lifecycle",
			p: storage.Policy_builder{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						SectionName: "section-1",
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ImageTag,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "latest",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "Asfasf",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "asfasfaa",
									}.Build(),
								},
							}.Build(),
						},
					}.Build(),
				},
				PolicyVersion: "1.1",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT,
				},
			}.Build(),
			expectedSize: 2,
		},
		{
			description: "Remove invalid enforcement with build lifecycle",
			p: storage.Policy_builder{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						SectionName: "section-1",
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ImageTag,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "latest",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "Asfasf",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "asfasfaa",
									}.Build(),
								},
							}.Build(),
						},
					}.Build(),
				},
				PolicyVersion: "1.1",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
				},
			}.Build(),
			expectedSize: 1,
		},
		{
			description: "Remove invalid enforcement with deployment lifecycle",
			p: storage.Policy_builder{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_DEPLOY,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						SectionName: "section-1",
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ImageTag,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "latest",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "Asfasf",
									}.Build(),
								},
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									storage.PolicyValue_builder{
										Value: "asfasfaa",
									}.Build(),
								},
							}.Build(),
						},
					}.Build(),
				},
				PolicyVersion: "1.1",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT,
				},
			}.Build(),
			expectedSize: 2,
		},
	}

	for _, c := range testCases {
		s.T().Run(c.description, func(t *testing.T) {
			c.p.SetName("BLAHBLAH")
			s.validator.removeEnforcementsForMissingLifecycles(c.p)
			assert.Equal(t, c.expectedSize, len(c.p.GetEnforcementActions()), "enforcement size does not match")
		})
	}
}

func (s *PolicyValidatorTestSuite) TestValidateSeverity() {
	policy := &storage.Policy{}
	policy.SetSeverity(storage.Severity_LOW_SEVERITY)
	err := s.validator.validateSeverity(policy)
	s.NoError(err, "severity should pass when set")

	policy = &storage.Policy{}
	policy.SetSeverity(storage.Severity_UNSET_SEVERITY)
	err = s.validator.validateSeverity(policy)
	s.Error(err, "severity should fail when not set")
}

func (s *PolicyValidatorTestSuite) TestValidateCategories() {
	policy := &storage.Policy{}
	err := s.validator.validateCategories(policy)
	s.Error(err, "at least one category should be required")

	policy = &storage.Policy{}
	policy.SetCategories([]string{
		"cat1",
		"cat2",
		"cat1",
	})
	err = s.validator.validateCategories(policy)
	s.Error(err, "duplicate categories should fail")

	policy = &storage.Policy{}
	policy.SetCategories([]string{
		"cat1",
		"cat2",
	})
	err = s.validator.validateCategories(policy)
	s.NoError(err, "valid categories should not fail")
}

func (s *PolicyValidatorTestSuite) TestValidateNotifiers() {
	policy := &storage.Policy{}
	policy.SetNotifiers([]string{
		"id1",
	})
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id1").Return((*storage.Notifier)(nil), true, nil)
	err := s.validator.validateNotifiers(s.requestContext, policy)
	s.NoError(err, "severity should pass when set")

	policy = &storage.Policy{}
	policy.SetNotifiers([]string{
		"id2",
	})
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id2").Return((*storage.Notifier)(nil), false, nil)
	err = s.validator.validateNotifiers(s.requestContext, policy)
	s.Error(err, "should fail when it does not exist")

	policy = &storage.Policy{}
	policy.SetNotifiers([]string{
		"id3",
	})
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id3").Return((*storage.Notifier)(nil), true, errors.New("oh noes"))
	err = s.validator.validateNotifiers(s.requestContext, policy)
	s.Error(err, "should fail when an error is thrown")
}

func (s *PolicyValidatorTestSuite) TestValidateExclusions() {
	policy := &storage.Policy{}
	err := s.validator.validateExclusions(policy)
	s.NoError(err, "excluded scopes should not be required")

	deployment := &storage.Exclusion_Deployment{}
	deployment.SetName("that phat cluster")
	deploymentExclusion := &storage.Exclusion{}
	deploymentExclusion.SetDeployment(deployment)
	policy = &storage.Policy{}
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	policy.SetExclusions([]*storage.Exclusion{
		deploymentExclusion,
	})
	err = s.validator.validateExclusions(policy)
	s.NoError(err, "valid to excluded scope by deployment name")

	ei := &storage.Exclusion_Image{}
	ei.SetName("stackrox.io")
	imageExclusion := &storage.Exclusion{}
	imageExclusion.SetImage(ei)
	policy = &storage.Policy{}
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_BUILD,
	})
	policy.SetExclusions([]*storage.Exclusion{
		imageExclusion,
	})
	err = s.validator.validateExclusions(policy)
	s.NoError(err, "valid to excluded scope by image registry")

	policy = &storage.Policy{}
	policy.SetExclusions([]*storage.Exclusion{
		imageExclusion,
	})
	err = s.validator.validateExclusions(policy)
	s.Error(err, "not valid to excluded scope by image registry since build time lifecycle isn't present")

	emptyExclusion := &storage.Exclusion{}
	policy = &storage.Policy{}
	policy.SetExclusions([]*storage.Exclusion{
		emptyExclusion,
	})
	err = s.validator.validateExclusions(policy)
	s.Error(err, "excluded scope requires either container or deployment configuration")

	emptyLabelExclusion := storage.Exclusion_builder{
		Deployment: storage.Exclusion_Deployment_builder{
			Scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key: "",
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	policy = &storage.Policy{}
	policy.SetExclusions([]*storage.Exclusion{
		emptyLabelExclusion,
	})
	err = s.validator.validateExclusions(policy)
	s.Error(err, "label regex in excluded scope, if not nil, must be non-empty")

	anyKeyLabelExclusion := storage.Exclusion_builder{
		Deployment: storage.Exclusion_Deployment_builder{
			Scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   ".*",
					Value: "",
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	policy = &storage.Policy{}
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	policy.SetExclusions([]*storage.Exclusion{
		anyKeyLabelExclusion,
	})
	s.NoError(s.validator.validateExclusions(policy))

	anyLabelExclusion := storage.Exclusion_builder{
		Deployment: storage.Exclusion_Deployment_builder{
			Scope: storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   ".*",
					Value: ".*",
				}.Build(),
			}.Build(),
		}.Build(),
	}.Build()
	policy = &storage.Policy{}
	policy.SetLifecycleStages([]storage.LifecycleStage{
		storage.LifecycleStage_DEPLOY,
	})
	policy.SetExclusions([]*storage.Exclusion{
		anyLabelExclusion,
	})
	s.NoError(s.validator.validateExclusions(policy))
}

func (s *PolicyValidatorTestSuite) TestAllDefaultPoliciesValidate() {
	defaultPolicies, err := policies.DefaultPolicies()
	s.Require().NoError(err)

	for _, policy := range defaultPolicies {
		err = s.validator.validate(context.Background(), policy)
		s.NoError(err, fmt.Sprintf("Policy %q failed validation with error: %v", policy.GetName(), err))
	}
}

func (s *PolicyValidatorTestSuite) TestNoScopeLabelsForAuditEventSource() {
	validPolicy := storage.Policy_builder{
		Name:            "runtime-policy-valid",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "SECRETS",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "GET",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		Scope: []*storage.Scope{
			storage.Scope_builder{
				Cluster: "cluster-remote",
			}.Build(),
			storage.Scope_builder{
				Namespace: "cluster-namespace",
			}.Build(),
		},
	}.Build()
	assert.NoError(s.T(), s.validator.validateEventSource(validPolicy))

	invalidScopePolicy := storage.Policy_builder{
		Name:            "runtime-policy-invalid-scope",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "SECRETS",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "GET",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		Scope: []*storage.Scope{
			storage.Scope_builder{
				Label: storage.Scope_Label_builder{
					Key:   "label",
					Value: "label-value",
				}.Build(),
			}.Build(),
		},
	}.Build()
	assert.Error(s.T(), s.validator.validateEventSource(invalidScopePolicy))
}

func (s *PolicyValidatorTestSuite) TestValidateAuditEventSource() {
	assert.Error(s.T(), s.validator.validateEventSource(storage.Policy_builder{
		Name:            "deploy-policy",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "SECRETS",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "GET",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}.Build()))

	assert.Error(s.T(), s.validator.validateEventSource(storage.Policy_builder{
		Name:            "build-policy",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "SECRETS",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "GET",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}.Build()))
}

func (s *PolicyValidatorTestSuite) TestValidateNoDockerfileLineFrom() {
	validator := newPolicyValidator(s.nStorage)

	goodPolicy := booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE, map[string]string{
		fieldnames.DockerfileLine: "COPY=",
	})
	goodPolicy.SetName("GOOD")
	badPolicy := booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE, map[string]string{
		fieldnames.DockerfileLine: "FROM=",
	})
	badPolicy.SetName("BAD")

	for _, testCase := range []struct {
		p             *storage.Policy
		includeOption bool
		errExpected   bool
	}{
		{
			p:             goodPolicy,
			includeOption: false,
			errExpected:   false,
		},
		{
			p:             badPolicy,
			includeOption: false,
			errExpected:   false,
		},
		{
			p:             goodPolicy,
			includeOption: true,
			errExpected:   false,
		},
		{
			p:             badPolicy,
			includeOption: true,
			errExpected:   true,
		},
	} {
		s.Run(fmt.Sprintf("%s_%v", testCase.p.GetName(), testCase.includeOption), func() {
			var options []booleanpolicy.ValidateOption
			if testCase.includeOption {
				options = append(options, booleanpolicy.ValidateNoFromInDockerfileLine())
			}
			err := validator.validateCompilableForLifecycle(testCase.p, options...)
			s.Equal(testCase.errExpected, err != nil, "Result didn't match expectations (got error: %v)", err)
		})
	}

}

func (s *PolicyValidatorTestSuite) TestValidateEnforcement() {
	validatorWithFlag := newPolicyValidator(s.nStorage)

	cases := map[string]struct {
		policy        *storage.Policy
		expectedError string
		featureFlag   bool
	}{
		"Missing Egress Policy Field": {
			policy: storage.Policy_builder{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: augmentedobjs.HasEgressPolicyCustomTag,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			featureFlag:   true,
			expectedError: fmt.Sprintf("enforcement of %s is not allowed", augmentedobjs.HasEgressPolicyCustomTag),
		},
		"Missing Ingress Policy Field": {
			policy: storage.Policy_builder{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: augmentedobjs.HasIngressPolicyCustomTag,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			featureFlag:   true,
			expectedError: fmt.Sprintf("enforcement of %s is not allowed", augmentedobjs.HasIngressPolicyCustomTag),
		},
		"Enforceable Policy Field": {
			policy: storage.Policy_builder{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: augmentedobjs.ContainerNameCustomTag,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			featureFlag:   true,
			expectedError: "",
		},
	}
	for name, c := range cases {
		s.T().Run(name, func(t *testing.T) {
			err := validatorWithFlag.validateEnforcement(c.policy)
			if c.expectedError != "" {
				assert.Equal(t, c.expectedError, err.Error())
			} else {
				assert.Truef(t, err == nil, "Error not expected")
			}
		})
	}
}
