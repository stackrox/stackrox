package tests

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	nginxDeploymentName     = `nginx`
	expectedLatestTagPolicy = `Latest tag`

	waitTimeout = 2 * time.Minute
)

var (
	log = logging.LoggerForModule()
)

//lint:ignore U1000 Ignore unused code check since this function could be useful in future.
func assumeFeatureFlagHasValue(t *testing.T, featureFlag features.FeatureFlag, assumedValue bool) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	featureService := v1.NewFeatureFlagServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	featureFlagsResp, err := featureService.GetFeatureFlags(ctx, &v1.Empty{})
	require.NoError(t, err, "failed to get feature flags from central")

	for _, flag := range featureFlagsResp.GetFeatureFlags() {
		if flag.GetEnvVar() == featureFlag.EnvVar() {
			if flag.GetEnabled() == assumedValue {
				return
			}
			t.Skipf("skipping test because value of feature flag %s is not %t", featureFlag.EnvVar(), assumedValue)
		}
	}

	t.Fatalf("Central has no knowledge about feature flag %s", featureFlag.EnvVar())
}

func retrieveDeployment(service v1.DeploymentServiceClient, deploymentID string) (*storage.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.GetDeployment(ctx, &v1.ResourceByID{Id: deploymentID})
}

func retrieveDeployments(service v1.DeploymentServiceClient, deps []*storage.ListDeployment) ([]*storage.Deployment, error) {
	deployments := make([]*storage.Deployment, 0, len(deps))
	for _, d := range deps {
		deployment, err := retrieveDeployment(service, d.GetId())
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

func waitForDeploymentCount(t testutils.T, query string, count int) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			deploymentCount, err := service.CountDeployments(ctx, &v1.RawQuery{Query: query})
			cancel()
			if err != nil {
				log.Errorf("Error listing deployments: %s", err)
				continue
			}
			if deploymentCount.GetCount() == int32(count) {
				return
			}

		case <-timer.C:
			t.Fatalf("Timed out waiting for deployments %q", query)
		}
	}

}

func waitForDeployment(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

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
				log.Errorf("Error listing deployments: %s", err)
				continue
			}

			deployments, err := retrieveDeployments(service, listDeployments.GetDeployments())
			if err != nil {
				log.Errorf("Error retrieving deployments: %s", err)
				continue
			}

			if len(deployments) > 0 {
				d := deployments[0]

				if len(d.GetContainers()) > 0 && d.GetContainers()[0].GetImage().GetId() != "" {
					return
				}
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for deployment %s", deploymentName)
		}
	}
}

func waitForTermination(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

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
				log.Error(err)
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

func applyFile(t testutils.T, path string) {
	cmd := exec.Command(`kubectl`, `create`, `-f`, path)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// The deploymentName must be copied form the file path passed in
func setupDeploymentFromFile(t testutils.T, deploymentName, path string) {
	applyFile(t, path)
	waitForDeployment(t, deploymentName)
}

func setupNginxLatestTagDeployment(t *testing.T) {
	setupDeployment(t, "nginx", nginxDeploymentName)
}

func setupDeployment(t *testing.T, image, deploymentName string) {
	setupDeploymentWithReplicas(t, image, deploymentName, 1)
}

func setupDeploymentWithReplicas(t *testing.T, image, deploymentName string, replicas int) {
	cmd := exec.Command(`kubectl`, `create`, `deployment`, deploymentName, fmt.Sprintf("--image=%s", image), fmt.Sprintf("--replicas=%d", replicas))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, deploymentName)
}

func setImage(t *testing.T, deploymentName string, deploymentID string, containerName string, image string) {
	cmd := exec.Command(`kubectl`, `set`, `image`, fmt.Sprintf("deployment/%s", deploymentName), fmt.Sprintf("%s=%s", containerName, image))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDeploymentServiceClient(conn)

	waitForCondition(t, func() bool {
		deployment, err := retrieveDeployment(service, deploymentID)
		if err != nil {
			log.Error(err)
			return false
		}
		containers := deployment.GetContainers()
		for _, container := range containers {
			if container.GetImage().GetName().GetFullName() != image {
				return false
			}
		}
		log.Infof("Image set to %s for deployment %s(%s) container %s", image, deploymentName, deploymentID, containerName)
		return true
	}, "image updated", time.Minute, 5*time.Second)
}

func teardownFile(t testutils.T, path string) {
	cmd := exec.Command(`kubectl`, `delete`, `-f`, path, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func teardownDeploymentFromFile(t testutils.T, deploymentName, path string) {
	teardownFile(t, path)
	waitForTermination(t, deploymentName)
}

func teardownDeployment(t *testing.T, deploymentName string) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, deploymentName, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForTermination(t, deploymentName)
}

func teardownNginxLatestTagDeployment(t *testing.T) {
	teardownDeployment(t, nginxDeploymentName)
}

func createK8sClient(t *testing.T) kubernetes.Interface {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(t, err, "could not load default Kubernetes client config")

	restCfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	require.NoError(t, err, "could not get REST client config from kubernetes config")

	k8sClient, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err, "creating Kubernetes client from REST config")

	return k8sClient
}

func waitForCondition(t testutils.T, condition func() bool, desc string, timeout time.Duration, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for condition %s to stop", desc)
		}
	}
}
