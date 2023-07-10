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
	policy := &storage.Policy{
		Name: "Robert",
	}
	err := s.validator.validateName(policy)
	s.NoError(err, "\"Robert\" should be a valid name")

	policy = &storage.Policy{
		Name: "Jim-Bob",
	}
	err = s.validator.validateName(policy)
	s.NoError(err, "\"Jim-Bob\" should be a valid name")

	policy = &storage.Policy{
		Name: "Jimmy_John",
	}
	err = s.validator.validateName(policy)
	s.NoError(err, "\"Jimmy_John\" should be a valid name")

	policy = &storage.Policy{
		Name: "",
	}
	err = s.validator.validateName(policy)
	s.Error(err, "a name should be required")

	policy = &storage.Policy{
		Name: "Rob",
	}
	err = s.validator.validateName(policy)
	s.Error(err, "names that are too short should not be supported")

	policy = &storage.Policy{
		Name: "RobertIsTheCoolestDudeEverToLiveUnlessYouCountMrTBecauseHeIsEvenDoperHisVanIsSweetAndHisHairIsCoolAndIReallyLikeAllTheGoldChainsHeWears",
	}
	err = s.validator.validateName(policy)
	s.Error(err, "names that are more than 128 chars are not supported")

	policy = &storage.Policy{
		Name: "Rob$",
	}
	err = s.validator.validateName(policy)
	s.Error(err, "special characters should not be supported")

	policy = &storage.Policy{
		Name: "  Boo's policy  ",
	}
	err = s.validator.validateName(policy)
	s.NoError(err, "leading and trailing spaces should be trimmed")
	s.Equal("Boo's policy", policy.Name)
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
			policy := &storage.Policy{
				PolicyVersion: c.version,
			}
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
				{
					Value: "hi",
				},
			},
			expectedError: false,
		},
		{
			name: "drops only",
			drops: []*storage.PolicyValue{
				{
					Value: "hi",
				},
			},
			expectedError: false,
		},
		{
			name: "different adds and drops",
			adds: []*storage.PolicyValue{
				{
					Value: "hey",
				},
			},
			drops: []*storage.PolicyValue{
				{
					Value: "hello",
				},
			},
			expectedError: false,
		},
		{
			name: "same adds and drops",
			adds: []*storage.PolicyValue{
				{
					Value: "hello",
				},
			},
			drops: []*storage.PolicyValue{
				{
					Value: "hello",
				},
			},
			expectedError: true,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			policy := &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "section-1",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.AddCaps,
								Values:    c.adds,
							},
							{
								FieldName: fieldnames.DropCaps,
								Values:    c.drops,
							},
						},
					},
				},
				PolicyVersion: "1.1",
			}
			assert.Equal(t, c.expectedError, s.validator.validateCapabilities(policy) != nil)
		})
	}
}

func (s *PolicyValidatorTestSuite) TestValidateDescription() {
	policy := &storage.Policy{
		Description: "",
	}
	err := s.validator.validateDescription(policy)
	s.NoError(err, "descriptions are not required")

	policy = &storage.Policy{
		Description: "Yo",
	}
	err = s.validator.validateDescription(policy)
	s.NoError(err, "descriptions can be as short as they like")

	policy = &storage.Policy{
		Description: "This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!",
	}
	err = s.validator.validateDescription(policy)
	s.NoError(err, "descriptions should take the form of a sentence")

	policy = &storage.Policy{
		Description: `This policy is the stop when an image is terrible and will cause us to lose lots-o-dough. Why? Cause Money!
			Oh, and I almost forgot that this is also to help the good people of nowhere-ville get back on their
			feet after that tornado ripped their town to shreds and left them nothing but pineapple and gum.  It was the It was
			the best of times, it was the worst of times, it was the age of wisdom, it was the age of foolishness, it was the
			epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of Darkness, it was
			the spring of hope, it was the winter of despair, we had everything before us, we had nothing before us, we were all
			going direct to Heaven, we were all going direct the other way--in short, the period was so far like the present
			period that some of its noisiest authorities insisted on its being received, for good or for evil, in the superlative
			degree of comparison only.`,
	}
	err = s.validator.validateDescription(policy)
	s.Error(err, "descriptions should be no more than 800 chars")

	policy = &storage.Policy{
		Description: "This$Rox",
	}
	err = s.validator.validateDescription(policy)
	s.Error(err, "no special characters")
}

