package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestWhitelist(t *testing.T) {
	defer teardownTestWhitelist(t)
	setupNginxLatestTagDeployment(t)
	verifyNoAlertForWhitelist(t)
	verifyAlertForWhitelistRemoval(t)
}

func waitForAlert(t *testing.T, service v1.AlertServiceClient, req *v1.ListAlertsRequest, desired int) {
	var alerts []*storage.ListAlert
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		resp, err := service.ListAlerts(ctx, req)
		cancel()
		require.NoError(t, err)
		alerts = resp.GetAlerts()
		if len(alerts) == desired {
			return
		}
		time.Sleep(2 * time.Second)
	}
	alertStrings := ""
	for _, alert := range alerts {
		alertStrings = fmt.Sprintf("%s%s\n", alertStrings, proto.MarshalTextString(alert))
	}
	log.Infof("Received alerts:\n%s", alertStrings)
	require.Fail(t, fmt.Sprintf("Failed to have %d alerts, instead received %d alerts", desired, len(alerts)))
}

func verifyNoAlertForWhitelist(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	service := v1.NewPolicyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy).Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	latestPolicy, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()
	require.NoError(t, err)

	latestPolicy.Whitelists = []*storage.Whitelist{
		{
			Deployment: &storage.Whitelist_Deployment{
				Name: nginxDeploymentName,
			},
		},
	}
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, nginxDeploymentName).AddStrings(search.PolicyName, latestPolicy.GetName()).AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	alertService := v1.NewAlertServiceClient(conn)
	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 0)
}

func verifyAlertForWhitelistRemoval(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	service := v1.NewPolicyServiceClient(conn)

	qb := search.NewQueryBuilder().AddStrings(search.PolicyName, expectedLatestTagPolicy)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	resp, err := service.ListPolicies(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	latestPolicy, err := service.GetPolicy(ctx, &v1.ResourceByID{
		Id: resp.GetPolicies()[0].GetId(),
	})
	cancel()
	require.NoError(t, err)

	latestPolicy.Whitelists = nil
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.PutPolicy(ctx, latestPolicy)
	cancel()
	require.NoError(t, err)

	alertService := v1.NewAlertServiceClient(conn)

	qb = search.NewQueryBuilder().AddStrings(search.DeploymentName, nginxDeploymentName).AddStrings(search.PolicyName, latestPolicy.GetName()).AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	waitForAlert(t, alertService, &v1.ListAlertsRequest{
		Query: qb.Query(),
	}, 1)
}

func teardownTestWhitelist(t *testing.T) {
	teardownNginxLatestTagDeployment(t)
}
