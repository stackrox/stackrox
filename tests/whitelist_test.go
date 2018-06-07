package tests

import (
	"context"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/central/search"
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

func waitForAlert(t *testing.T, service v1.AlertServiceClient, req *v1.ListAlertsRequest, desired int) {
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		resp, err := service.ListAlerts(ctx, req)
		cancel()
		require.NoError(t, err)
		if len(resp.GetAlerts()) == desired {
			return
		}
		time.Sleep(2 * time.Second)
	}
	require.Failf(t, "failed waiting for alerts", "Failed to have %d alerts", desired)
}

func verifyNoAlertForWhitelist(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)
	qb := search.NewQueryBuilder().AddString(search.PolicyName, expectedLatestTagPolicy)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.GetPolicies(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	cancel()
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
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	qb = search.NewQueryBuilder().AddString(search.DeploymentName, nginxDeploymentName).AddString(search.PolicyName, latestPolicy.GetName()).AddBool(search.Stale, false)

	alertService := v1.NewAlertServiceClient(conn)

	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 0)
}

func verifyAlertForWhitelistRemoval(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewPolicyServiceClient(conn)
	qb := search.NewQueryBuilder().AddString(search.PolicyName, expectedLatestTagPolicy)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.GetPolicies(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)

	latestPolicy := resp.Policies[0]
	latestPolicy.Whitelists = nil
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	alertService := v1.NewAlertServiceClient(conn)

	qb = search.NewQueryBuilder().AddString(search.DeploymentName, nginxDeploymentName).AddString(search.PolicyName, latestPolicy.GetName()).AddBool(search.Stale, false)
	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 1)
}

func teardownTestWhitelist(t *testing.T) {
	teardownNginxLatestTagDeployment(t)
}
