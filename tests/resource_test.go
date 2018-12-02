package tests

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	imageName   = `stackrox/main`
	imageTagEnv = `MAIN_IMAGE_TAG`
)

func TestClusters(t *testing.T) {

	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	clusters, err := service.GetClusters(ctx, &v1.Empty{})
	cancel()
	require.NoError(t, err)
	require.Len(t, clusters.GetClusters(), 1)

	c := clusters.GetClusters()[0]
	assert.Equal(t, v1.ClusterType_KUBERNETES_CLUSTER, c.GetType())
	assert.Equal(t, `remote`, c.GetName())

	img := utils.GenerateImageFromString(c.GetMainImage())
	assert.Equal(t, imageName, img.GetName().GetRemote())
	if sha, ok := os.LookupEnv(`MAIN_IMAGE_TAG`); ok {
		assert.Equal(t, sha, img.GetName().GetTag())
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	cByID, err := service.GetCluster(ctx, &v1.ResourceByID{Id: c.GetId()})
	cancel()
	require.NoError(t, err)

	cByID.GetCluster().LastContact = c.GetLastContact()
	assert.Equal(t, c, cByID.GetCluster())
}

func TestDeployments(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewDeploymentServiceClient(conn)

	qb := search.NewQueryBuilder().AddStrings(search.DeploymentName, "central").AddStrings(search.DeploymentName, "sensor")
	deployments, err := service.ListDeployments(ctx, &v1.RawQuery{
		Query: qb.Query(),
	})
	require.NoError(t, err)
	require.Len(t, deployments.GetDeployments(), 2)

	var centralDeployment, sensorDeployment *v1.Deployment

	for _, d := range deployments.GetDeployments() {
		if d.GetName() == `central` {
			centralDeployment, err = retrieveDeployment(service, d)
			require.NoError(t, err)
		} else if d.GetName() == `sensor` {
			sensorDeployment, err = retrieveDeployment(service, d)
			require.NoError(t, err)
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

	require.True(t, len(centralDeployment.GetContainers()) >= 1)
	c := centralDeployment.GetContainers()[0]

	assert.Equal(t, imageName, c.GetImage().GetName().GetRemote())
	if sha, ok := os.LookupEnv(imageTagEnv); ok {
		assert.Equal(t, sha, c.GetImage().GetName().GetTag())
	}

	require.True(t, len(c.GetSecrets()) >= 3)
	var paths []string
	for _, secret := range c.GetSecrets() {
		if !strings.Contains(secret.GetPath(), "monitoring") {
			paths = append(paths, secret.GetPath())
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})

	expectedPathPrefixes := []string{
		"/run/secrets/stackrox.io/certs",
		"/run/secrets/stackrox.io/htpasswd/",
		"/run/secrets/stackrox.io/jwt",
		"/usr/local/share/ca-certificates/",
	}
	for i, path := range paths {
		assert.True(t, strings.HasPrefix(path, expectedPathPrefixes[i]))
	}

	require.Len(t, c.GetPorts(), 1)
	p := c.GetPorts()[0]
	assert.Equal(t, int32(443), p.GetContainerPort())
	assert.Equal(t, "TCP", p.GetProtocol())
}

func verifySensorDeployment(t *testing.T, sensorDeployment *v1.Deployment) {
	verifyDeployment(t, sensorDeployment)
	assert.Equal(t, "sensor", sensorDeployment.GetLabels()["app"])

	require.True(t, len(sensorDeployment.GetContainers()) >= 1)
	c := sensorDeployment.GetContainers()[0]

	assert.Equal(t, imageName, c.GetImage().GetName().GetRemote())
	if sha, ok := os.LookupEnv(imageTagEnv); ok {
		assert.Equal(t, sha, c.GetImage().GetName().GetTag())
	}

	require.True(t, len(c.GetSecrets()) >= 1)
	s := c.GetSecrets()[0]
	if strings.Contains(s.GetPath(), "monitoring") {
		s = c.GetSecrets()[1]
	}
	assert.True(t, strings.HasPrefix(s.GetPath(), "/run/secrets/stackrox.io/"))
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

	conn, err := grpcConnection()
	require.NoError(t, err)

	service := v1.NewImageServiceClient(conn)

	images, err := service.ListImages(ctx, &v1.RawQuery{})
	require.NoError(t, err)

	require.NotEmpty(t, images.GetImages())

	imageMap := make(map[string][]*v1.Image)
	for _, img := range images.GetImages() {
		image, err := service.GetImage(ctx, &v1.ResourceByID{Id: img.GetId()})
		assert.NoError(t, err)
		imageMap[image.GetName().GetRegistry()] = append(imageMap[image.GetName().GetRegistry()], image)
	}

	const dockerRegistry = `docker.io`

	require.NotEmpty(t, imageMap[dockerRegistry])

	foundMainImage := false

	for _, img := range imageMap[dockerRegistry] {
		if img.GetName().GetRemote() == imageName {
			foundMainImage = true

			if sha, ok := os.LookupEnv(imageTagEnv); ok {
				assert.Equal(t, sha, img.GetName().GetTag())
			}
		}
	}

	assert.True(t, foundMainImage)
}
