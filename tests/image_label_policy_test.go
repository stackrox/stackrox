package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

const (
	reqLabelPolicyID = "d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6"
)

func TestRequiredImageLabelPolicy(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	// Make sure image label policy is enabled
	policyService := v1.NewPolicyServiceClient(conn)
	originalState := getPolicy(t, policyService, reqLabelPolicyID)
	enabledPolicy := originalState.Clone()
	enabledPolicy.Disabled = false
	setPolicy(t, policyService, enabledPolicy)
	defer setPolicy(t, policyService, originalState)
	time.Sleep(5 * time.Second) // Allow time to sync the policy with sensor

	// Create a deployment which should violate the policy
	setupNginxLatestTagDeployment(t)
	defer teardownNginxLatestTagDeployment(t)

	// Retry to get the new alert
	err := retry.WithRetry(
		func() error {
			// Get all alerts
			alertService := v1.NewAlertServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			alertResp, err := alertService.ListAlerts(ctx, &v1.ListAlertsRequest{})
			cancel()
			require.NoError(t, err)

			// Make sure there are violations for the test deployment
			var imageLabelAlerts []*storage.ListAlert
			for _, alert := range alertResp.GetAlerts() {
				if alert.GetPolicy().GetId() == reqLabelPolicyID && alert.GetDeployment().GetName() == nginxDeploymentName {
					imageLabelAlerts = append(imageLabelAlerts, alert)
				}
			}
			if len(imageLabelAlerts) == 0 {
				return errors.New("No image label alerts found")
			}
			return nil
		},
		// Upper bound should be 60 seconds, first try happens immediately so this will wait up to 65 seconds
		retry.Tries(66),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(time.Second)
		}),
	)
	require.NoError(t, err)

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
	_, err := client.PutPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)
}
