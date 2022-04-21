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
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

const (
	reqLabelPolicyID    = "d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6"
	nginxDeploymentYaml = `yamls/nginx.yaml`
)

type ImageLabelPolicyTestSuite struct {
	suite.Suite
	conn          *grpc.ClientConn
	originalState *storage.Policy
	policyService v1.PolicyServiceClient
}

func (suite *ImageLabelPolicyTestSuite) SetupSuite() {
	suite.conn = testutils.GRPCConnectionToCentral(suite.T())

	// Make sure image label policy is enabled
	suite.policyService = v1.NewPolicyServiceClient(suite.conn)
	suite.originalState = getPolicy(suite.T(), suite.policyService, reqLabelPolicyID)
	enabledPolicy := suite.originalState.Clone()
	enabledPolicy.Disabled = false
	setPolicy(suite.T(), suite.policyService, enabledPolicy)
	time.Sleep(5 * time.Second) // Allow time to sync the policy with sensor
}

func (suite *ImageLabelPolicyTestSuite) TeardownSuite() {
	// restore original image label policy
	setPolicy(suite.T(), suite.policyService, suite.originalState)
	if err := suite.conn.Close(); err != nil {
		log.Errorf("Failed to tear down grpc connection %v", err)
	}
}

func (suite *ImageLabelPolicyTestSuite) TestRequiredImageLabelPolicy() {
	// Create a deployment which should violate the policy
	deployment := createDeployment(suite.T(), nginxDeploymentYaml)
	defer deleteDeployment(suite.T(), deployment)

	// Retry to get the new alert
	err := retry.WithRetry(
		func() error {
			// Get all alerts
			alertService := v1.NewAlertServiceClient(suite.conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			alertResp, err := alertService.ListAlerts(ctx, &v1.ListAlertsRequest{})
			cancel()
			require.NoError(suite.T(), err)

			// Make sure there are violations for the test deployment
			var imageLabelAlerts []*storage.ListAlert
			for _, alert := range alertResp.GetAlerts() {
				if alert.GetPolicy().GetId() == reqLabelPolicyID && alert.GetDeployment().GetName() == deployment.GetName() {
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
	require.NoError(suite.T(), err)

	// TODO: Test an image which should not violate the policy
}

func TestImageLabelPolicyTestSuite(t *testing.T) {
	suite.Run(t, new(ImageLabelPolicyTestSuite))
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
