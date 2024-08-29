//go:build test_e2e || sql_integration || compliance || destructive || externalbackups || test_compatibility

package tests

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
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
	NginxDeploymentName     = `nginx`
	ExpectedLatestTagPolicy = `Latest tag`

	WaitTimeout = 5 * time.Minute
	Timeout     = 30 * time.Second
)

var (
	Log = logging.LoggerForModule()
)

// Logf logs using the testing logger, prefixing a high-resolution timestamp.
// Using testing.T.Logf means that the output is hidden unless the test fails or verbose logging is enabled with -v.
func Logf(t *testing.T, format string, args ...any) {
	t.Logf(time.Now().Format(time.StampMilli)+" "+format, args...)
}

// TestContexts returns a couple of contexts for the given test: with a Timeout for the testing logic and another
// with an additional longer Timeout for debug artifact gathering and cleanup.
func TestContexts(t *testing.T, name string, timeout time.Duration) (testCtx context.Context, overallCtx context.Context, cancel func()) {
	var (
		overallCancel func()
		testCancel    func()
	)
	cleanupTimeout := 10 * time.Minute
	t.Logf("Running %s with a Timeout of %s plus %s for cleanup", name, timeout, cleanupTimeout)
	overallTimeout := timeout + cleanupTimeout
	overallErr := fmt.Errorf("overall %s test+cleanup Timeout of %s reached", name, overallTimeout)
	testErr := fmt.Errorf("%s test Timeout of %s reached", name, timeout)
	overallCtx, overallCancel = context.WithTimeoutCause(context.Background(), overallTimeout, overallErr)
	testCtx, testCancel = context.WithTimeoutCause(overallCtx, timeout, testErr)
	cancel = func() {
		testCancel()
		overallCancel()
	}
	return
}

// MustGetEnv calls os.GetEnv and fails the test if result is empty.
func MustGetEnv(t *testing.T, varName string) string {
	val := os.Getenv(varName)
	require.NotEmptyf(t, val, "Environment variable %q must be set.", varName)
	return val
}

func RetrieveDeployment(service v1.DeploymentServiceClient, deploymentID string) (*storage.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.GetDeployment(ctx, &v1.ResourceByID{Id: deploymentID})
}

