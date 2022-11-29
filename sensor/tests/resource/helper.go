package resource

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/networking/v1"
	v12 "k8s.io/api/rbac/v1"

	// import gcp
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	// DefaultNamespace the default namespace used to create the resources
	DefaultNamespace string = "sensor-integration"

	// defaultCreationTimeout maximum time the test will wait until sensor emits
	// resource creation event to central after fake resource was applied.
	defaultCreationTimeout time.Duration = 10 * time.Second
)

// YamlTestFile is a test file in YAML
type YamlTestFile struct {
	Kind string
	File string
}

// requiredWaitResources slice of resources that need to be awaited
var requiredWaitResources = []string{"Service"}

func shouldWaitForResource(kind string) bool {
	for _, k := range requiredWaitResources {
		if k == kind {
			return true
		}
	}
	return false
}

// objByKind returns the supported dynamic k8s resources that can be created
// add new ones here to support adding new resource files
func objByKind(kind string) k8s.Object {
	switch kind {
	case "Deployment":
		return &appsV1.Deployment{}
	case "Role":
		return &v12.Role{}
	case "Binding":
		return &v12.RoleBinding{}
	case "Pod":
		return &v1.Pod{}
	case "ServiceAccount":
		return &v1.ServiceAccount{}
	case "NetworkPolicy":
		return &v13.NetworkPolicy{}
	case "Service":
		return &v1.Service{}
	default:
		log.Fatalf("unrecognized resource kind %s\n", kind)
		return nil
	}
}

// TestCallback represents the test case function written in the go test file.
type TestCallback func(t *testing.T, testContext *TestContext, objects map[string]k8s.Object)

// TestContext holds all the information about the cluster and sensor under test. A TestContext represents
// a test case run where the input is a set of resources applied to the cluster and the output is a set of
// messages emitted by Sensor. Each Go test should use a single TestContext instance to manage cluster interaction
// and assertions.
type TestContext struct {
	t               *testing.T
	r               *resources.Resources
	env             *envconf.Config
	fakeCentral     *centralDebug.FakeService
	centralReceived chan *central.MsgFromSensor
	stopFn          func()
}

func defaultCentralConfig() CentralConfig {
	// Uses replayed policies.json file as default policies for tests.
	// These are all policies in ACS, which means many alerts might be generated.
	policies, err := testutils.GetPoliciesFromFile("../../replay/data/policies.json")
	if err != nil {
		log.Fatalln(err)
	}

	return CentralConfig{
		InitialSystemPolicies: policies,
	}
}

// NewContext creates a new test context with default configuration.
func NewContext(t *testing.T) (*TestContext, error) {
	return NewContextWithConfig(t, defaultCentralConfig())
}

// NewContextWithConfig creates a new test context with custom central configuration.
func NewContextWithConfig(t *testing.T, config CentralConfig) (*TestContext, error) {
	envConfig := envconf.New().WithKubeconfigFile(conf.ResolveKubeConfigFile())
	r, err := resources.New(envConfig.Client().RESTConfig())
	data, _ := os.Open(conf.ResolveKubeConfigFile())
	fileScanner := bufio.NewScanner(data)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.Contains(line, "cluster: ") {
			log.Printf("KUBECONFIG = %s", line)
		}
	}
	if err != nil {
		return nil, err
	}
	fakeCentral, startFn, stopFn := startSensorAndFakeCentral(envConfig, config)
	ch := make(chan *central.MsgFromSensor, 100)
	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		ch <- msg
	})

	startFn()
	return &TestContext{
		t, r, envConfig, fakeCentral, ch, stopFn,
	}, nil
}

// Stop test context and sensor.
func (c *TestContext) Stop() {
	c.stopFn()
}

// Resources object is used to interact with the cluster and apply new resources.
func (c *TestContext) Resources() *resources.Resources {
	return c.r
}

