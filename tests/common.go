//go:build test_e2e || sql_integration || compliance || destructive || externalbackups || test_compatibility

package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//lint:file-ignore U1000 since common.go is included in several different go:build tags but not every function is used
// within every tag, hence the linter incorrectly shows most functions in this file as unused.

const (
	nginxDeploymentName     = `nginx`
	expectedLatestTagPolicy = `Latest tag`

	sensorDeployment = "sensor"
	sensorContainer  = "sensor"

	waitTimeout = 5 * time.Minute
	timeout     = 30 * time.Second
)

var (
	log = logging.LoggerForModule()

	sensorPodLabels = map[string]string{"app": "sensor"}
)

// logf logs using the testing logger, prefixing a high-resolution timestamp.
// Using testing.T.Logf means that the output is hidden unless the test fails or verbose logging is enabled with -v.
func logf(t *testing.T, format string, args ...any) {
	t.Logf(time.Now().Format(time.StampMilli)+" "+format, args...)
}

// testContexts returns a couple of contexts for the given test: with a timeout for the testing logic and another
// with an additional longer timeout for debug artifact gathering and cleanup.
func testContexts(t *testing.T, name string, timeout time.Duration) (testCtx context.Context, overallCtx context.Context, cancel func()) {
	var (
		overallCancel func()
		testCancel    func()
	)
	cleanupTimeout := 10 * time.Minute
	logf(t, "Running %s with a timeout of %s plus %s for cleanup", name, timeout, cleanupTimeout)
	overallTimeout := timeout + cleanupTimeout
	overallErr := fmt.Errorf("overall %s test+cleanup timeout of %s reached", name, overallTimeout)
	testErr := fmt.Errorf("%s test timeout of %s reached", name, timeout)
	overallCtx, overallCancel = context.WithTimeoutCause(context.Background(), overallTimeout, overallErr)
	testCtx, testCancel = context.WithTimeoutCause(overallCtx, timeout, testErr)
	cancel = func() {
		testCancel()
		overallCancel()
	}
	return
}

// mustGetEnv calls os.GetEnv and fails the test if result is empty.
func mustGetEnv(t *testing.T, varName string) string {
	val := os.Getenv(varName)
	require.NotEmptyf(t, val, "Environment variable %q must be set.", varName)
	return val
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

	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, deploymentName)

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

func getPodFromFile(t testutils.T, path string) *coreV1.Pod {
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	var pod coreV1.Pod
	err = decoder.Decode(&pod)
	require.NoError(t, err)
	return &pod
}

func setupDeployment(t *testing.T, image, deploymentName string) {
	setupDeploymentWithReplicas(t, image, deploymentName, 1)
}

func setupDeploymentWithReplicas(t *testing.T, image, deploymentName string, replicas int) {
	setupDeploymentNoWait(t, image, deploymentName, replicas)
	waitForDeployment(t, deploymentName)
}

func setupDeploymentNoWait(t *testing.T, image, deploymentName string, replicas int) {
	cmd := exec.Command(`kubectl`, `create`, `deployment`, deploymentName, fmt.Sprintf("--image=%s", image), fmt.Sprintf("--replicas=%d", replicas))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	if os.Getenv("REMOTE_CLUSTER_ARCH") == "arm64" {
		cmd = exec.Command(`kubectl`, `patch`, `deployment`, deploymentName, `-p`, `{"spec":{"template":{"spec":{"nodeSelector":{"kubernetes.io/arch":"arm64"}}}}}`)
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))
	}
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

func createPod(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Infof("Creating pod %s %s", pod.GetNamespace(), pod.GetName())
	_, err := client.CoreV1().Pods(pod.GetNamespace()).Create(ctx, pod, metaV1.CreateOptions{})
	require.NoError(t, err)

	waitForDeployment(t, pod.GetName())
}

func teardownPod(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{GracePeriodSeconds: pointers.Int64(0)})
	require.NoError(t, err)

	waitForTermination(t, pod.GetName())
}

