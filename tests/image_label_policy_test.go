package tests

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestRequiredImageLabelPolicy(t *testing.T) {
	id := "d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6"
	conn := testutils.GRPCConnectionToCentral(t)

	// Create a deployment which should violate the policy
	setupNginxLatestTagDeployment(t)
	defer teardownNginxLatestTagDeployment(t)

	// Make sure image label policy is enabled
	policyService := v1.NewPolicyServiceClient(conn)
	originalState := getPolicy(t, policyService, id)
	enabledPolicy := proto.Clone(originalState).(*storage.Policy)
	enabledPolicy.Disabled = false
	setPolicy(t, policyService, enabledPolicy)
	defer setPolicy(t, policyService, originalState)

	// Get all alerts
	alertService := v1.NewAlertServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alertResp, err := alertService.ListAlerts(ctx, &v1.ListAlertsRequest{})
	cancel()
	require.NoError(t, err)

	// Make sure there are violations for the test deployment
	var imageLabelAlerts []*storage.ListAlert
	for _, alert := range alertResp.GetAlerts() {
		if alert.GetPolicy().GetId() == id && alert.GetDeployment().GetName() == nginxDeploymentName {
			imageLabelAlerts = append(imageLabelAlerts, alert)
		}
	}
	require.NotEmpty(t, imageLabelAlerts)

	// TODO: Test an image which should not violate the policy
}

func getPolicy(t *testing.T, client v1.PolicyServiceClient, policyID string) *storage.Policy {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	policy, err := client.GetPolicy(ctx, &v1.ResourceByID{Id: policyID})
	cancel()
	require.NoError(t, err, "Could not find policy for ID %s", policyID)

	return policy
}

func setPolicy(t *testing.T, client v1.PolicyServiceClient, policy *storage.Policy) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	// This API call does not return until new alerts have been generated, no need for additional waiting.
	_, err := client.PutPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)
}