func createTestNs(ctx context.Context, r *resources.Resources, name string) (*v1.Namespace, func() error, error) {
	nsObj := v1.Namespace{}
	nsObj.Name = name
	if err := r.Create(ctx, &nsObj); err != nil {
		return nil, nil, err
	}
	return &nsObj, func() error {
		err := r.Delete(ctx, &nsObj)
		if err != nil {
			return err
		}

		// wait for deletion to be finished
		if err := wait.For(conditions.New(r).ResourceDeleted(&nsObj)); err != nil {
			fmt.Println("failed to wait for namespace deletion")
		}
		return nil
	}, nil
}

// GetFakeCentral gets a fake central instance. This is used to fetch messages sent by sensor under test.
func (c *TestContext) GetFakeCentral() *centralDebug.FakeService {
	return c.fakeCentral
}

// RunWithResources runs the test case applying files in `files` slice in order.
func (c *TestContext) RunWithResources(files []YamlTestFile, testCase TestCallback) {
	_, removeNamespace, err := createTestNs(context.Background(), c.r, DefaultNamespace)
	defer utils.IgnoreError(removeNamespace)
	if err != nil {
		c.t.Fatalf("failed to create namespace: %s", err)
	}
	var removeFunctions []func() error
	fileToObj := map[string]k8s.Object{}
	for _, file := range files {
		obj := objByKind(file.Kind)
		removeFn, err := c.ApplyFile(context.Background(), DefaultNamespace, file, obj)
		if err != nil {
			c.t.Fatalf("fail to apply resource: %s", err)
		}
		removeFunctions = append(removeFunctions, removeFn)
		fileToObj[file.File] = obj
	}
	defer func() {
		for _, fn := range removeFunctions {
			utils.IgnoreError(fn)
		}
	}()
	testCase(c.t, c, fileToObj)
}

// RunBare runs a test case without applying any resources to the cluster.
func (c *TestContext) RunBare(name string, testCase TestCallback) {
	c.t.Run(name, func(t *testing.T) {
		_, removeNamespace, err := createTestNs(context.Background(), c.r, DefaultNamespace)
		defer utils.IgnoreError(removeNamespace)
		if err != nil {
			t.Fatalf("failed to create namespace: %s", err)
		}
		testCase(t, c, nil)
	})
}

// RunWithResourcesPermutation runs the test cases using `files` similarly to `RunWithResources` but it will run the
// test case for each possible permutation of `files` slice.
func (c *TestContext) RunWithResourcesPermutation(files []YamlTestFile, name string, testCase TestCallback) {
	runPermutation(files, 0, func(f []YamlTestFile) {
		newF := make([]YamlTestFile, len(f))
		copy(newF, f)
		c.t.Run(fmt.Sprintf("%s_Permutation_%s", name, permutationKind(newF)), func(t *testing.T) {
			c.RunWithResources(newF, testCase)
		})
	})
}

func permutationKind(perm []YamlTestFile) string {
	kinds := make([]string, len(perm))
	for i, p := range perm {
		kinds[i] = p.Kind
	}
	return strings.Join(kinds, "_")
}

func runPermutation(files []YamlTestFile, i int, cb func([]YamlTestFile)) {
	if i > len(files) {
		cb(files)
		return
	}
	runPermutation(files, i+1, cb)
	for j := i + 1; j < len(files); j++ {
		files[i], files[j] = files[j], files[i]
		runPermutation(files, i+1, cb)
		files[i], files[j] = files[j], files[i]
	}
}

// AssertFunc is the deployment state assertion function signature.
type AssertFunc func(deployment *storage.Deployment) error

// LastDeploymentState checks the deployment state similarly to `LastDeploymentStateWithTimeout` with a default 3 seconds timeout.
func (c *TestContext) LastDeploymentState(name string, assertion AssertFunc, message string) {
	c.LastDeploymentStateWithTimeout(name, assertion, message, 3*time.Second)
}