func teardownDeployment(t *testing.T, deploymentName string) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, deploymentName, `--ignore-not-found=true`, `--grace-period=1`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	waitForTermination(t, deploymentName)
}

func teardownDeploymentWithoutCheck(t *testing.T, deploymentName string) {
	// In cases where deployment will not impact other tests,
	// we can trigger deletion and assume that it will be deleted eventually.
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, deploymentName, `--ignore-not-found=true`, `--grace-period=1`)
	if err := cmd.Run(); err != nil {
		logf(t, "Deleting deployment %q failed: %v", deploymentName, err)
	}
}

func getConfig(t *testing.T) *rest.Config {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(t, err, "could not load default Kubernetes client config")

	restCfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	require.NoError(t, err, "could not get REST client config from kubernetes config")

	return restCfg
}

func createK8sClient(t *testing.T) kubernetes.Interface {
	restCfg := getConfig(t)
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

type KubernetesSuite struct {
	suite.Suite
	k8s kubernetes.Interface
}

func (ks *KubernetesSuite) SetupSuite() {
	ks.k8s = createK8sClient(ks.T())
}

func (ks *KubernetesSuite) logf(format string, args ...any) {
	logf(ks.T(), format, args...)
}

type logMatcher interface {
	Match(reader io.ReadSeeker) (bool, error)
	fmt.Stringer
}

// waitUntilLog waits until ctx expires or logs of container in all pods matching podLabels satisfy all logMatchers.
func (ks *KubernetesSuite) waitUntilLog(ctx context.Context, namespace string, podLabels map[string]string, container string, description string, logMatchers ...logMatcher) {
	ls := labels.SelectorFromSet(podLabels).String()
	checkLogs := ks.checkLogsClosure(ctx, namespace, ls, container, logMatchers...)
	ks.logf("Waiting until %q pods logs %s: %s", ls, description, logMatchers)
	mustEventually(ks.T(), ctx, checkLogs, 10*time.Second, fmt.Sprintf("Not all %q pods logs %s", ls, description))
}

// checkLogsMatch will check if the logs of container in all pods matching pod labels satisfy all log matchers.
func (ks *KubernetesSuite) checkLogsMatch(ctx context.Context, namespace string, podLabels map[string]string, container string, description string, logMatchers ...logMatcher) {
	ls := labels.SelectorFromSet(podLabels).String()
	ks.logf("Checking %q pods logs %s: %s", ls, description, logMatchers)
	err := ks.checkLogsClosure(ctx, namespace, ls, container, logMatchers...)()
	require.NoError(ks.T(), err, fmt.Sprintf("%q was untrue", description))
}

// checkLogsClosure returns a function that checks if the logs of container in all pods matching pod labels satisfy all the log matchers.
func (ks *KubernetesSuite) checkLogsClosure(ctx context.Context, namespace, labelSelector, container string, logMatchers ...logMatcher) func() error {
	return func() error {
		podList, err := ks.k8s.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return fmt.Errorf("could not list pods matching %q in namespace %q: %w", labelSelector, namespace, err)
		}
		if len(podList.Items) == 0 {
			if ok, err := allMatch(strings.NewReader(""), logMatchers...); ok {
				return nil
			} else if err != nil {
				return fmt.Errorf("empty list of pods caused failure: %w", err)
			}
			return fmt.Errorf("empty list of pods does not satisfy the condition")
		}
		for _, pod := range podList.Items {
			resp := ks.k8s.CoreV1().Pods(namespace).GetLogs(pod.GetName(), &coreV1.PodLogOptions{Container: container}).Do(ctx)
			log, err := resp.Raw()
			if err != nil {
				return fmt.Errorf("retrieving logs of pod %q in namespace %q failed: %w", pod.GetName(), namespace, err)
			}
			if ok, err := allMatch(bytes.NewReader(log), logMatchers...); ok {
				continue
			} else if err != nil {
				return fmt.Errorf("log of pod %q in namespace %q caused failure: %w", pod.GetName(), namespace, err)
			}
			return fmt.Errorf("log of pod %q in namespace %q does not satisfy the condition", pod.GetName(), namespace)
		}
		return nil
	}
}

