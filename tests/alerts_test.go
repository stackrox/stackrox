package tests

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nginxDeploymentName     = `nginx`
	expectedLatestTagPolicy = `Latest tag`
	expectedPort22Policy    = `Container Port 22`
	expectedSecretEnvPolicy = `Don't use environment variables with secrets`

	waitTimeout = 2 * time.Minute
)

var alertQuery = func() string {
	return search.NewQueryBuilder().AddStrings(search.DeploymentName, nginxDeploymentName).AddStrings(search.LabelKey, "hello").AddStrings(search.LabelValue, "world").AddBools(search.Stale, false).Query()
}()

var (
	alertRequestOptions = v1.ListAlertsRequest{
		Query: alertQuery,
	}
)

func TestAlerts(t *testing.T) {
	defer teardownNginxDeployment(t)
	setupNginxDeployment(t)

	defer revertPolicyScopeChange(t, expectedPort22Policy)
	addPolicyClusterScope(t, expectedPort22Policy)

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
	cmd := exec.Command(`kubectl`, `run`, nginxDeploymentName, `--image=nginx`, `--port=22`, `--env=SECRET=true`, `--labels=hello=world`, `--limits=cpu=10m,memory=50M`, `--requests=cpu=10m,memory=50M`)
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

func retrieveDeployment(service v1.DeploymentServiceClient, listDeployment *v1.ListDeployment) (*v1.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.GetDeployment(ctx, &v1.ResourceByID{Id: listDeployment.GetId()})
}

func retrieveDeployments(service v1.DeploymentServiceClient, deps []*v1.ListDeployment) ([]*v1.Deployment, error) {
	deployments := make([]*v1.Deployment, 0, len(deps))
	for _, d := range deps {
		deployment, err := retrieveDeployment(service, d)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

func waitForDeployment(t *testing.T, deploymentName string) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentName)

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			listDeployments, err := service.ListDeployments(ctx, &v1.RawQuery{
				Query: qb.Query(),
			},
			)
			cancel()
			if err != nil {
				logger.Errorf("Error listing deployments: %s", err)
				continue
			}

			deployments, err := retrieveDeployments(service, listDeployments.GetDeployments())
			if err != nil {
				logger.Errorf("Error retrieving deployments: %s", err)
				continue
			}

			if err == nil && len(deployments) > 0 {
				d := deployments[0]

				if len(d.GetContainers()) > 0 {
					return
				}
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for deployment %s", deploymentName)
		}
	}
}

func waitForTermination(t *testing.T, deploymentName string) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	query := search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentName).Query()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			listDeployments, err := service.ListDeployments(ctx, &v1.RawQuery{
				Query: query,
			})
			cancel()
			if err != nil {
				logger.Error(err)
				continue
			}

			if len(listDeployments.GetDeployments()) == 0 {
				return
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for deployment %s to stop", deploymentName)
		}
	}
}

func addPolicyClusterScope(t *testing.T, policyName string) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	clusterService := v1.NewClustersServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	clusters, err := clusterService.GetClusters(ctx, &v1.Empty{})
	cancel()
	require.NoError(t, err)
	require.Len(t, clusters.GetClusters(), 1)

	c := clusters.GetClusters()[0]
	clusterID := c.GetId()

	policyService := v1.NewPolicyServiceClient(conn)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	policyResp, err := policyService.ListPolicies(ctx, &v1.RawQuery{
		Query: search.NewQueryBuilder().AddStrings(search.PolicyName, policyName).Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, policyResp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	policy, err := policyService.GetPolicy(ctx, &v1.ResourceByID{
		Id: policyResp.GetPolicies()[0].GetId(),
	})

	policy.Scope = append(policy.Scope, &v1.Scope{
		Cluster: clusterID,
	})
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = policyService.PutPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)
}

func revertPolicyScopeChange(t *testing.T, policyName string) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	policyService := v1.NewPolicyServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	policyResp, err := policyService.ListPolicies(ctx, &v1.RawQuery{
		Query: search.NewQueryBuilder().AddStrings(search.PolicyName, policyName).Query(),
	})
	cancel()
	require.NoError(t, err)
	require.Len(t, policyResp.GetPolicies(), 1)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	policy, err := policyService.GetPolicy(ctx, &v1.ResourceByID{
		Id: policyResp.GetPolicies()[0].GetId(),
	})
	policy.Scope = policy.Scope[:len(policy.GetScope())-1]

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = policyService.PutPolicy(ctx, policy)
	cancel()
	require.NoError(t, err)
}