// LastDeploymentStateWithTimeout checks that a deployment reaches a state asserted by `assertion`. If the deployment does not reach
// that state until `timeout` the test fails.
func (c *TestContext) LastDeploymentStateWithTimeout(name string, assertion AssertFunc, message string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	var lastErr error
	for {
		select {
		case <-timer.C:
			c.t.Fatalf("timeout reached waiting for state: (%s): %s", message, lastErr)
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			lastDeploymentUpdate := GetLastMessageWithDeploymentName(messages, "sensor-integration", name)
			deployment := lastDeploymentUpdate.GetEvent().GetDeployment()
			if deployment != nil {
				if lastErr = assertion(deployment); lastErr == nil {
					return
				}
			}
		}
	}
}

// AlertAssertFunc is the alert assertion function signature.
type AlertAssertFunc func(alertResults *central.AlertResults) error

// LastViolationState checks the violation state similarly to `LastViolationStateWithTimeout` with a default 3 seconds timeout.
func (c *TestContext) LastViolationState(name string, assertion AlertAssertFunc, message string) {
	c.LastViolationStateWithTimeout(name, assertion, message, 3*time.Second)
}

// LastViolationStateWithTimeout checks that a violation state for a deployment must match `assertion`. If violation state does not match
// until `timeout` the test fails.
func (c *TestContext) LastViolationStateWithTimeout(name string, assertion AlertAssertFunc, message string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	var lastErr error
	for {
		select {
		case <-timer.C:
			c.t.Fatalf("timeout reached waiting for violation state (%s): %s", message, lastErr)
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			alerts := GetAllAlertsForDeploymentName(messages, name)
			var lastViolationState *central.AlertResults
			if len(alerts) > 0 {
				lastViolationState = alerts[len(alerts)-1].GetEvent().GetAlertResults()
			}
			if lastErr = assertion(lastViolationState); lastErr == nil {
				// Assertion matched the case. We can return here without failing the test case
				return
			}
		}
	}

}

// GetAllAlertsForDeploymentName filters sensor messages and gets all alerts for a deployment with `name`
func GetAllAlertsForDeploymentName(messages []*central.MsgFromSensor, name string) []*central.MsgFromSensor {
	var selected []*central.MsgFromSensor
	for _, m := range messages {
		for _, alert := range m.GetEvent().GetAlertResults().GetAlerts() {
			if alert.GetDeployment().GetName() == name {
				selected = append(selected, m)
				break
			}
		}
	}
	return selected
}

// CentralConfig allows tests to inject ACS policies in the tests
type CentralConfig struct {
	InitialSystemPolicies []*storage.Policy
}

func startSensorAndFakeCentral(env *envconf.Config, config CentralConfig) (*centralDebug.FakeService, func(), func()) {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../../../tools/local-sensor/certs/caKey.pem"))

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("00000000-0000-4000-A000-000000000000"),
		message.ClusterConfig(),
		message.PolicySync(config.InitialSystemPolicies),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	conn, spyCentral, _ := createConnectionAndStartServer(fakeCentral)
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(client.MustCreateInterfaceFromRest(env.Client().RESTConfig())).
		WithLocalSensor(true).
		WithResyncPeriod(1 * time.Second).
		WithCentralConnectionFactory(fakeConnectionFactory))

	if err != nil {
		panic(err)
	}

	return fakeCentral, func() {
			go s.Start()
			spyCentral.ConnectionStarted.Wait()
		}, func() {
			go s.Stop()
			spyCentral.KillSwitch.Done()
		}
}

func createConnectionAndStartServer(fakeCentral *centralDebug.FakeService) (*grpc.ClientConn, *centralDebug.FakeService, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer()
	central.RegisterSensorServiceServer(server, fakeCentral)

	go func() {
		utils.IgnoreError(func() error {
			return server.Serve(listener)
		})
	}()

	conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		panic(err)
	}

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, fakeCentral, closeF
}

// ApplyFileNoObject - apply a file without an object
func (c *TestContext) ApplyFileNoObject(ctx context.Context, ns string, file YamlTestFile) (func() error, error) {
	obj := objByKind(file.Kind)
	return c.ApplyFile(ctx, ns, file, obj)
}