// getLastLogBytePos will return the position of the last log byte for a particular pod.  This is intended to be used with
// the log matchers to ensure a log from previous tests isn't matched against.
func (ks *KubernetesSuite) getLastLogBytePos(ctx context.Context, namespace, pod, container string) (int64, error) {
	resp := ks.k8s.CoreV1().Pods(namespace).GetLogs(pod, &coreV1.PodLogOptions{Container: container}).Do(ctx)

	logB, err := resp.Raw()
	if err != nil {
		return 0, fmt.Errorf("retrieving logs of pod %q in namespace %q failed: %w", pod, namespace, err)
	}

	return int64(len(logB)), nil
}

// getSensorPod is a convenience method to get details of the Sensor pod.
func (ks *KubernetesSuite) getSensorPod(ctx context.Context, namespace string) (*coreV1.Pod, error) {
	ls := labels.SelectorFromSet(sensorPodLabels).String()
	podList, err := ks.k8s.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: ls})
	if err != nil {
		return nil, fmt.Errorf("could not list pods matching %q in namespace %q: %w", sensorPodLabels, namespace, err)
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("empty list of pods does not satisfy the condition")
	}
	if len(podList.Items) > 1 {
		return nil, fmt.Errorf("more than one sensor pod running")
	}

	return &podList.Items[0], nil
}

// waitUntilK8sDeploymentReady waits until k8s reports all pods for a deployment are ready or context done.
func (ks *KubernetesSuite) waitUntilK8sDeploymentReady(ctx context.Context, namespace string, deploymentName string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)

	for {
		select {
		case <-ctx.Done():
			require.NoError(ks.T(), ctx.Err())
		case <-ticker.C:
			deploy, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			require.NoError(ks.T(), err, "getting deployment %q from namespace %q", deploymentName, namespace)

			if deploy.GetGeneration() != deploy.Status.ObservedGeneration {
				ks.logf("deployment %q in namespace %q NOT ready, generation %d, observed generation", deploymentName, namespace, deploy.GetGeneration(), deploy.Status.ObservedGeneration)
				continue
			}

			if deploy.Status.Replicas == 0 || deploy.Status.Replicas != deploy.Status.ReadyReplicas {
				ks.logf("deployment %q in namespace %q NOT ready (%d/%d ready replicas)", deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
				continue
			}
			ks.logf("deployment %q in namespace %q READY (%d/%d ready replicas)", deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
			return
		case <-timer.C:
			ks.T().Fatalf("Timed out waiting for deployment %s", deploymentName)
		}
	}
}

// mustEventually retries every pauseInterval until ctx expires or f succeeds.
func mustEventually(t *testing.T, ctx context.Context, f func() error, pauseInterval time.Duration, failureMsgPrefix string) {
	require.NoError(t, retry.WithRetry(f,
		retry.Tries(math.MaxInt),
		retry.WithContext(ctx),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(pauseInterval)
		}),
		retry.OnFailedAttempts(func(err error) { logf(t, failureMsgPrefix+": %s", err) })))
}

// createService creates a k8s Service object.
func (ks *KubernetesSuite) createService(ctx context.Context, namespace string, name string, labels map[string]string, ports map[int32]int32) {
	svc := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: coreV1.ServiceSpec{
			Selector: labels,
			Type:     coreV1.ServiceTypeClusterIP,
		},
	}
	for portNum, targetPortNum := range ports {
		svc.Spec.Ports = append(svc.Spec.Ports, coreV1.ServicePort{
			Name:       fmt.Sprintf("%d", portNum),
			Protocol:   coreV1.ProtocolTCP,
			Port:       portNum,
			TargetPort: intstr.IntOrString{IntVal: targetPortNum},
		})
	}
	_, err := ks.k8s.CoreV1().Services(namespace).Create(ctx, svc, metaV1.CreateOptions{})
	ks.Require().NoError(err, "cannot create service %q in namespace %q", name, namespace)
}