func verifyStaleAlerts(t *testing.T) {
	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewAlertServiceClient(conn)
	request := alertRequestOptions
	request.Query = search.NewQueryBuilder().AddBools(search.Stale, true).Query()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alerts, err := service.ListAlerts(ctx, &request)
	cancel()
	require.NoError(t, err)
	assert.NotEmpty(t, alerts.GetAlerts())
}

func verifyAlerts(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alerts, err := service.ListAlerts(ctx, &alertRequestOptions)
	cancel()
	require.NoError(t, err)
	assert.Len(t, alerts.GetAlerts(), 3)

	alertMap := make(map[string]*v1.ListAlert)
	for _, a := range alerts.GetAlerts() {
		if n := a.GetPolicy().GetName(); n == expectedLatestTagPolicy || n == expectedPort22Policy || n == expectedSecretEnvPolicy {

			alertMap[a.GetId()] = a
		}
	}
	require.Len(t, alertMap, 3)

	for id := range alertMap {
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
		_, err := service.GetAlert(ctx, &v1.ResourceByID{Id: id})
		cancel()
		require.NoError(t, err)
	}
}

func verifyAlertCounts(t *testing.T, service v1.AlertServiceClient) {
	// Ungrouped
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alerts, err := service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions})
	cancel()
	require.NoError(t, err)
	require.Len(t, alerts.GetGroups(), 1)
	assert.NotEmpty(t, alerts.GetGroups()[0].GetCounts())

	// Group by cluster.
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	alertCounts, err := service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions, GroupBy: v1.GetAlertsCountsRequest_CLUSTER})
	cancel()
	require.NoError(t, err)

	require.Len(t, alertCounts.GetGroups(), 1)
	group := alertCounts.GetGroups()[0]

	// TODO(cg): Consider verifying the cluster ID that is returned.
	// Doing so would require either putting with a specific ID during setup, or getting it here.
	assert.NotEmpty(t, group.GetCounts())

	// Group by category.
	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	alertCounts, err = service.GetAlertsCounts(ctx, &v1.GetAlertsCountsRequest{Request: &alertRequestOptions, GroupBy: v1.GetAlertsCountsRequest_CATEGORY})
	cancel()
	require.NoError(t, err)

	require.Len(t, alertCounts.GetGroups(), 2)

	var securityGroup, devopsGroups *v1.GetAlertsCountsResponse_AlertGroup

	for _, g := range alertCounts.GetGroups() {
		if g.Group == "Security Best Practices" {
			securityGroup = g
		} else if g.Group == "DevOps Best Practices" {
			devopsGroups = g
		}
	}

	require.NotNil(t, securityGroup)
	require.NotNil(t, devopsGroups)

	assert.NotEmpty(t, securityGroup.GetCounts())
	assert.NotEmpty(t, devopsGroups.GetCounts())
}

func verifyAlertGroups(t *testing.T, service v1.AlertServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	alerts, err := service.GetAlertsGroup(ctx, &alertRequestOptions)
	cancel()
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
	alerts, err := service.ListAlerts(ctx, &alertRequestOptions)
	cancel()
	require.NoError(t, err)

	alertMap := make(map[string]*v1.ListAlert)
	for _, a := range alerts.GetAlerts() {
		if n := a.GetPolicy().GetName(); n == expectedLatestTagPolicy || n == expectedPort22Policy || n == expectedSecretEnvPolicy {
			alertMap[a.GetId()] = a
		}
	}
	require.Len(t, alertMap, 3)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	timeseries, err := service.GetAlertTimeseries(ctx, &alertRequestOptions)
	cancel()
	require.NoError(t, err)
	require.Len(t, timeseries.Clusters, 1)
	cluster := timeseries.Clusters[0]

	numCreatedEvents := 0

	for _, alertGroups := range cluster.GetSeverities() {
		for _, e := range alertGroups.GetEvents() {
			if e.Type == v1.Type_CREATED {
				numCreatedEvents++
			}
			if alert, ok := alertMap[e.GetId()]; ok && e.Type == v1.Type_CREATED {
				assert.Equal(t, alert.GetTime().GetSeconds()*1000, e.GetTime())
				assert.Equal(t, alert.GetPolicy().GetSeverity(), alertGroups.GetSeverity())
			}
		}
	}

	assert.Equal(t, numCreatedEvents, 3)
}
