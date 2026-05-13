package views

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAlertSelectProtosMatchDests(t *testing.T) {
	var sc ListAlertScanner
	dests := sc.Dests()

	require.Equal(t, len(ListAlertSelectProtos), len(dests),
		"ListAlertSelectProtos and Dests() must have the same length")

	expectedOrder := []search.FieldLabel{
		search.AlertID,
		search.LifecycleStage,
		search.ViolationTime,
		search.ViolationState,
		search.PolicyID,
		search.PolicyName,
		search.Severity,
		search.Description,
		search.Category,
		search.EnforcementAction,
		search.EnforcementCount,
		search.EntityType,
		search.ClusterID,
		search.Cluster,
		search.Namespace,
		search.NamespaceID,
		search.DeploymentID,
		search.DeploymentName,
		search.DeploymentType,
		search.Inactive,
		search.NodeID,
		search.Node,
		search.ResourceName,
		search.ResourceType,
	}

	require.Equal(t, len(expectedOrder), len(ListAlertSelectProtos),
		"expectedOrder must match ListAlertSelectProtos length")

	for i, sel := range ListAlertSelectProtos {
		assert.Equal(t, expectedOrder[i].String(), sel.GetField().GetName(),
			"ListAlertSelectProtos[%d] field name mismatch", i)
	}
}

func TestDeploymentTypeOrDefault(t *testing.T) {
	cases := map[string]struct {
		input    pgtype.Text
		expected string
	}{
		"valid value": {
			input:    pgtype.Text{String: "DaemonSet", Valid: true},
			expected: "DaemonSet",
		},
		"null": {
			input:    pgtype.Text{Valid: false},
			expected: "Deployment",
		},
		"empty string": {
			input:    pgtype.Text{String: "", Valid: true},
			expected: "Deployment",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, deploymentTypeOrDefault(tc.input))
		})
	}
}