// ensureSecretExists creates a k8s Secret object. If one exists, it makes sure the type and data matches.
func (ks *KubernetesSuite) ensureSecretExists(ctx context.Context, namespace string, name string, secretType coreV1.SecretType, data map[string][]byte) {
	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
		},
		Type: secretType,
		Data: data,
	}
	if _, err := ks.k8s.CoreV1().Secrets(namespace).Create(ctx, secret, metaV1.CreateOptions{}); err != nil {
		if apiErrors.IsAlreadyExists(err) {
			actualSecret, err := ks.k8s.CoreV1().Secrets(namespace).Get(ctx, name, metaV1.GetOptions{})
			ks.Require().NoError(err, "secret %q in namespace %q already exists but cannot retrieve it for verification", name, namespace)
			ks.Equal(secretType, actualSecret.Type, "secret %q in namespace %q already exists but its type is not as expected", name, namespace)
			ks.Equal(data, actualSecret.Data, "secret %q in namespace %q already exists but its data is not as expected", name, namespace)
			ks.Require().False(ks.T().Failed(), "secrets do not match")
		}
		ks.Require().NoError(err, "cannot create secret %q in namespace %q", name, namespace)
	}
}

// ensureConfigMapExists creates a k8s ConfigMap object. If one exists, it makes sure the data matches.
func (ks *KubernetesSuite) ensureConfigMapExists(ctx context.Context, namespace string, name string, data map[string]string) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
	if _, err := ks.k8s.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metaV1.CreateOptions{}); err != nil {
		if apiErrors.IsAlreadyExists(err) {
			actualCM, err := ks.k8s.CoreV1().ConfigMaps(namespace).Get(ctx, name, metaV1.GetOptions{})
			ks.Require().NoError(err, "configMap %q in namespace %q already exists but cannot retrieve it for verification", name, namespace)
			ks.Require().Equal(data, actualCM.Data, "configMap %q in namespace %q already exists but its data is not as expected", name, namespace)
		}
		ks.Require().NoError(err, "cannot create configMap %q in namespace %q", name, namespace)
	}
}

// waitUntilCentralSensorConnectionIs makes sure there is only one cluster defined on central, and its status is eventually
// one of the specified statuses.
func waitUntilCentralSensorConnectionIs(t *testing.T, ctx context.Context, statuses ...storage.ClusterHealthStatus_HealthStatusLabel) {
	logf(t, "Waiting until central-sensor connection is in state(s) %s...", statuses)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	checkCentralSensorConnection := func() error {
		cluster, err := getCluster(ctx, conn)
		if err != nil {
			return err
		}

		clusterStatus := cluster.GetHealthStatus().GetSensorHealthStatus()
		for _, status := range statuses {
			if clusterStatus == status {
				logf(t, "Central-sensor connection now in state %q.", clusterStatus)
				return nil
			}
		}
		return fmt.Errorf("encountered cluster status %q", clusterStatus.String())
	}
	// TODO: Sensor reports status every 30 seconds, consider sleeping here to ensure we have the most up to date status.
	mustEventually(t, ctx, checkCentralSensorConnection, 10*time.Second, "Central-sensor connection not in desired state (yet)")
}

// mustSetDeploymentEnvVal sets the specified env variable on a container in a deployment using strategic merge patch, or fails the test.
func (ks *KubernetesSuite) mustSetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string, value string) {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":%q,"env":[{"name":%q,"value":%q}]}]}}}}`,
		container, envVar, value))
	ks.logf("Setting variable %q on deployment %q in namespace %q to %q", envVar, deployment, namespace, value)
	_, err := ks.k8s.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	ks.Require().NoError(err, "cannot patch deployment %q in namespace %q", deployment, namespace)
}

