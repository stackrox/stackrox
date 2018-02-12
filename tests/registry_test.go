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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	alpineDeploymentName = `alpine`
	alpineImageSha       = `7df6db5aa61ae9480f52f0b3a06a140ab98d427f86d8d5de0bedab9b8df6b1c0`
)

var (
	dockerRegistry = &v1.Registry{
		Name:          "public dockerhub",
		Type:          "docker",
		Endpoint:      "registry-1.docker.io",
		ImageRegistry: "docker.io",
	}
)

func TestRegistry(t *testing.T) {
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
			test: verifyCreateRegistry,
		},
		{
			name: "read",
			test: verifyReadRegistry,
		},
		{
			name: "update",
			test: verifyUpdateRegistry,
		},
		{
			name: "delete",
			test: verifyDeleteRegistry,
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
	verifyMetadata(t, conn, false)
}

func verifyMetadataPopulated(t *testing.T, conn *grpc.ClientConn) {
	verifyMetadata(t, conn, true)
}

func verifyMetadata(t *testing.T, conn *grpc.ClientConn, metadata bool) {
	verifyImageMetadata(t, conn, metadata)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	deploymentService := v1.NewDeploymentServiceClient(conn)
	deployments, err := deploymentService.GetDeployments(ctx, &v1.GetDeploymentsRequest{Name: []string{alpineDeploymentName}})
	require.NoError(t, err)
	require.NotEmpty(t, deployments.GetDeployments())

	for _, d := range deployments.GetDeployments() {
		require.NotEmpty(t, d.GetContainers())
		c := d.GetContainers()[0]

		if metadata {
			assert.NotNil(t, c.GetImage().GetMetadata())
		} else {
			assert.Nil(t, c.GetImage().GetMetadata())
		}
	}

	alertService := v1.NewAlertServiceClient(conn)

	alerts, err := alertService.GetAlerts(ctx, &v1.GetAlertsRequest{
		PolicyName: []string{expectedPort22Policy}, DeploymentName: []string{alpineDeploymentName}, Stale: []bool{false}})
	require.NoError(t, err)
	require.NotEmpty(t, alerts.GetAlerts())

	for _, a := range alerts.GetAlerts() {
		require.NotEmpty(t, a.GetDeployment().GetContainers())
		c := a.GetDeployment().GetContainers()[0]

		if metadata {
			assert.NotNil(t, c.GetImage().GetMetadata())
		} else {
			assert.Nil(t, c.GetImage().GetMetadata())
		}
	}
}

func verifyImageMetadata(t *testing.T, conn *grpc.ClientConn, metadata bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	imageService := v1.NewImageServiceClient(conn)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		image, err := imageService.GetImage(ctx, &v1.ResourceByID{Id: alpineImageSha})
		if err != nil && ctx.Err() == context.DeadlineExceeded {
			t.Error(err)
			return
		}

		if err == nil && image != nil {
			if metadata && image.GetMetadata() != nil {
				return
			} else if !metadata && image.GetMetadata() == nil {
				return
			}
		}
	}
}

func verifyCreateRegistry(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	service := v1.NewRegistryServiceClient(conn)

	postResp, err := service.PostRegistry(ctx, dockerRegistry)
	require.NoError(t, err)

	dockerRegistry.Id = postResp.GetId()
	assert.Equal(t, dockerRegistry, postResp)
}

func verifyReadRegistry(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	service := v1.NewRegistryServiceClient(conn)

	getResp, err := service.GetRegistry(ctx, &v1.ResourceByID{Id: dockerRegistry.GetId()})
	require.NoError(t, err)
	assert.Equal(t, dockerRegistry, getResp)

	getManyResp, err := service.GetRegistries(ctx, &v1.GetRegistriesRequest{Name: dockerRegistry.GetName()})
	require.NoError(t, err)
	assert.Equal(t, 1, len(getManyResp.GetRegistries()))
	if len(getManyResp.GetRegistries()) > 0 {
		assert.Equal(t, dockerRegistry, getManyResp.GetRegistries()[0])
	}
}

func verifyUpdateRegistry(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	service := v1.NewRegistryServiceClient(conn)

	dockerRegistry.Name = "updated docker registry"

	_, err := service.PutRegistry(ctx, dockerRegistry)
	require.NoError(t, err)

	getResp, err := service.GetRegistry(ctx, &v1.ResourceByID{Id: dockerRegistry.GetId()})
	require.NoError(t, err)
	assert.Equal(t, dockerRegistry, getResp)
}

func verifyDeleteRegistry(t *testing.T, conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	service := v1.NewRegistryServiceClient(conn)

	_, err := service.DeleteRegistry(ctx, &v1.ResourceByID{Id: dockerRegistry.GetId()})
	require.NoError(t, err)

	_, err = service.GetRegistry(ctx, &v1.ResourceByID{Id: dockerRegistry.GetId()})
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, s.Code())
}