func booleanPolicyWithFields(lifecycleStage storage.LifecycleStage, eventSource storage.EventSource, fieldsToVals map[string]string) *storage.Policy {
	groups := make([]*storage.PolicyGroup, 0, len(fieldsToVals))
	for k, v := range fieldsToVals {
		groups = append(groups, &storage.PolicyGroup{FieldName: k, Values: []*storage.PolicyValue{{Value: v}}})
	}
	return &storage.Policy{
		PolicyVersion:   policyversion.CurrentVersion().String(),
		LifecycleStages: []storage.LifecycleStage{lifecycleStage},
		EventSource:     eventSource,
		PolicySections:  []*storage.PolicySection{{PolicyGroups: groups}},
	}
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
			c.p.Name = "BLAHBLAH"

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
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_RUNTIME,
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
							{
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									{
										Value: "Asfasf",
									},
								},
							},
							{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									{
										Value: "asfasfaa",
									},
								},
							},
						},
					},
				},
				PolicyVersion: "1.1",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT,
				},
			},
			expectedSize: 2,
		},
		{
			description: "Remove invalid enforcement with build lifecycle",
			p: &storage.Policy{
				LifecycleStages: []storage.LifecycleStage{
					storage.LifecycleStage_BUILD,
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
							{
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									{
										Value: "Asfasf",
									},
								},
							},
							{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									{
										Value: "asfasfaa",
									},
								},
							},
						},
					},
				},
				PolicyVersion: "1.1",
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
								FieldName: fieldnames.VolumeName,
								Values: []*storage.PolicyValue{
									{
										Value: "Asfasf",
									},
								},
							},
							{
								FieldName: fieldnames.ProcessName,
								Values: []*storage.PolicyValue{
									{
										Value: "asfasfaa",
									},
								},
							},
						},
					},
				},
				PolicyVersion: "1.1",
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
					storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT,
					storage.EnforcementAction_KILL_POD_ENFORCEMENT,
					storage.EnforcementAction_FAIL_KUBE_REQUEST_ENFORCEMENT,
				},
			},
			expectedSize: 2,
		},
	}

	for _, c := range testCases {
		s.T().Run(c.description, func(t *testing.T) {
			c.p.Name = "BLAHBLAH"
			s.validator.removeEnforcementsForMissingLifecycles(c.p)
			assert.Equal(t, c.expectedSize, len(c.p.EnforcementActions), "enforcement size does not match")
		})
	}
}

func (s *PolicyValidatorTestSuite) TestValidateSeverity() {
	policy := &storage.Policy{
		Severity: storage.Severity_LOW_SEVERITY,
	}
	err := s.validator.validateSeverity(policy)
	s.NoError(err, "severity should pass when set")

	policy = &storage.Policy{
		Severity: storage.Severity_UNSET_SEVERITY,
	}
	err = s.validator.validateSeverity(policy)
	s.Error(err, "severity should fail when not set")
}

func (s *PolicyValidatorTestSuite) TestValidateCategories() {
	policy := &storage.Policy{}
	err := s.validator.validateCategories(policy)
	s.Error(err, "at least one category should be required")

	policy = &storage.Policy{
		Categories: []string{
			"cat1",
			"cat2",
			"cat1",
		},
	}
	err = s.validator.validateCategories(policy)
	s.Error(err, "duplicate categories should fail")

	policy = &storage.Policy{
		Categories: []string{
			"cat1",
			"cat2",
		},
	}
	err = s.validator.validateCategories(policy)
	s.NoError(err, "valid categories should not fail")
}

func (s *PolicyValidatorTestSuite) TestValidateNotifiers() {
	policy := &storage.Policy{
		Notifiers: []string{
			"id1",
		},
	}
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id1").Return((*storage.Notifier)(nil), true, nil)
	err := s.validator.validateNotifiers(s.requestContext, policy)
	s.NoError(err, "severity should pass when set")

	policy = &storage.Policy{
		Notifiers: []string{
			"id2",
		},
	}
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id2").Return((*storage.Notifier)(nil), false, nil)
	err = s.validator.validateNotifiers(s.requestContext, policy)
	s.Error(err, "should fail when it does not exist")

	policy = &storage.Policy{
		Notifiers: []string{
			"id3",
		},
	}
	s.nStorage.EXPECT().GetNotifier(s.requestContext, "id3").Return((*storage.Notifier)(nil), true, errors.New("oh noes"))
	err = s.validator.validateNotifiers(s.requestContext, policy)
	s.Error(err, "should fail when an error is thrown")
}