// mustGetDeploymentEnvVal retrieves the value of environment variable in a deployment, or fails the test.
func (ks *KubernetesSuite) mustGetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string) string {
	val, err := ks.getDeploymentEnvVal(ctx, namespace, deployment, container, envVar)
	ks.Require().NoError(err, "cannot find envVar %q in container %q in deployment %q in namespace %q", envVar, container, deployment, namespace)
	return val
}

// getDeploymentEnvVal returns the value of an environment variable or the empty string if not found.
func (ks *KubernetesSuite) getDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string) (string, error) {
	d, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deployment, metaV1.GetOptions{})
	ks.Require().NoError(err, "cannot retrieve deployment %q in namespace %q", deployment, namespace)
	c, err := getContainer(d, container)
	ks.Require().NoError(err, "cannot find container %q in deployment %q in namespace %q", container, deployment, namespace)
	return getEnvVal(c, envVar)
}

// mustDeleteDeploymentEnvVar deletes an env var from all containers of a deployment, if any errors
// are encountered the suite will fail. This is a no-op if the env var is not found.
func (ks *KubernetesSuite) mustDeleteDeploymentEnvVar(ctx context.Context, namespace, deployment, envVar string) {
	d, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deployment, metaV1.GetOptions{})
	ks.Require().NoError(err, "cannot retrieve deployment %q in namespace %q", deployment, namespace)

	sb := strings.Builder{}
	for i, c := range d.Spec.Template.Spec.Containers {
		for j, e := range c.Env {
			if e.Name == envVar {
				if sb.Len() > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(fmt.Sprintf(`{"op":"remove", "path":"/spec/template/spec/containers/%d/env/%d"}`, i, j))
				break
			}
		}
	}

	patch := fmt.Sprintf("[%s]", sb.String())
	_, err = ks.k8s.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.JSONPatchType, []byte(patch), metaV1.PatchOptions{})
	ks.Require().NoError(err, "failed to remove env var %q from deployment %q namespace %q", envVar, deployment, namespace)
	ks.logf("Removed env variable %q from deployment %q namespace %q", envVar, deployment, namespace)
}

// listK8sAPIResources will return a list of all custom resources the k8s (or OCP) API supports.
func (ks *KubernetesSuite) listK8sAPIResources() []*metaV1.APIResourceList {
	t := ks.T()

	apiResourceList, err := ks.k8s.Discovery().ServerPreferredResources()
	require.NoError(t, err)

	return apiResourceList
}

// mustCreateAPIToken will create an API token. Returns the token ID and value.
func mustCreateAPIToken(t *testing.T, ctx context.Context, name string, roles []string) (string, string) {
	require := require.New(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewAPITokenServiceClient(conn)

	tokResp, err := service.GenerateToken(ctx, &v1.GenerateTokenRequest{
		Name:  name,
		Roles: roles,
	})
	require.NoError(err)

	logf(t, "Created API Token %q (%v): %v", name, tokResp.GetMetadata().GetId(), roles)
	return tokResp.GetMetadata().GetId(), tokResp.GetToken()
}

// revokeAPIToken will revoke an API token.
func revokeAPIToken(t *testing.T, ctx context.Context, idOrName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewAPITokenServiceClient(conn)

	toksResp, err := service.GetAPITokens(ctx, &v1.GetAPITokensRequest{
		RevokedOneof: &v1.GetAPITokensRequest_Revoked{Revoked: false},
	})
	require.NoError(t, err)

	for _, tok := range toksResp.GetTokens() {
		if tok.GetId() == idOrName || tok.GetName() == idOrName {
			_, err = service.RevokeToken(ctx, &v1.ResourceByID{Id: tok.GetId()})
			require.NoError(t, err)
			logf(t, "Revoked API token %q (%s)", tok.GetName(), tok.GetId())
			return
		}
	}
}

// mustCreatePermissionSet will create a permission set and return the associated ID.
func mustCreatePermissionSet(t *testing.T, ctx context.Context, permissionSet *storage.PermissionSet) string {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	ps, err := service.PostPermissionSet(ctx, permissionSet)
	require.NoError(t, err)

	logf(t, "Created permission set: %v", ps)
	return ps.GetId()
}

// deletePermissionSet will delete the specified permission set.
func deletePermissionSet(t *testing.T, ctx context.Context, idOrName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	resp, err := service.ListPermissionSets(ctx, &v1.Empty{})
	require.NoError(t, err)

	for _, ps := range resp.GetPermissionSets() {
		if ps.GetId() == idOrName || ps.GetName() == idOrName {
			_, err = service.DeletePermissionSet(ctx, &v1.ResourceByID{Id: ps.GetId()})
			require.NoError(t, err)
			logf(t, "Deleted permission set %q (%s)", ps.GetName(), ps.GetId())
			return
		}
	}
}

// mustCreateRole will create a role.
func mustCreateRole(t *testing.T, ctx context.Context, role *storage.Role) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	_, err := service.CreateRole(ctx, &v1.CreateRoleRequest{
		Name: role.GetName(),
		Role: role,
	})
	require.NoError(t, err)

	logf(t, "Created role: %v", role)
}

