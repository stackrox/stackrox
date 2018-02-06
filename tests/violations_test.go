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
)

const (
	nginxDeploymentName     = `nginx`
	expectedLatestTagPolicy = `Latest tag`
	expectedPort22Policy    = `Container Port 22`
	expectedSecretEnvPolicy = `Don't use environment variables with secrets`
)

var (
	alertRequestOptions = v1.GetAlertsRequest{DeploymentName: []string{nginxDeploymentName}, Stale: []bool{false}}
)

func TestAlerts(t *testing.T) {
	defer teardownNginxDeployment(t)
	setupNginxDeployment(t)

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewAlertServiceClient(conn)

	subtests := []struct {
		name string
		test func(t *testing.T, service v1.AlertServiceClient)
	}{
		{
			name: "alerts",
			test: verifyAlerts,
		},
		{
			name: "alertCounts",
			test: verifyAlertCounts,
		},
		{
			name: "alertGroups",
			test: verifyAlertGroups,
		},
		{
			name: "alertTimeseries",
			test: verifyAlertTimeseries,
		},
	}

	for _, sub := range subtests {
		t.Run(sub.name, func(t *testing.T) {
			sub.test(t, service)
		})
	}
}

func setupNginxDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `run`, nginxDeploymentName, `--image=nginx`, `--port=22`, `--env=SECRET=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, nginxDeploymentName)
}

func teardownNginxDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, nginxDeploymentName, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	if !t.Failed() {
		waitForTermination(t, nginxDeploymentName)
		t.Run("staleAlerts", verifyStaleAlerts)
	}
}

func waitForDeployment(t *testing.T, deploymentName string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		deployments, err := service.GetDeployments(ctx, &v1.GetDeploymentsRequest{Name: []string{deploymentName}})
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			t.Fatal(err)
		}

		if err == nil && len(deployments.GetDeployments()) > 0 {
			d := deployments.GetDeployments()[0]

			if len(d.GetContainers()) > 0 && d.GetContainers()[0].GetImage().GetSha() != "" {
				return
			}
		}
	}
}

func waitForTermination(t *testing.T, deploymentName string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		deployments, err := service.GetDeployments(ctx, &v1.GetDeploymentsRequest{Name: []string{deploymentName}})
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			t.Fatal(err)
		}

		if err == nil && len(deployments.GetDeployments()) == 0 {
			return
		}
	}
}

func verifyStaleAlerts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewAlertServiceClient(conn)
	request := alertRequestOptions
	request.Stale = []bool{true}

	alerts, err := service.GetAlerts(ctx, &request)
	require.NoError(t, err)
	assert.NotEmpty(t, alerts.GetAlerts())
}

func verifyAlerts(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	alerts, err := service.GetAlerts(ctx, &alertRequestOptions)
	require.NoError(t, err)
	assert.Len(t, alerts.GetAlerts(), 3)

	alertMap := make(map[string]*v1.Alert)
	for _, a := range alerts.GetAlerts() {
		if n := a.GetPolicy().GetName(); n == expectedLatestTagPolicy || n == expectedPort22Policy || n == expectedSecretEnvPolicy {
			alertMap[a.GetId()] = a
		}
	}
	require.Len(t, alertMap, 3)

	for id, expected := range alertMap {
		a, err := service.GetAlert(ctx, &v1.ResourceByID{Id: id})
		require.NoError(t, err)

		assert.Equal(t, expected, a)
	}
}

func verifyAlertCounts(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Ungrouped
	alerts, err := service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions})
	require.NoError(t, err)
	require.Len(t, alerts.GetGroups(), 1)
	assert.NotEmpty(t, alerts.GetGroups()[0].GetCounts())

	// Group by cluster.
	alertCounts, err := service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions, GroupBy: v1.GetAlertsCountsRequest_CLUSTER})
	require.NoError(t, err)

	require.Len(t, alertCounts.GetGroups(), 1)
	group := alertCounts.GetGroups()[0]

	assert.Equal(t, "remote", group.GetGroup())
	assert.NotEmpty(t, group.GetCounts())

	// Group by category.
	alertCounts, err = service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions, GroupBy: v1.GetAlertsCountsRequest_CATEGORY})
	require.NoError(t, err)

	require.Len(t, alertCounts.GetGroups(), 2)

	var imageGroup, containerGroup *v1.GetAlertsCountsResponse_AlertGroup

	for _, g := range alertCounts.GetGroups() {
		if g.Group == v1.Policy_Category_IMAGE_ASSURANCE.String() {
			imageGroup = g
		} else if g.Group == v1.Policy_Category_CONTAINER_CONFIGURATION.String() {
			containerGroup = g
		}
	}

	require.NotNil(t, imageGroup)
	require.NotNil(t, containerGroup)

	assert.NotEmpty(t, imageGroup.GetCounts())
	assert.NotEmpty(t, containerGroup.GetCounts())
}

func verifyAlertGroups(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	alerts, err := service.GetAlertsGroup(ctx, &alertRequestOptions)
	require.NoError(t, err)

	require.True(t, len(alerts.GetAlertsByPolicies()) >= 3)

	var tagPolicyAlerts, portPolicyAlerts, secretPolicyAlerts int64

	for _, group := range alerts.GetAlertsByPolicies() {
		switch group.GetPolicy().GetName() {
		case expectedLatestTagPolicy:
			tagPolicyAlerts = group.GetNumAlerts()
		case expectedPort22Policy:
			portPolicyAlerts = group.GetNumAlerts()
		case expectedSecretEnvPolicy:
			secretPolicyAlerts = group.GetNumAlerts()
		}
	}

	assert.NotZero(t, tagPolicyAlerts)
	assert.NotZero(t, portPolicyAlerts)
	assert.NotZero(t, secretPolicyAlerts)
}

func verifyAlertTimeseries(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	alerts, err := service.GetAlerts(ctx, &alertRequestOptions)
	require.NoError(t, err)

	alertMap := make(map[string]*v1.Alert)
	for _, a := range alerts.GetAlerts() {
		if n := a.GetPolicy().GetName(); n == expectedLatestTagPolicy || n == expectedPort22Policy || n == expectedSecretEnvPolicy {
			alertMap[a.GetId()] = a
		}
	}
	require.Len(t, alertMap, 3)

	timeseries, err := service.GetAlertTimeseries(ctx, &alertRequestOptions)
	require.NoError(t, err)

	assert.True(t, len(timeseries.GetEvents()) >= 3)

	numCreatedEvents := 0

	for _, e := range timeseries.GetEvents() {
		if e.Type == v1.Type_CREATED {
			numCreatedEvents++
		}

		if alert, ok := alertMap[e.GetId()]; ok && e.Type == v1.Type_CREATED {
			assert.Equal(t, alert.GetTime().GetSeconds()*1000, e.GetTime())
			assert.Equal(t, alert.GetPolicy().GetSeverity(), e.GetSeverity())
		}
	}

	assert.Equal(t, numCreatedEvents, 3)
}
