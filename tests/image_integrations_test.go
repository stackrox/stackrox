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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	alpineDeploymentName = `alpine`
	alpineImageSha       = `7df6db5aa61ae9480f52f0b3a06a140ab98d427f86d8d5de0bedab9b8df6b1c0`
)

var (
	integration = &v1.ImageIntegration{
		Name: "public dockerhub",
		Type: "docker",
		IntegrationConfig: &v1.ImageIntegration_Docker{
			Docker: &v1.DockerConfig{
				Endpoint: "registry-1.docker.io",
			},
		},
		Categories: []v1.ImageIntegrationCategory{v1.ImageIntegrationCategory_REGISTRY},
	}
	integrationWithInvalidCluster = &v1.ImageIntegration{
		Name: "public dockerhub",
		Type: "docker",
		IntegrationConfig: &v1.ImageIntegration_Docker{
			Docker: &v1.DockerConfig{
				Endpoint: "registry-1.docker.io",
			},
		},
		Clusters:   []string{"foo"},
		Categories: []v1.ImageIntegrationCategory{v1.ImageIntegrationCategory_REGISTRY},
	}
	integrationWithNoCategories = &v1.ImageIntegration{
		Name: "public dockerhub",
		Type: "docker",
		IntegrationConfig: &v1.ImageIntegration_Docker{
			Docker: &v1.DockerConfig{
				Endpoint: "registry-1.docker.io",
			},
		},
		Clusters:   []string{"remote"},
		Categories: []v1.ImageIntegrationCategory{},
	}
)

func getAlert(service v1.AlertServiceClient, id string) (*v1.Alert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return service.GetAlert(ctx, &v1.ResourceByID{Id: id})
}

func TestImageIntegration(t *testing.T) {
	defer teardownAlpineDeployment(t)
	setupAlpineDeployment(t)

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	subtests := []struct {
		name string
		test func(t *testing.T, conn *grpc.ClientConn)
	}{
		{
			name: "no metadata",
			test: verifyNoMetadata,
		},
		{
			name: "create",
			test: verifyCreateImageIntegration,
		},
		{
			name: "createWithInvalidCluster",
			test: verifyInvalidClusterCreateImageIntegration,
		},
		{
			name: "createWithEmptyCategories",
			test: verifyEmptyCategoriesCreateImageIntegration,
		},
		{
			name: "read",
			test: verifyReadImageIntegration,
		},
		{
			name: "update",
			test: verifyUpdateImageIntegration,
		},
		{
			name: "delete",
			test: verifyDeleteImageIntegration,
		},
		{
			name: "metadata populated",
			test: verifyMetadataPopulated,
		},
	}

	for _, sub := range subtests {
		t.Run(sub.name, func(t *testing.T) {
			sub.test(t, conn)
		})
	}
}

func setupAlpineDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `run`, alpineDeploymentName, `--image=alpine:3.7@sha256:`+alpineImageSha, `--port=22`, `--command=true`, `--`, `sleep`, `1000`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, alpineDeploymentName)
}

func teardownAlpineDeployment(t *testing.T) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, alpineDeploymentName, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForTermination(t, alpineDeploymentName)
}

func verifyNoMetadata(t *testing.T, conn *grpc.ClientConn) {
	if assertion := verifyMetadata(t, conn, func(metadata *v1.ImageMetadata) bool { return metadata == nil }); !assertion {
		t.Error("image metadata is not nil")
	}
}

func verifyMetadataPopulated(t *testing.T, conn *grpc.ClientConn) {
	t.Skip("Skipping metadata populated - AP-391")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if verifyMetadata(t, conn, func(metadata *v1.ImageMetadata) bool { return metadata != nil }) {
				return
			}
		case <-timer.C:
			t.Error("image metadata not populated after 1 minute")
			return
		}
	}
}

func verifyMetadata(t *testing.T, conn *grpc.ClientConn, assertFunc func(*v1.ImageMetadata) bool) bool {
	if assertion := verifyImageMetadata(t, conn, assertFunc); !assertion {
		return false
	}

	deploymentService := v1.NewDeploymentServiceClient(conn)
	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, alpineDeploymentName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	listDeployments, err := deploymentService.ListDeployments(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.NotEmpty(t, listDeployments.GetDeployments())

	deployments, err := retrieveDeployments(deploymentService, listDeployments.GetDeployments())
	if err != nil {
		return false
	}

	for _, d := range deployments {
		require.NotEmpty(t, d.GetContainers())
		c := d.GetContainers()[0]

		if assertion := assertFunc(c.GetImage().GetMetadata()); !assertion {
			return false
		}
	}

	alertService := v1.NewAlertServiceClient(conn)
	qb = search.NewQueryBuilder().AddStrings(search.PolicyName, expectedPort22Policy).AddStrings(search.DeploymentName, alpineDeploymentName)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	alerts, err := alertService.ListAlerts(ctx, &v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	cancel()
	require.NoError(t, err)
	require.NotEmpty(t, alerts.GetAlerts())

	for _, a := range alerts.GetAlerts() {
		alert, err := getAlert(alertService, a.GetId())
		require.NoError(t, err)
		require.NotEmpty(t, alert.GetDeployment().GetContainers())
		c := alert.GetDeployment().GetContainers()[0]

		if assertion := assertFunc(c.GetImage().GetMetadata()); !assertion {
			return false
		}
	}

	return true
}

func verifyImageMetadata(t *testing.T, conn *grpc.ClientConn, assertFunc func(*v1.ImageMetadata) bool) bool {
	imageService := v1.NewImageServiceClient(conn)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			image, err := imageService.GetImage(ctx, &v1.ResourceByID{Id: alpineImageSha})
			cancel()
			if err != nil {
				logger.Error(err)
				continue
			}
			if err == nil && image != nil {
				return assertFunc(image.GetMetadata())
			}
		case <-timer.C:
			logger.Error("Failed to verify image metadata")
			return false
		}
	}
}

func verifyCreateImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	postResp, err := service.PostImageIntegration(ctx, integration)
	cancel()
	require.NoError(t, err)

	integration.Id = postResp.GetId()
	assert.Equal(t, integration, postResp)
}

func verifyReadImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	getResp, err := service.GetImageIntegration(ctx, &v1.ResourceByID{Id: integration.GetId()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, integration, getResp)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	getManyResp, err := service.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{Name: integration.GetName()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, 1, len(getManyResp.GetIntegrations()))
	if len(getManyResp.GetIntegrations()) > 0 {
		assert.Equal(t, integration, getManyResp.GetIntegrations()[0])
	}
}

func verifyUpdateImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)

	integration.Name = "updated docker registry"
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.PutImageIntegration(ctx, integration)
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	getResp, err := service.GetImageIntegration(ctx, &v1.ResourceByID{Id: integration.GetId()})
	cancel()
	require.NoError(t, err)
	assert.Equal(t, integration, getResp)
}

func verifyDeleteImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: integration.GetId()})
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	_, err = service.GetImageIntegration(ctx, &v1.ResourceByID{Id: integration.GetId()})
	cancel()
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}

func verifyInvalidClusterCreateImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.PostImageIntegration(ctx, integrationWithInvalidCluster)
	cancel()
	require.Error(t, err)
}

func verifyEmptyCategoriesCreateImageIntegration(t *testing.T, conn *grpc.ClientConn) {
	service := v1.NewImageIntegrationServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	_, err := service.PostImageIntegration(ctx, integrationWithNoCategories)
	cancel()
	require.Error(t, err)
}