// deleteRole will delete the specified role.
func deleteRole(t *testing.T, ctx context.Context, name string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	resp, err := service.GetRoles(ctx, &v1.Empty{})
	require.NoError(t, err)

	for _, role := range resp.GetRoles() {
		if role.GetName() == name {
			_, err = service.DeleteRole(ctx, &v1.ResourceByID{Id: name})
			require.NoError(t, err)
			logf(t, "Deleted role %q", name)
			return
		}
	}
}

// getEnvVal returns the value of envVar from a given container or returns a helpful error.
func getEnvVal(c *coreV1.Container, envVar string) (string, error) {
	var vars []string
	for _, v := range c.Env {
		if v.Name == envVar {
			return v.Value, nil
		}
		vars = append(vars, v.Name)
	}
	return "", fmt.Errorf("actual vars are %q", vars)
}

// getContainer returns the given container from a deployment or returns a helpful error.
func getContainer(deployment *appsV1.Deployment, container string) (*coreV1.Container, error) {
	var containers []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == container {
			return &c, nil
		}
		containers = append(containers, c.Name)
	}
	return nil, fmt.Errorf("actual containers are %q", containers)
}

// collectLogs collects service logs from a given namespace into a subdirectory that will be gathered by the CI scripts.
func collectLogs(t *testing.T, ns string, dir string) {
	subDir := "e2e-test-logs/" + dir
	start := time.Now()
	err := exec.Command("../scripts/ci/collect-service-logs.sh", ns, "/tmp/"+subDir).Run()
	if err != nil {
		logf(t, "Collecting %q logs returned error %s", ns, err)
	}
	logf(t, "Collected logs from namespace %q to directory %q, took: %s", ns, subDir, time.Since(start))
}

// isOpenshift returns true when the test env is a flavor of OCP, false otherwise.
func isOpenshift() bool {
	return os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift"
}

// mustGetCluster returns the details of the one and only known secured cluster.
func mustGetCluster(t *testing.T, ctx context.Context) *storage.Cluster {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	cluster, err := getCluster(ctx, conn)
	require.NoError(t, err)

	return cluster
}

// getCluster returns the details of the one and only known secured cluster.
func getCluster(ctx context.Context, conn *grpc.ClientConn) (*storage.Cluster, error) {
	clustersSvc := v1.NewClustersServiceClient(conn)

	clusters, err := clustersSvc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve clusters from central: %w", err)
	}

	if len(clusters.Clusters) != 1 {
		var clusterNames []string
		for _, cluster := range clusters.Clusters {
			clusterNames = append(clusterNames, cluster.Name)
		}
		return nil, fmt.Errorf("expected one cluster, found %d: %+v", len(clusters.Clusters), clusterNames)
	}

	return clusters.Clusters[0], nil
}
