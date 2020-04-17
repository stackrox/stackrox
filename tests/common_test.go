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

func assumeFeatureFlagHasValue(t *testing.T, featureFlag features.FeatureFlag, assumedValue bool) {
	conn := testutils.GRPCConnectionToCentral(t)
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

func retrieveDeployment(service v1.DeploymentServiceClient, listDeployment *storage.ListDeployment) (*storage.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.GetDeployment(ctx, &v1.ResourceByID{Id: listDeployment.GetId()})
}

func retrieveDeployments(service v1.DeploymentServiceClient, deps []*storage.ListDeployment) ([]*storage.Deployment, error) {
	deployments := make([]*storage.Deployment, 0, len(deps))
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
	conn := testutils.GRPCConnectionToCentral(t)

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

			if err == nil && len(deployments) > 0 {
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

func waitForTermination(t *testing.T, deploymentName string) {
	conn := testutils.GRPCConnectionToCentral(t)

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

// The deploymentName must be copied form the file path passed in
func setupDeploymentFromFile(t *testing.T, deploymentName, path string) {
	cmd := exec.Command(`kubectl`, `create`, `-f`, path)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, deploymentName)
}

func setupNginxLatestTagDeployment(t *testing.T) {
	setupDeployment(t, "nginx", nginxDeploymentName)
}

func setupDeployment(t *testing.T, image, deploymentName string) {
	cmd := exec.Command(`kubectl`, `run`, deploymentName, fmt.Sprintf("--image=%s", image))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, deploymentName)
}

func teardownDeploymentFromFile(t *testing.T, deploymentName, path string) {
	cmd := exec.Command(`kubectl`, `delete`, `-f`, path, `--ignore-not-found=true`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

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