func RetrieveDeployments(service v1.DeploymentServiceClient, deps []*storage.ListDeployment) ([]*storage.Deployment, error) {
	deployments := make([]*storage.Deployment, 0, len(deps))
	for _, d := range deps {
		deployment, err := RetrieveDeployment(service, d.GetId())
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

func WaitForDeploymentCount(t testutils.T, query string, count int) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(WaitTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			deploymentCount, err := service.CountDeployments(ctx, &v1.RawQuery{Query: query})
			cancel()
			if err != nil {
				Log.Errorf("Error listing deployments: %s", err)
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

func WaitForDeployment(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(WaitTimeout)
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
				Log.Errorf("Error listing deployments: %s", err)
				continue
			}

			deployments, err := RetrieveDeployments(service, listDeployments.GetDeployments())
			if err != nil {
				Log.Errorf("Error retrieving deployments: %s", err)
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

func WaitForTermination(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(WaitTimeout)
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
				Log.Error(err)
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

func GetPodFromFile(t testutils.T, path string) *coreV1.Pod {
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	var pod coreV1.Pod
	err = decoder.Decode(&pod)
	require.NoError(t, err)
	return &pod
}

func SetupDeployment(t *testing.T, image, deploymentName string) {
	SetupDeploymentWithReplicas(t, image, deploymentName, 1)
}

func SetupDeploymentWithReplicas(t *testing.T, image, deploymentName string, replicas int) {
	cmd := exec.Command(`kubectl`, `create`, `deployment`, deploymentName, fmt.Sprintf("--image=%s", image), fmt.Sprintf("--replicas=%d", replicas))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	WaitForDeployment(t, deploymentName)
}

func SetImage(t *testing.T, deploymentName string, deploymentID string, containerName string, image string) {
	cmd := exec.Command(`kubectl`, `set`, `image`, fmt.Sprintf("deployment/%s", deploymentName), fmt.Sprintf("%s=%s", containerName, image))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDeploymentServiceClient(conn)

	WaitForCondition(t, func() bool {
		deployment, err := RetrieveDeployment(service, deploymentID)
		if err != nil {
			Log.Error(err)
			return false
		}
		containers := deployment.GetContainers()
		for _, container := range containers {
			if container.GetImage().GetName().GetFullName() != image {
				return false
			}
		}
		Log.Infof("Image set to %s for deployment %s(%s) container %s", image, deploymentName, deploymentID, containerName)
		return true
	}, "image updated", time.Minute, 5*time.Second)
}

func CreatePod(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Log.Infof("Creating pod %s %s", pod.GetNamespace(), pod.GetName())
	_, err := client.CoreV1().Pods(pod.GetNamespace()).Create(ctx, pod, metaV1.CreateOptions{})
	require.NoError(t, err)

	WaitForDeployment(t, pod.GetName())
}

func TeardownPod(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{GracePeriodSeconds: pointers.Int64(0)})
	require.NoError(t, err)

	WaitForTermination(t, pod.GetName())
}

func TeardownDeployment(t *testing.T, deploymentName string) {
	cmd := exec.Command(`kubectl`, `delete`, `deployment`, deploymentName, `--ignore-not-found=true`, `--grace-period=1`)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	WaitForTermination(t, deploymentName)
}

func TeardownDeploymentWithoutCheck(deploymentName string) {
	// In cases where deployment will not impact other tests,
	// we can trigger deletion and assume that it will be deleted eventually.
	exec.Command(`kubectl`, `delete`, `deployment`, deploymentName, `--ignore-not-found=true`, `--grace-period=1`)
}

func GetConfig(t *testing.T) *rest.Config {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(t, err, "could not load default Kubernetes client config")

	restCfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	require.NoError(t, err, "could not get REST client config from kubernetes config")

	return restCfg
}

func CreateK8sClient(t *testing.T) kubernetes.Interface {
	restCfg := GetConfig(t)
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err, "creating Kubernetes client from REST config")

	return k8sClient
}

func WaitForCondition(t testutils.T, condition func() bool, desc string, timeout time.Duration, frequency time.Duration) {
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
	ks.k8s = CreateK8sClient(ks.T())
}

func (ks *KubernetesSuite) Logf(format string, args ...any) {
	Logf(ks.T(), format, args...)
}

type LogMatcher interface {
	Match(reader io.Reader) (bool, error)
	fmt.Stringer
}

// WaitUntilLog waits until ctx expires or logs of container in all pods matching podLabels satisfy all logMatchers.
func (ks *KubernetesSuite) WaitUntilLog(ctx context.Context, namespace string, podLabels map[string]string, container string, description string, logMatchers ...LogMatcher) {
	ls := labels.SelectorFromSet(podLabels).String()
	checkLogs := func() error {
		podList, err := ks.k8s.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: ls})
		if err != nil {
			return fmt.Errorf("could not list pods matching %q in namespace %q: %w", ls, namespace, err)
		}
		if len(podList.Items) == 0 {
			if ok, err := AllMatch(strings.NewReader(""), logMatchers...); ok {
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
			if ok, err := AllMatch(bytes.NewReader(log), logMatchers...); ok {
				continue
			} else if err != nil {
				return fmt.Errorf("Log of pod %q in namespace %q caused failure: %w", pod.GetName(), namespace, err)
			}
			return fmt.Errorf("Log of pod %q in namespace %q does not satisfy the condition", pod.GetName(), namespace)
		}
		return nil
	}
	ks.Logf("Waiting until %q pods logs "+description+": %s", ls, logMatchers)
	MustEventually(ks.T(), ctx, checkLogs, 10*time.Second, fmt.Sprintf("Not all %q pods logs "+description, ls))
}

// ContainsLineMatching returns a simple line-based regex matcher to go with WaitUntilLog.
// Note: currently limited by bufio.Reader default buffer size (4KB) for simplicity.
func ContainsLineMatching(re *regexp.Regexp) *lineMatcher {
	return &lineMatcher{re: re}
}

func AllMatch(reader io.ReadSeeker, matchers ...LogMatcher) (ok bool, err error) {
	for i, matcher := range matchers {
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return false, fmt.Errorf("could not rewind the reader: %w", err)
		}
		ok, err := matcher.Match(reader)
		if err != nil {
			return false, fmt.Errorf("matcher %d returned an error: %w", i, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// MustEventually retries every pauseInterval until ctx expires or f succeeds.
func MustEventually(t *testing.T, ctx context.Context, f func() error, pauseInterval time.Duration, failureMsgPrefix string) {
	require.NoError(t, retry.WithRetry(f,
		retry.Tries(math.MaxInt),
		retry.WithContext(ctx),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(pauseInterval)
		}),
		retry.OnFailedAttempts(func(err error) { Logf(t, failureMsgPrefix+": %s", err) })))
}

type lineMatcher struct {
	re *regexp.Regexp
}

func (lm *lineMatcher) String() string {
	return fmt.Sprintf("contains line matching %q", lm.re)
}

func (lm *lineMatcher) Match(reader io.Reader) (ok bool, err error) {
	br := bufio.NewReader(reader)
	for {
		// We do not care about partial reads, as the things we look for should fit in default buf size.
		line, _, err := br.ReadLine()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if lm.re.Match(line) {
			return true, nil
		}
	}
}

// CreateService creates a k8s Service object.
func (ks *KubernetesSuite) CreateService(ctx context.Context, namespace string, name string, labels map[string]string, ports map[int32]int32) {
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

// EnsureSecretExists creates a k8s Secret object. If one exists, it makes sure the type and data matches.
func (ks *KubernetesSuite) EnsureSecretExists(ctx context.Context, namespace string, name string, secretType coreV1.SecretType, data map[string][]byte) {
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

// EnsureConfigMapExists creates a k8s ConfigMap object. If one exists, it makes sure the data matches.
func (ks *KubernetesSuite) EnsureConfigMapExists(ctx context.Context, namespace string, name string, data map[string]string) {
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

// WaitUntilCentralSensorConnectionIs makes sure there is only one cluster defined on central, and its status is eventually
// one of the specified statuses.
func WaitUntilCentralSensorConnectionIs(t *testing.T, ctx context.Context, statuses ...storage.ClusterHealthStatus_HealthStatusLabel) {
	Logf(t, "Waiting until central-sensor connection is in state(s) %s...", statuses)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	checkCentralSensorConnection := func() error {
		clustersSvc := v1.NewClustersServiceClient(conn)
		clusters, err := clustersSvc.GetClusters(ctx, &v1.GetClustersRequest{})
		if err != nil {
			return fmt.Errorf("failed to retrieve clusters from central: %w", err)
		}
		if len(clusters.Clusters) != 1 {
			var clusterNames []string
			for _, cluster := range clusters.Clusters {
				clusterNames = append(clusterNames, cluster.Name)
			}
			return fmt.Errorf("expected one cluster, found %d: %+v", len(clusters.Clusters), clusterNames)
		}
		clusterStatus := clusters.Clusters[0].GetHealthStatus().GetSensorHealthStatus()
		for _, status := range statuses {
			if clusterStatus == status {
				Logf(t, "Central-sensor connection now in state %q.", clusterStatus)
				return nil
			}
		}
		return fmt.Errorf("encountered cluster status %q", clusterStatus.String())
	}
	MustEventually(t, ctx, checkCentralSensorConnection, 10*time.Second, "Central-sensor connection not in desired state (yet)")
}

// SetDeploymentEnvVal sets the specified env variable on a container in a deployment using strategic merge patch, or fails the test.
func (ks *KubernetesSuite) SetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string, value string) {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":%q,"env":[{"name":%q,"value":%q}]}]}}}}`,
		container, envVar, value))
	ks.Logf("Setting variable %q on deployment %q in namespace %q to %q", envVar, deployment, namespace, value)
	_, err := ks.k8s.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	ks.Require().NoError(err, "cannot patch deployment %q in namespace %q", deployment, namespace)

}

// GetDeploymentEnvVal retrieves the value of environment variable in a deployment, or fails the test.
func (ks *KubernetesSuite) GetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string) string {
	d, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deployment, metaV1.GetOptions{})
	ks.Require().NoError(err, "cannot retrieve deployment %q in namespace %q", deployment, namespace)
	c, err := getContainer(d, container)
	ks.Require().NoError(err, "cannot find container %q in deployment %q in namespace %q", container, deployment, namespace)
	val, err := getEnvVal(c, envVar)
	ks.Require().NoError(err, "cannot find envVar %q in container %q in deployment %q in namespace %q", envVar, container, deployment, namespace)
	return val
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

// getEnvVal returns the given container from a deployment or returns a helpful error.
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

// CollectLogs collects service logs from a given namespace into a subdirectory that will be gathered by the CI scripts.
func CollectLogs(t *testing.T, ns string, dir string) {
	err := exec.Command("../scripts/ci/collect-service-logs.sh", ns, "/tmp/e2e-test-logs/"+dir).Run()
	if err != nil {
		t.Logf("Collecting %q logs returned error %s", ns, err)
	}
}