func (s *PolicyValidatorTestSuite) TestValidateExclusions() {
	policy := &storage.Policy{}
	err := s.validator.validateExclusions(policy)
	s.NoError(err, "excluded scopes should not be required")

	deployment := &storage.Exclusion_Deployment{
		Name: "that phat cluster",
	}
	deploymentExclusion := &storage.Exclusion{
		Deployment: deployment,
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
		},
		Exclusions: []*storage.Exclusion{
			deploymentExclusion,
		},
	}
	err = s.validator.validateExclusions(policy)
	s.NoError(err, "valid to excluded scope by deployment name")

	imageExclusion := &storage.Exclusion{
		Image: &storage.Exclusion_Image{
			Name: "stackrox.io",
		},
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_BUILD,
		},
		Exclusions: []*storage.Exclusion{
			imageExclusion,
		},
	}
	err = s.validator.validateExclusions(policy)
	s.NoError(err, "valid to excluded scope by image registry")

	policy = &storage.Policy{
		Exclusions: []*storage.Exclusion{
			imageExclusion,
		},
	}
	err = s.validator.validateExclusions(policy)
	s.Error(err, "not valid to excluded scope by image registry since build time lifecycle isn't present")

	emptyExclusion := &storage.Exclusion{}
	policy = &storage.Policy{
		Exclusions: []*storage.Exclusion{
			emptyExclusion,
		},
	}
	err = s.validator.validateExclusions(policy)
	s.Error(err, "excluded scope requires either container or deployment configuration")

	emptyLabelExclusion := &storage.Exclusion{
		Deployment: &storage.Exclusion_Deployment{
			Scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key: "",
				},
			},
		},
	}
	policy = &storage.Policy{
		Exclusions: []*storage.Exclusion{
			emptyLabelExclusion,
		},
	}
	err = s.validator.validateExclusions(policy)
	s.Error(err, "label regex in excluded scope, if not nil, must be non-empty")

	anyKeyLabelExclusion := &storage.Exclusion{
		Deployment: &storage.Exclusion_Deployment{
			Scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   ".*",
					Value: "",
				},
			},
		},
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
		},
		Exclusions: []*storage.Exclusion{
			anyKeyLabelExclusion,
		},
	}
	s.NoError(s.validator.validateExclusions(policy))

	anyLabelExclusion := &storage.Exclusion{
		Deployment: &storage.Exclusion_Deployment{
			Scope: &storage.Scope{
				Label: &storage.Scope_Label{
					Key:   ".*",
					Value: ".*",
				},
			},
		},
	}
	policy = &storage.Policy{
		LifecycleStages: []storage.LifecycleStage{
			storage.LifecycleStage_DEPLOY,
		},
		Exclusions: []*storage.Exclusion{
			anyLabelExclusion,
		},
	}
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
	validPolicy := &storage.Policy{
		Name:            "runtime-policy-valid",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
		Scope: []*storage.Scope{
			{
				Cluster: "cluster-remote",
			},
			{
				Namespace: "cluster-namespace",
			},
		},
	}
	assert.NoError(s.T(), s.validator.validateEventSource(validPolicy))

	invalidScopePolicy := &storage.Policy{
		Name:            "runtime-policy-invalid-scope",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
		Scope: []*storage.Scope{
			{
				Label: &storage.Scope_Label{
					Key:   "label",
					Value: "label-value",
				},
			},
		},
	}
	assert.Error(s.T(), s.validator.validateEventSource(invalidScopePolicy))
}

func (s *PolicyValidatorTestSuite) TestValidateAuditEventSource() {
	assert.Error(s.T(), s.validator.validateEventSource(&storage.Policy{
		Name:            "deploy-policy",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
		EventSource:     storage.EventSource_AUDIT_LOG_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}))

	assert.Error(s.T(), s.validator.validateEventSource(&storage.Policy{
		Name:            "build-policy",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
		PolicyVersion:   policyversion.CurrentVersion().String(),
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.KubeResource,
						Values: []*storage.PolicyValue{
							{
								Value: "SECRETS",
							},
						},
					},
					{
						FieldName: fieldnames.KubeAPIVerb,
						Values: []*storage.PolicyValue{
							{
								Value: "GET",
							},
						},
					},
				},
			},
		},
	}))
}

func (s *PolicyValidatorTestSuite) TestValidateNoDockerfileLineFrom() {
	validator := newPolicyValidator(s.nStorage)

	goodPolicy := booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE, map[string]string{
		fieldnames.DockerfileLine: "COPY=",
	})
	goodPolicy.Name = "GOOD"
	badPolicy := booleanPolicyWithFields(storage.LifecycleStage_DEPLOY, storage.EventSource_NOT_APPLICABLE, map[string]string{
		fieldnames.DockerfileLine: "FROM=",
	})
	badPolicy.Name = "BAD"

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
			policy: &storage.Policy{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: augmentedobjs.HasEgressPolicyCustomTag,
							},
						},
					},
				},
			},
			featureFlag:   true,
			expectedError: fmt.Sprintf("enforcement of %s is not allowed", augmentedobjs.HasEgressPolicyCustomTag),
		},
		"Missing Ingress Policy Field": {
			policy: &storage.Policy{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: augmentedobjs.HasIngressPolicyCustomTag,
							},
						},
					},
				},
			},
			featureFlag:   true,
			expectedError: fmt.Sprintf("enforcement of %s is not allowed", augmentedobjs.HasIngressPolicyCustomTag),
		},
		"Enforceable Policy Field": {
			policy: &storage.Policy{
				EnforcementActions: []storage.EnforcementAction{
					storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: augmentedobjs.ContainerNameCustomTag,
							},
						},
					},
				},
			},
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
