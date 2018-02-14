package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewClustersServiceClient(conn)

	clusters, err := service.GetClusters(ctx, &empty.Empty{})
	require.NoError(t, err)
	require.Len(t, clusters.GetClusters(), 1)

	c := clusters.GetClusters()[0]
	assert.Equal(t, v1.ClusterType_KUBERNETES_CLUSTER, c.GetType())
	assert.Equal(t, `remote`, c.GetName())

	img := images.GenerateImageFromString(c.GetMitigateImage())
	assert.Equal(t, `stackrox/mitigate`, img.GetName().GetRemote())
	if sha, ok := os.LookupEnv(`CIRCLE_SHA1`); ok {
		assert.Equal(t, sha, img.GetName().GetTag())
	}

	cByID, err := service.GetCluster(ctx, &v1.ResourceByID{Id: c.GetId()})
	require.NoError(t, err)

	cByID.GetCluster().LastContact = c.GetLastContact()
	assert.Equal(t, c, cByID.GetCluster())
	assert.NotEmpty(t, cByID.GetDeploymentYaml())
	assert.NotEmpty(t, cByID.GetDeploymentCommand())
}

func TestDeployments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	deployments, err := service.GetDeployments(ctx, &v1.GetDeploymentsRequest{Name: []string{"central", "sensor"}})
	require.NoError(t, err)
	require.Len(t, deployments.GetDeployments(), 2)

	var centralDeployment, sensorDeployment *v1.Deployment

	for _, d := range deployments.GetDeployments() {
		if d.GetName() == `central` {
			centralDeployment = d
		} else if d.GetName() == `sensor` {
			sensorDeployment = d
		}
	}

	require.NotNil(t, centralDeployment)
	require.NotNil(t, sensorDeployment)

	verifyCentralDeployment(t, centralDeployment)
	verifySensorDeployment(t, sensorDeployment)

	centralByID, err := service.GetDeployment(ctx, &v1.ResourceByID{Id: centralDeployment.GetId()})
	require.NoError(t, err)
	assert.Equal(t, centralDeployment, centralByID)

	sensorByID, err := service.GetDeployment(ctx, &v1.ResourceByID{Id: sensorDeployment.GetId()})
	require.NoError(t, err)
	assert.Equal(t, sensorDeployment, sensorByID)
}

func verifyCentralDeployment(t *testing.T, centralDeployment *v1.Deployment) {
	verifyDeployment(t, centralDeployment)
	assert.Equal(t, "central", centralDeployment.GetLabels()["app"])

	require.Len(t, centralDeployment.GetContainers(), 1)
	c := centralDeployment.GetContainers()[0]

	assert.Equal(t, `stackrox/mitigate`, c.GetImage().GetName().GetRemote())
	if sha, ok := os.LookupEnv(`CIRCLE_SHA1`); ok {
		assert.Equal(t, sha, c.GetImage().GetName().GetTag())
	}

	require.Len(t, c.GetVolumes(), 1)
	v := c.GetVolumes()[0]
	assert.Equal(t, "/run/secrets/stackrox.io/", v.GetPath())
	assert.True(t, v.GetReadOnly())

	require.Len(t, c.GetPorts(), 1)
	p := c.GetPorts()[0]
	assert.Equal(t, int32(443), p.GetContainerPort())
	assert.Equal(t, "TCP", p.GetProtocol())
}

func verifySensorDeployment(t *testing.T, sensorDeployment *v1.Deployment) {
	verifyDeployment(t, sensorDeployment)
	assert.Equal(t, "sensor", sensorDeployment.GetLabels()["app"])

	require.Len(t, sensorDeployment.GetContainers(), 1)
	c := sensorDeployment.GetContainers()[0]

	assert.Equal(t, `stackrox/mitigate`, c.GetImage().GetName().GetRemote())
	if sha, ok := os.LookupEnv(`CIRCLE_SHA1`); ok {
		assert.Equal(t, sha, c.GetImage().GetName().GetTag())
	}

	require.Len(t, c.GetVolumes(), 1)
	v := c.GetVolumes()[0]
	assert.Equal(t, "/run/secrets/stackrox.io/", v.GetPath())
	assert.True(t, v.GetReadOnly())
}

func verifyDeployment(t *testing.T, deployment *v1.Deployment) {
	assert.Equal(t, "Deployment", deployment.GetType())
	assert.Equal(t, int64(1), deployment.GetReplicas())
	assert.NotEmpty(t, deployment.GetId())
	assert.NotEmpty(t, deployment.GetVersion())
	assert.NotEmpty(t, deployment.GetUpdatedAt())
}

func TestImages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
	require.NoError(t, err)

	service := v1.NewImageServiceClient(conn)

	images, err := service.GetImages(ctx, &v1.GetImagesRequest{})
	require.NoError(t, err)

	require.NotEmpty(t, images.GetImages())

	imageMap := make(map[string][]*v1.Image)
	for _, img := range images.GetImages() {
		imageMap[img.GetName().GetRegistry()] = append(imageMap[img.GetName().GetRegistry()], img)
	}

	const dockerRegistry = `docker.io`

	require.NotEmpty(t, imageMap[dockerRegistry])

	foundMitigateImage := false

	for _, img := range imageMap[dockerRegistry] {
		if img.GetName().GetRemote() == `stackrox/mitigate` {
			foundMitigateImage = true

			if sha, ok := os.LookupEnv(`CIRCLE_SHA1`); ok {
				assert.Equal(t, sha, img.GetName().GetTag())
			}
		}
	}

	assert.True(t, foundMitigateImage)
}
