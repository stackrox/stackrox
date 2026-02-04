package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type DeploymentDetectionTestSuite struct {
	suite.Suite
}

func TestDeploymentDetection(t *testing.T) {
	suite.Run(t, new(DeploymentDetectionTestSuite))
}

func (s *DeploymentDetectionTestSuite) TestDeploymentFileAccess() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Deployment file open policy with matching event",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file open policy with mismatching event (UNLINK)",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file open policy with mismatching event (/tmp/foo)",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with negated file operation",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple operations",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with multiple negated operations",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple files and single operation",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with multiple files and multiple operations",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/tmp/foo", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with no operations",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with all allowed files",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with suffix",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/passwd-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/shadow-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/ssh/sshd_config-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentActualFileAccessEvent("/etc/sudoers-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(s.T())
		defer ResetFieldMetadataSingleton(s.T())

		s.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			s.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDeployment := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDeployment, event.access)
				s.Require().NoError(err)

				if event.expectAlert {
					s.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					s.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					s.Require().NotNil(fileAccess, "expected file access info")

					// Verify the file access details match
					s.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					s.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					s.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					s.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

func (s *DeploymentDetectionTestSuite) TestDeploymentEffectiveFileAccess() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		{
			description: "Deployment effective file open policy with matching event",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file open policy with mismatching event (UNLINK)",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file open policy with mismatching event (/etc/sudoers)",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with negated file operation",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple operations",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple negated operations",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple files and single operation",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with multiple files and multiple operations",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment effective file policy with no operations",
			policy:      s.getEffectiveFileAccessPolicy("/etc/passwd"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment effective file policy with all allowed files",
			policy:      s.getEffectiveFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with suffix",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/passwd-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/shadow-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/ssh/sshd_config-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentEffectiveFileAccessEvent("/etc/sudoers-suffix", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(s.T())
		defer ResetFieldMetadataSingleton(s.T())

		s.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			s.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDeployment := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDeployment, event.access)
				s.Require().NoError(err)

				if event.expectAlert {
					s.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					s.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					s.Require().NotNil(fileAccess, "expected file access info")

					// Verify the file access details match
					s.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					s.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					s.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					s.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

func (s *DeploymentDetectionTestSuite) TestDeploymentDualPathMatching() {
	deployment := &storage.Deployment{
		Name: "test-deployment",
		Id:   "test-deployment-id",
	}

	type eventWrapper struct {
		access      *storage.FileAccess
		expectAlert bool
	}

	for _, tc := range []struct {
		description string
		policy      *storage.Policy
		events      []eventWrapper
	}{
		// Valid test cases - expected behavior
		{
			description: "Event with both paths - policy matches actual path only",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy matches effective only",
			policy: s.getEffectiveFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH paths (AND within section)",
			policy:      s.getDualPathPolicy("/etc/passwd", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section policy - first section matches (OR behavior)",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section policy - second section matches (OR behavior)",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.FileOperation,
							Values:    []*storage.PolicyValue{{Value: "OPEN"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with mixed path types - actual path section matches",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with mixed path types - effective path section matches",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Multi-section with dual paths in one section - complex AND/OR",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/ssh/sshd_config"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					// Matches section 1 (both actual and effective paths match)
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},

		// Invalid/edge cases - unexpected behaviors
		{
			description: "Event with both paths - policy matches neither",
			policy: s.getDeploymentFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/sudoers",
			),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH but only actual path matches",
			policy:      s.getDualPathPolicy("/etc/passwd", "/etc/sudoers", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH but only effective matches",
			policy:      s.getDualPathPolicy("/etc/sudoers", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_OPEN}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Event with both paths - policy requires BOTH but operation doesn't match",
			policy:      s.getDualPathPolicy("/etc/passwd", "/etc/shadow", []storage.FileAccess_Operation{storage.FileAccess_CREATE}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Multi-section policy - no sections match",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/ssh/sshd_config"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Multi-section with dual paths - neither section matches completely",
			policy: s.getMultiSectionPolicy([]*storage.PolicySection{
				{
					SectionName: "section 1",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/passwd"}},
						},
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/sudoers"}},
						},
					},
				},
				{
					SectionName: "section 2",
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fieldnames.ActualPath,
							Values:    []*storage.PolicyValue{{Value: "/etc/ssh/sshd_config"}},
						},
						{
							FieldName: fieldnames.EffectivePath,
							Values:    []*storage.PolicyValue{{Value: "/etc/shadow"}},
						},
					},
				},
			}),
			events: []eventWrapper{
				{
					// Section 1: actual matches, effective doesn't (AND fails)
					// Section 2: actual doesn't match, effective does (AND fails)
					// Overall: no section fully matches (OR fails)
					access:      s.getDualPathFileAccessEvent("/etc/passwd", "/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
	} {
		testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, true)
		defer testutils.MustUpdateFeature(s.T(), features.SensitiveFileActivity, false)
		ResetFieldMetadataSingleton(s.T())
		defer ResetFieldMetadataSingleton(s.T())

		s.Run(tc.description, func() {
			matcher, err := BuildDeploymentWithFileAccessMatcher(tc.policy)
			s.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				enhancedDeployment := EnhancedDeployment{
					Deployment:             deployment,
					Images:                 nil,
					NetworkPoliciesApplied: nil,
				}
				violations, err := matcher.MatchDeploymentWithFileAccess(&cache, enhancedDeployment, event.access)
				s.Require().NoError(err)

				if event.expectAlert {
					s.Require().Len(violations.AlertViolations, 1, "expected one file access violation in alert")
					s.Require().Equal(storage.Alert_Violation_FILE_ACCESS, violations.AlertViolations[0].GetType(), "expected FILE_ACCESS type")

					fileAccess := violations.AlertViolations[0].GetFileAccess()
					s.Require().NotNil(fileAccess, "expected file access info")

					// Verify the file access details match
					s.Require().Equal(event.access.GetFile().GetEffectivePath(), fileAccess.GetFile().GetEffectivePath())
					s.Require().Equal(event.access.GetFile().GetActualPath(), fileAccess.GetFile().GetActualPath())
					s.Require().Equal(event.access.GetOperation(), fileAccess.GetOperation())
				} else {
					s.Require().Empty(violations.AlertViolations, "expected no alerts")
				}
			}
		})
	}
}

// getFileAccessEvent is a generic helper for creating file access events.
func (s *DeploymentDetectionTestSuite) getFileAccessEvent(path string, operation storage.FileAccess_Operation, isActualPath bool) *storage.FileAccess {
	file := &storage.FileAccess_File{}
	if isActualPath {
		file.ActualPath = path
	} else {
		file.EffectivePath = path
	}
	return &storage.FileAccess{
		File:      file,
		Operation: operation,
	}
}

func (s *DeploymentDetectionTestSuite) getDeploymentActualFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return s.getFileAccessEvent(path, operation, true)
}

func (s *DeploymentDetectionTestSuite) getDeploymentEffectiveFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return s.getFileAccessEvent(path, operation, false)
}

// getFileAccessPolicy is a generic helper for creating file access policies.
func (s *DeploymentDetectionTestSuite) getFileAccessPolicy(isActualPath bool, operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	var pathValues []*storage.PolicyValue
	for _, path := range paths {
		pathValues = append(pathValues, &storage.PolicyValue{
			Value: path,
		})
	}

	fieldName := fieldnames.ActualPath
	if !isActualPath {
		fieldName = fieldnames.EffectivePath
	}

	policyGroups := []*storage.PolicyGroup{
		{
			FieldName: fieldName,
			Values:    pathValues,
		},
	}

	var operationValues []*storage.PolicyValue
	for _, op := range operations {
		operationValues = append(operationValues, &storage.PolicyValue{
			Value: op.String(),
		})
	}

	if len(operationValues) != 0 {
		policyGroups = append(policyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Values:    operationValues,
			Negate:    negate,
		})
	}

	return &storage.Policy{
		Id:            uuid.NewV4().String(),
		PolicyVersion: "1.1",
		Name:          "Sensitive File Access in Deployment",
		Severity:      storage.Severity_HIGH_SEVERITY,
		Categories:    []string{"File System"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName:  "section 1",
				PolicyGroups: policyGroups,
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
	}
}

func (s *DeploymentDetectionTestSuite) getDeploymentFileAccessPolicyWithOperations(operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(true, operations, negate, paths...)
}

func (s *DeploymentDetectionTestSuite) getDeploymentFileAccessPolicy(paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(true, nil, false, paths...)
}

func (s *DeploymentDetectionTestSuite) getEffectiveFileAccessPolicyWithOperations(operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(false, operations, negate, paths...)
}

func (s *DeploymentDetectionTestSuite) getEffectiveFileAccessPolicy(paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(false, nil, false, paths...)
}

// Helper to create file access events with BOTH actual path and effective path populated
func (s *DeploymentDetectionTestSuite) getDualPathFileAccessEvent(actualPath, effectivePath string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return &storage.FileAccess{
		File: &storage.FileAccess_File{
			ActualPath:    actualPath,
			EffectivePath: effectivePath,
		},
		Operation: operation,
	}
}

// Helper to create a policy with both ActualPath AND EffectivePath in the same section (AND behavior)
func (s *DeploymentDetectionTestSuite) getDualPathPolicy(actualPath, effectivePath string, operations []storage.FileAccess_Operation) *storage.Policy {
	policyGroups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.ActualPath,
			Values:    []*storage.PolicyValue{{Value: actualPath}},
		},
		{
			FieldName: fieldnames.EffectivePath,
			Values:    []*storage.PolicyValue{{Value: effectivePath}},
		},
	}

	if len(operations) > 0 {
		var operationValues []*storage.PolicyValue
		for _, op := range operations {
			operationValues = append(operationValues, &storage.PolicyValue{
				Value: op.String(),
			})
		}
		policyGroups = append(policyGroups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Values:    operationValues,
		})
	}

	return &storage.Policy{
		Id:            uuid.NewV4().String(),
		PolicyVersion: "1.1",
		Name:          "Dual Path Policy",
		Severity:      storage.Severity_HIGH_SEVERITY,
		Categories:    []string{"File System"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName:  "section 1",
				PolicyGroups: policyGroups,
			},
		},
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
	}
}

// Helper to create a multi-section policy (OR behavior across sections)
func (s *DeploymentDetectionTestSuite) getMultiSectionPolicy(sections []*storage.PolicySection) *storage.Policy {
	return &storage.Policy{
		Id:              uuid.NewV4().String(),
		PolicyVersion:   "1.1",
		Name:            "Multi-Section Policy",
		Severity:        storage.Severity_HIGH_SEVERITY,
		Categories:      []string{"File System"},
		PolicySections:  sections,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     storage.EventSource_DEPLOYMENT_EVENT,
	}
}
