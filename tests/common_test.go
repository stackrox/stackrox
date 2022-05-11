package tests

import (
	"context"
	"fmt"
	"os"
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
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
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
	log                     = logging.LoggerForModule()
	clientSet               = newClientSet()
	defaultDeploymentClient = clientSet.AppsV1().Deployments(apiv1.NamespaceDefault)
)

// newClientSet creates a new kubernetes.Clientset object from the default configuration
func newClientSet() *kubernetes.Clientset {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		log.Errorf("Failed to get k8s client config %v", err)
		return nil
	}

	result, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Failed to create k8s client from config %v", err)
		return nil
	}
	return result
}

// getDeploymentFromFile returns a decoded deployment object given a path to a deployment yaml file
func getDeploymentFromFile(t testutils.T, path string) *appsv1.Deployment {
	file, err := os.Open(path)
	require.NoError(t, err, fmt.Sprintf("Failed to open deployment yaml (%s)", path))
	result := &appsv1.Deployment{}

	// bufferSize in this function defines how far into the stream to look for an open brace, but we only expect yaml
	err = yaml.NewYAMLOrJSONDecoder(file, 0).Decode(result)
	require.NoError(t, err, fmt.Sprintf("failed to decode yaml file (%s)", path))

	return result
}

// deleteDeployment deletes the provided deployment
func deleteDeployment(t *testing.T, deployment *appsv1.Deployment) {
	deletePolicy := metav1.DeletePropagationForeground
	err := defaultDeploymentClient.Delete(context.Background(), deployment.GetName(), metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	require.NoError(t, err, fmt.Sprintf("Failed to tear down deployment (%s)", deployment.GetName()))

	waitForTermination(t, deployment.GetName())
}

// createDeployment creates a deployment from a yaml file at the provided path
func createDeployment(t *testing.T, path string) *appsv1.Deployment {
	deployment := getDeploymentFromFile(t, path)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	result, err := defaultDeploymentClient.Create(ctx, deployment, metav1.CreateOptions{})
	cancel()
	require.NoError(t, err, fmt.Sprintf("Failed to create deployment (%s)", deployment.GetName()))

	waitForDeployment(t, deployment.GetName())

	return result
}

//lint:ignore U1000 Ignore unused code check since this function could be useful in future.
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

func waitForDeployment(t testutils.T, deploymentName string) {
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

func applyFile(t testutils.T, path string) {
	cmd := exec.Command(`kubectl`, `create`, `-f`, path)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// The deploymentName must be copied form the file path passed in TODO
func setupDeploymentFromFile(t testutils.T, deploymentName, path string) {
	//applyFile(t, path)
	//waitForDeployment(t, deploymentName)
	deployment := getDeploymentFromFile(t, path)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	_, err := defaultDeploymentClient.Create(ctx, deployment, metav1.CreateOptions{})
	cancel()
	require.NoError(t, err, fmt.Sprintf("Failed to create deployment (%s)", deployment.GetName()))

	waitForDeployment(t, deployment.GetName())
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
	conn := testutils.GRPCConnectionToCentral(t)
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

func scaleDeployment(t testutils.T, deploymentName, replicas string) {
	cmd := exec.Command(`kubectl`, `scale`, `deployment`, deploymentName, `--replicas`, replicas)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForDeployment(t, deploymentName)
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
