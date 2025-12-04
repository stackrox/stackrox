package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
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
					access:      s.getDeploymentNodeFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
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
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/tmp/foo", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment file policy with no operations",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment file policy with all allowed files",
			policy:      s.getDeploymentFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentNodeFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
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
					s.Require().NotNil(violations.FileAccessViolation, "expected file access violation in alert")

					fileAccessViolation := violations.FileAccessViolation
					s.Require().Len(fileAccessViolation.GetAccesses(), 1, "expected one file access in alert")

					protoassert.Equal(s.T(), event.access, fileAccessViolation.GetAccesses()[0])
				} else {
					s.Require().Nil(violations.FileAccessViolation, "expected no alerts")
				}
			}
		})
	}
}

// getFileAccessPolicy is a generic helper for creating file access policies.
func (s *DeploymentDetectionTestSuite) getFileAccessPolicy(isNodePath bool, operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	var pathValues []*storage.PolicyValue
	for _, path := range paths {
		pathValues = append(pathValues, &storage.PolicyValue{
			Value: path,
		})
	}

	fieldName := fieldnames.NodeFilePath
	if !isNodePath {
		fieldName = fieldnames.MountedFilePath
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

func (s *DeploymentDetectionTestSuite) getMountedFileAccessPolicyWithOperations(operations []storage.FileAccess_Operation, negate bool, paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(false, operations, negate, paths...)
}

func (s *DeploymentDetectionTestSuite) getMountedFileAccessPolicy(paths ...string) *storage.Policy {
	return s.getFileAccessPolicy(false, nil, false, paths...)
}

func (s *DeploymentDetectionTestSuite) TestDeploymentMountedFileAccess() {
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
			description: "Deployment mounted file open policy with matching event",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment mounted file open policy with mismatching event (UNLINK)",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment mounted file open policy with mismatching event (/etc/sudoers)",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment mounted file policy with negated file operation",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment mounted file policy with multiple operations",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment mounted file policy with multiple negated operations",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment mounted file policy with multiple files and single operation",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment mounted file policy with multiple files and multiple operations",
			policy: s.getMountedFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/sudoers", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Deployment mounted file policy with no operations",
			policy:      s.getMountedFileAccessPolicy("/etc/passwd"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Deployment mounted file policy with all allowed files",
			policy:      s.getMountedFileAccessPolicy("/etc/passwd", "/etc/ssh/sshd_config", "/etc/shadow", "/etc/sudoers"),
			events: []eventWrapper{
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/ssh/sshd_config", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      s.getDeploymentMountedFileAccessEvent("/etc/sudoers", storage.FileAccess_OPEN),
					expectAlert: true,
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
					s.Require().NotNil(violations.FileAccessViolation, "expected file access violation in alert")

					fileAccessViolation := violations.FileAccessViolation
					s.Require().Len(fileAccessViolation.GetAccesses(), 1, "expected one file access in alert")

					protoassert.Equal(s.T(), event.access, fileAccessViolation.GetAccesses()[0])
				} else {
					s.Require().Nil(violations.FileAccessViolation, "expected no alerts")
				}
			}
		})
	}
}

// getFileAccessEvent is a generic helper for creating file access events.
func (s *DeploymentDetectionTestSuite) getFileAccessEvent(path string, operation storage.FileAccess_Operation, isNodePath bool) *storage.FileAccess {
	file := &storage.FileAccess_File{}
	if isNodePath {
		file.NodePath = path
	} else {
		file.MountedPath = path
	}
	return &storage.FileAccess{
		File:      file,
		Operation: operation,
	}
}

func (s *DeploymentDetectionTestSuite) getDeploymentNodeFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return s.getFileAccessEvent(path, operation, true)
}

func (s *DeploymentDetectionTestSuite) getDeploymentMountedFileAccessEvent(path string, operation storage.FileAccess_Operation) *storage.FileAccess {
	return s.getFileAccessEvent(path, operation, false)
}