// ApplyFile - applies a file
func (c *TestContext) ApplyFile(ctx context.Context, ns string, file YamlTestFile, obj k8s.Object) (func() error, error) {
	d := os.DirFS("yaml")
	if err := decoder.DecodeFile(
		d,
		file.File,
		obj,
		decoder.MutateNamespace(ns),
	); err != nil {
		return nil, err
	}

	if shouldWaitForResource(file.Kind) {
		if err := execWithRetry(5*time.Minute, 5*time.Second, func() error {
			return c.r.Create(ctx, obj)
		}); err != nil {
			return nil, err
		}
	} else {
		if err := c.r.Create(ctx, obj); err != nil {
			return nil, err
		}
	}

	if file.Kind == "Deployment" || file.Kind == "Pod" {
		if err := c.waitForResource(defaultCreationTimeout, deploymentName(obj.GetName())); err != nil {
			return nil, err
		}
	}

	return func() error {
		if shouldWaitForResource(file.Kind) {
			err := c.r.Delete(ctx, obj)
			if err != nil {
				return err
			}

			// wait for deletion to be finished
			if err := wait.For(conditions.New(c.r).ResourceDeleted(obj)); err != nil {
				fmt.Println("failed to wait for resource deletion")
			}
			return nil
		}
		return c.r.Delete(ctx, obj)
	}, nil
}

func execWithRetry(timeout, interval time.Duration, fn backoff.Operation) error {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = timeout
	exponential.MaxInterval = interval
	var notifyErr error
	if backoffErr := backoff.RetryNotify(fn, exponential, func(err error, d time.Duration) {
		notifyErr = errors.Wrap(err, "timeout reached waiting for resource")
	}); backoffErr != nil {
		return backoffErr
	}
	if notifyErr != nil {
		return notifyErr
	}
	return nil
}

type condition func(event *central.SensorEvent) bool

func deploymentName(s string) condition {
	return func(event *central.SensorEvent) bool {
		return event.GetDeployment().GetName() == s
	}
}

func (c *TestContext) waitForResource(timeout time.Duration, fn condition) error {
	afterTimeout := time.After(timeout)
	for {
		select {
		case <-afterTimeout:
			return errors.New("timeout reached waiting for event")
		case d, more := <-c.centralReceived:
			if !more {
				return errors.New("channel closed")
			}
			if fn(d.GetEvent()) {
				return nil
			}
		}
	}
}

// GetLastMessageWithDeploymentName find most recent sensor messages by namespace and deployment name
func GetLastMessageWithDeploymentName(messages []*central.MsgFromSensor, ns, name string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		deployment := messages[i].GetEvent().GetDeployment()
		if deployment.GetName() == name && deployment.GetNamespace() == ns {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

// GetLastAlertsWithDeploymentID find most recent alert message by deployment ID
func GetLastAlertsWithDeploymentID(messages []*central.MsgFromSensor, id string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].GetEvent().GetAlertResults().GetDeploymentId() == id {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

// GetUniquePodNamesFromPrefix find all unique pod names from sensor events
func GetUniquePodNamesFromPrefix(messages []*central.MsgFromSensor, ns, prefix string) []string {
	uniqueNames := set.NewStringSet()
	for _, msg := range messages {
		pod := msg.GetEvent().GetPod()
		if pod != nil && pod.GetNamespace() == ns {
			if strings.Contains(pod.GetName(), prefix) {
				uniqueNames.Add(pod.GetName())
			}
		}
	}
	return uniqueNames.AsSlice()
}

// GetUniqueDeploymentNames find all unique deployment names from sensor events
func GetUniqueDeploymentNames(messages []*central.MsgFromSensor, ns string) []string {
	uniqueNames := set.NewStringSet()
	for _, msg := range messages {
		deployment := msg.GetEvent().GetDeployment()
		if deployment != nil && deployment.GetNamespace() == ns {
			uniqueNames.Add(deployment.GetName())
		}
	}
	return uniqueNames.AsSlice()
}
