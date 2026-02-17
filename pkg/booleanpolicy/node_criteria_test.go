package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type NodeCriteriaTestSuite struct {
	suite.Suite
}

func TestNodeCriteria(t *testing.T) {
	suite.Run(t, new(NodeCriteriaTestSuite))
}

func (s *NodeCriteriaTestSuite) TestNodeFileAccess() {
	node := &storage.Node{
		Name: "test-node-1",
		Id:   "test-node-1",
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
			description: "Node file policy with basic path and operation matching",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/opt/app/config.json",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: false, // Wrong operation
				},
				{
					access:      newActualFileAccessEvent("/opt/app/config.json", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/opt/app/config.json", storage.FileAccess_UNLINK),
					expectAlert: false, // Wrong operation
				},
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false, // Wrong path
				},
			},
		},
		{
			description: "Node file policy with negated file operation",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false, // open is the only event we should ignore
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with multiple negated operations",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, true,
				"/etc/passwd",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple files and single operation",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with multiple files and multiple operations",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT,
				[]storage.FileAccess_Operation{storage.FileAccess_OPEN, storage.FileAccess_CREATE}, false,
				"/etc/passwd", "/etc/shadow",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/shadow", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_CREATE),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/tmp/foo", storage.FileAccess_OPEN),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with no operations (matches all operations)",
			policy:      newFileAccessPolicy(storage.EventSource_NODE_EVENT, nil, false, "/etc/passwd"),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_UNLINK),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_RENAME),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with rename event",
			policy:      newFileAccessPolicy(storage.EventSource_NODE_EVENT, []storage.FileAccess_Operation{storage.FileAccess_RENAME}, false, "/etc/passwd"),
			events: []eventWrapper{
				{
					access:      newRenameFileAccessEvent("/etc/passwd", "/foo/bar"),
					expectAlert: true,
				},
				{
					access:      newRenameFileAccessEvent("/foo/bar", "/etc/passwd"),
					expectAlert: true,
				},
				{
					access:      newRenameFileAccessEvent("/foo/bar", "/bar/baz"),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with wildcards",
			policy: newFileAccessPolicy(storage.EventSource_NODE_EVENT, nil, false,
				"/etc/*", "/home/*/.config/**/*.config", "/home/user/file?", "/home/user/.ssh/id_{rsa,dsa}",
			),
			events: []eventWrapper{
				{
					access:      newActualFileAccessEvent("/etc/passwd", storage.FileAccess_OPEN),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/.config/foo/bar.config", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/.config/foo.config", storage.FileAccess_CREATE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/fileA", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/fileB", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/file", storage.FileAccess_OWNERSHIP_CHANGE),
					expectAlert: false,
				},
				{
					access:      newActualFileAccessEvent("/home/user/.ssh/id_rsa", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/.ssh/id_dsa", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: true,
				},
				{
					access:      newActualFileAccessEvent("/home/user/.ssh/id_ecdsa", storage.FileAccess_PERMISSION_CHANGE),
					expectAlert: false,
				},
			},
		},
		{
			description: "Node file policy with arbitrary path",
			policy:      s.getNodeFileAccessPolicy("/usr/local/bin/app"),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/usr/local/bin/app", storage.FileAccess_OPEN),
					expectAlert: true,
				},
			},
		},
		{
			description: "Node file policy with arbitrary log path",
			policy: s.getNodeFileAccessPolicyWithOperations(
				[]storage.FileAccess_Operation{storage.FileAccess_CREATE}, false,
				"/var/log/audit.log",
			),
			events: []eventWrapper{
				{
					access:      s.getNodeFileAccessEvent("/var/log/audit.log", storage.FileAccess_CREATE),
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
			matcher, err := BuildNodeEventMatcher(tc.policy)
			s.Require().NoError(err)

			for _, event := range tc.events {
				var cache CacheReceptacle
				violations, err := matcher.MatchNodeWithFileAccess(&cache, node, event.access)
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
