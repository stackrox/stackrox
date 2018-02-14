package tests

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/stretchr/testify/require"
)

func TestWhitelist(t *testing.T) {
	defer teardownTestWhitelist(t)
	setupNginxLatestTagDeployment(t)
	verifyNoAlertForWhitelist(t)
	verifyAlertForWhitelistRemoval(t)
}

func waitForAlert(t *testing.T, service v1.AlertServiceClient, req *v1.GetAlertsRequest, desired int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for i := 0; i < 5; i++ {
		resp, err := service.GetAlerts(ctx, req)
		require.NoError(t, err)
		if len(resp.GetAlerts()) == desired {
			return
		}
		time.Sleep(2 * time.Second)
	}
	require.Fail(t, "Failed to have %d alerts", desired)
}

func verifyNoAlertForWhitelist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)
	resp, err := service.GetPolicies(ctx, &v1.GetPoliciesRequest{Name: []string{expectedLatestTagPolicy}})
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)

	latestPolicy := resp.Policies[0]
	latestPolicy.Whitelists = []*v1.Whitelist{
		{
			Deployment: &v1.Whitelist_Deployment{
				Name: nginxDeploymentName,
			},
		},
	}
	_, err = service.PutPolicy(ctx, latestPolicy)
	require.NoError(t, err)

	alertService := v1.NewAlertServiceClient(conn)
	waitForAlert(t, alertService, &v1.GetAlertsRequest{
		DeploymentName: []string{nginxDeploymentName},
		PolicyId:       []string{latestPolicy.GetId()},
		Stale:          []bool{false},
	}, 0)
}

func verifyAlertForWhitelistRemoval(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)
	resp, err := service.GetPolicies(ctx, &v1.GetPoliciesRequest{Name: []string{expectedLatestTagPolicy}})
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)

	latestPolicy := resp.Policies[0]
	latestPolicy.Whitelists = nil
	_, err = service.PutPolicy(ctx, latestPolicy)
	require.NoError(t, err)

	alertService := v1.NewAlertServiceClient(conn)
	waitForAlert(t, alertService, &v1.GetAlertsRequest{
		DeploymentName: []string{nginxDeploymentName},
		PolicyId:       []string{latestPolicy.GetId()},
		Stale:          []bool{false},
	}, 1)
}

func teardownTestWhitelist(t *testing.T) {
	teardownNginxLatestTagDeployment(t)
}
