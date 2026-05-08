package views

import (
	"testing"

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
