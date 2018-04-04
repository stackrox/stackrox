package tests

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestEnforcement(t *testing.T) {
	defer teardownTestEnforcement(t)
	setupTestEnforcement(t)

	verifyDeploymentScaledToZero(t)
	verifyAlertWithEnforcement(t)
}

func setupTestEnforcement(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	togglePolicyEnforcement(t, conn, true)
	setupNginxLatestTagDeploymentForEnforcement(t)
}

func setupNginxLatestTagDeploymentForEnforcement(t *testing.T) {
	cmd := exec.Command(`kubectl`, `run`, nginxDeploymentName, `--image=nginx`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func verifyDeploymentScaledToZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		deployments, err := service.GetDeployments(ctx, &v1.RawQuery{
			Query: getDeploymentQuery(nginxDeploymentName),
		})
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			t.Fatal(err)
		}

		if err == nil && len(deployments.GetDeployments()) > 0 {
			d := deployments.GetDeployments()[0]

			if d.GetReplicas() == 0 {
				return
			}
		}
	}
}

func verifyAlertWithEnforcement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewAlertServiceClient(conn)

	alerts, err := service.GetAlerts(ctx, &v1.GetAlertsRequest{
		Query: getDeploymentQuery(nginxDeploymentName) + "+" + getPolicyQuery(expectedLatestTagPolicy),
		Stale: []bool{true},
	})
	require.NoError(t, err)
	require.NotEmpty(t, alerts.GetAlerts())

	for _, alert := range alerts.GetAlerts() {
		if alert.GetEnforcement().GetAction() == v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT {
			assert.NotEmpty(t, alert.GetEnforcement().GetMessage())
			return
		}
	}

	t.Errorf("no alerts with enforcement found")
}

func teardownTestEnforcement(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	teardownNginxLatestTagDeployment(t)
	togglePolicyEnforcement(t, conn, false)
}

func togglePolicyEnforcement(t *testing.T, conn *grpc.ClientConn, enable bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	service := v1.NewPolicyServiceClient(conn)

	resp, err := service.GetPolicies(ctx, &v1.RawQuery{
		Query: getPolicyQuery(expectedLatestTagPolicy),
	})
	require.NoError(t, err)
	require.Len(t, resp.GetPolicies(), 1)

	p := resp.GetPolicies()[0]

	if enable {
		p.Enforcement = v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
	} else {
		p.Enforcement = v1.EnforcementAction_UNSET_ENFORCEMENT
	}

	_, err = service.PutPolicy(ctx, p)
	require.NoError(t, err)
}
