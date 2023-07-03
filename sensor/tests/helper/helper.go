package helper

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
	// resource creation event to central after a resource was applied.
	defaultCreationTimeout = 30 * time.Second

	// defaultWaitTimeout maximum time the test will wait for a specific assertion
	defaultWaitTimeout = 3 * time.Second

	// defaultTicker the default interval for the assertion functions to retry the assertion
	defaultTicker = 500 * time.Millisecond

	// certID is the id in the certificate which is sent on the hello message
	certID = "00000000-0000-4000-A000-000000000000"
)

// K8sResourceInfo is a test file in YAML or a struct
type K8sResourceInfo struct {
	Kind     string
	YamlFile string
	Obj      interface{}
	Name     string
}

// requiredWaitResources slice of resources that need to be awaited
var requiredWaitResources = []string{"Service"}

func shouldRetryResource(kind string) bool {
	for _, k := range requiredWaitResources {
		if k == kind {
			return true
		}
	}
	return false
}

// objByKind returns the supported dynamic k8s resources that can be created
// add new ones here to support adding new resource types
func objByKind(kind string) k8s.Object {
	switch kind {
	case "Deployment":
		return &appsV1.Deployment{}
	case "Role":
		return &v12.Role{}
	case "Binding":
		return &v12.RoleBinding{}
	case "ClusterRole":
		return &v12.ClusterRole{}
	case "ClusterRoleBinding":
		return &v12.ClusterRoleBinding{}
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
	sensorStopped   concurrency.ReadOnlyErrorSignal
	centralStopped  atomic.Bool
	config          CentralConfig
	grpcFactory     centralDebug.FakeGRPCFactory

	// archivedMessages holds messages sent from Sensor to FakeCentral before stopping Central. These can be fetched
	// in case the test needs to assert on messages sent right before stopping the gRPC connection.
	archivedMessages [][]*central.MsgFromSensor
}

// DefaultCentralConfig hold default values when starting local sensor in tests.
func DefaultCentralConfig() CentralConfig {
	// Uses replayed policies.json file as default policies for tests.
	// These are all policies in ACS, which means many alerts might be generated.
	policies, err := testutils.GetPoliciesFromFile("../../replay/data/policies.json")
	if err != nil {
		log.Fatalln(err)
	}

	return CentralConfig{
		InitialSystemPolicies: policies,
		CertFilePath:          "../../../../tools/local-sensor/certs/",
	}
}

// NewContext creates a new test context with default configuration.
func NewContext(t *testing.T) (*TestContext, error) {
	return NewContextWithConfig(t, DefaultCentralConfig())
}

// NewContextWithConfig creates a new test context with custom central configuration.
func NewContextWithConfig(t *testing.T, config CentralConfig) (*TestContext, error) {
	envConfig := envconf.New().WithKubeconfigFile(conf.ResolveKubeConfigFile())
	r, err := resources.New(envConfig.Client().RESTConfig())
	if err != nil {
		return nil, err
	}

	tc := TestContext{
		t:                t,
		r:                r,
		env:              envConfig,
		centralStopped:   atomic.Bool{},
		config:           config,
		archivedMessages: [][]*central.MsgFromSensor{},
	}

	tc.StartFakeGRPC()
	tc.startSensorInstance(envConfig)

	return &tc, nil
}

// WithPermutation sets whether the test should run with permutations
func WithPermutation() TestRunFunc {
	return func(t *testRun) {
		t.permutation = true
	}
}

// WithResources sets the resources to be created by the test
func WithResources(resources []K8sResourceInfo) TestRunFunc {
	return func(t *testRun) {
		t.resources = resources
	}
}

// WithTestCase sets the TestCallback function to be run
func WithTestCase(test TestCallback) TestRunFunc {
	return func(t *testRun) {
		t.testCase = test
	}
}

// WithRetryCallback sets the RetryCallback function
func WithRetryCallback(retryCallback RetryCallback) TestRunFunc {
	return func(t *testRun) {
		t.retryCallback = retryCallback
	}
}

// RunTest runs a test case. Fails the test if the testRun cannot be created.
func (c *TestContext) RunTest(options ...TestRunFunc) {
	tr, err := newTestRun(options...)
	if err != nil {
		c.t.Fatal(err)
	}
	c.run(tr)
}

// Stop test context and sensor.
func (c *TestContext) Stop() {
	c.stopFn()
}

// Resources object is used to interact with the cluster and apply new resources.
func (c *TestContext) Resources() *resources.Resources {
	return c.r
}

func (c *TestContext) deleteNs(ctx context.Context, name string) error {
	nsObj := v1.Namespace{}
	nsObj.Name = name
	err := c.r.Delete(ctx, &nsObj)
	if err != nil {
		return err
	}

	// wait for deletion to be finished
	if err := wait.For(conditions.New(c.r).ResourceDeleted(&nsObj)); err != nil {
		c.t.Logf("failed to wait for namespace %s deletion\n", nsObj.Name)
	}
	return nil
}

// SensorStopped checks if sensor under test stopped.
func (c *TestContext) SensorStopped() bool {
	return c.sensorStopped.IsDone()
}

func (c *TestContext) createTestNs(ctx context.Context, name string) (*v1.Namespace, func() error, error) {
	utils.IgnoreError(func() error {
		return c.deleteNs(ctx, name)
	})
	nsObj := v1.Namespace{}
	nsObj.Name = name
	if err := c.r.Create(ctx, &nsObj); err != nil {
		return nil, nil, err
	}
	return &nsObj, func() error {
		return c.deleteNs(ctx, name)
	}, nil
}

// ArchivedMessages returns a slice of slices, each contain messages received by Central before restarting
func (c *TestContext) ArchivedMessages() [][]*central.MsgFromSensor {
	return c.archivedMessages
}

// StopCentralGRPC will attempt to stop fake central. If it was already stopped, nothing happens
func (c *TestContext) StopCentralGRPC() {
	if c.centralStopped.CompareAndSwap(false, true) {
		c.fakeCentral.Stop()
		messagesBeforeStopping := c.fakeCentral.GetAllMessages()
		c.archivedMessages = append(c.archivedMessages, messagesBeforeStopping)
	}
}

// RestartFakeCentralConnection creates a new fake central connection and updates factory pointers.
// It calls StopCentralGRPC to make sure current fake central is stopped before creating new instance.
func (c *TestContext) RestartFakeCentralConnection() {
	c.StopCentralGRPC()
	c.StartFakeGRPC()
}

// StartFakeGRPC will start a gRPC server to act as Central.
func (c *TestContext) StartFakeGRPC() {
	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello(certID),
		message.ClusterConfig(),
		message.PolicySync(c.config.InitialSystemPolicies),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	conn, shutdown := createConnectionAndStartServer(fakeCentral)

	// grpcFactory will be nil on the first run of the testContext
	if c.grpcFactory == nil {
		c.grpcFactory = centralDebug.MakeFakeConnectionFactory(conn)
	} else {
		c.grpcFactory.OverwriteCentralConnection(conn)
	}

	fakeCentral.OnShutdown(shutdown)
	c.fakeCentral = fakeCentral
}

// GetFakeCentral gets a fake central instance. This is used to fetch messages sent by sensor under test.
func (c *TestContext) GetFakeCentral() *centralDebug.FakeService {
	return c.fakeCentral
}

// run calls the test function depending on the configuration of the testRun.
// For example, if permutation is set to true, it will run call runWithResourcesPermutation.
func (c *TestContext) run(t *testRun) {
	if t.resources == nil {
		c.runBare(t.testCase)
	} else {
		if t.permutation {
			c.runWithResourcesPermutation(t)
		} else {
			if err := c.runWithResources(t.resources, t.testCase, t.retryCallback); err != nil {
				c.t.Fatalf(err.Error())
			}
		}
	}
}

// runWithResources runs the test case applying resources in `resources` slice in order.
// If it is set, the RetryCallback will be called if the application of a resource fails.
func (c *TestContext) runWithResources(resources []K8sResourceInfo, testCase TestCallback, retryFn RetryCallback) error {
	_, removeNamespace, err := c.createTestNs(context.Background(), DefaultNamespace)
	if err != nil {
		return errors.Errorf("failed to create namespace: %s", err)
	}
	defer utils.IgnoreError(removeNamespace)
	var removeFunctions []func() error
	fileToObj := map[string]k8s.Object{}
	for i := range resources {
		obj := objByKind(resources[i].Kind)
		removeFn, err := c.ApplyResourceAndWait(context.Background(), DefaultNamespace, &resources[i], obj, retryFn)
		if err != nil {
			return errors.Errorf("fail to apply resource: %s", err)
		}
		removeFunctions = append(removeFunctions, removeFn)
		fileToObj[resources[i].Name] = obj
	}
	defer func() {
		for _, fn := range removeFunctions {
			utils.IgnoreError(fn)
		}
	}()
	testCase(c.t, c, fileToObj)
	return nil
}

// runBare runs a test case without applying any resources to the cluster.
func (c *TestContext) runBare(testCase TestCallback) {
	_, removeNamespace, err := c.createTestNs(context.Background(), DefaultNamespace)
	defer utils.IgnoreError(removeNamespace)
	if err != nil {
		c.t.Fatalf("failed to create namespace: %s", err)
	}
	testCase(c.t, c, nil)
}

// runWithResourcesPermutation runs the test cases using `resources` similarly to `runWithResources` but it will run the
// test case for each possible permutation of `resources` slice.
func (c *TestContext) runWithResourcesPermutation(t *testRun) {
	runPermutation(t.resources, 0, func(f []K8sResourceInfo) {
		newF := make([]K8sResourceInfo, len(f))
		copy(newF, f)
		newTestRun := t.copy()
		newTestRun.resources = newF
		c.t.Run(fmt.Sprintf("Permutation_%s", permutationKind(newF)), func(_ *testing.T) {
			if err := c.runWithResources(t.resources, t.testCase, t.retryCallback); err != nil {
				c.t.Fatal(err.Error())
			}
		})
	})
}

func permutationKind(perm []K8sResourceInfo) string {
	kinds := make([]string, len(perm))
	for i, p := range perm {
		kinds[i] = p.Kind
	}
	return strings.Join(kinds, "_")
}

func runPermutation(resources []K8sResourceInfo, i int, cb func([]K8sResourceInfo)) {
	if i > len(resources) {
		cb(resources)
		return
	}
	runPermutation(resources, i+1, cb)
	for j := i + 1; j < len(resources); j++ {
		resources[i], resources[j] = resources[j], resources[i]
		runPermutation(resources, i+1, cb)
		resources[i], resources[j] = resources[j], resources[i]
	}
}

// AssertFunc is the deployment state assertion function signature.
type AssertFunc func(deployment *storage.Deployment, action central.ResourceAction) error

// MatchResource is a function to match sensor messages to be filtered.
type MatchResource func(resource *central.MsgFromSensor) bool

// AssertFuncAny is similar to AssertFunc but generic to any type of resource.
type AssertFuncAny func(resource interface{}) error

// LastResourceState same as LastResourceStateWithTimeout with a 3s default timeout.
func (c *TestContext) LastResourceState(matchResourceFn MatchResource, assertFn AssertFuncAny, message string) {
	c.LastResourceStateWithTimeout(matchResourceFn, assertFn, message, defaultWaitTimeout)
}

// LastResourceStateWithTimeout filters all messages by `matchResourceFn` and checks that the last message matches `assertFn`. Timeouts after `timeout`.
func (c *TestContext) LastResourceStateWithTimeout(matchResourceFn MatchResource, assertFn AssertFuncAny, message string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(defaultTicker)
	lastErr := errors.New("no resource found for matching function")
	for {
		select {
		case <-timer.C:
			c.t.Fatalf("timeout reached waiting for state: (%s): %s", message, lastErr)
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			msg := GetLastMessageMatching(messages, matchResourceFn)
			if msg != nil {
				lastErr = assertFn(msg.GetEvent())
				if lastErr == nil {
					return
				}
			}
		}
	}
}

// WaitForSyncEvent will wait until sensor transmits a `Synced` event to Central, at the end of the reconciliation.
func (c *TestContext) WaitForSyncEvent(timeout time.Duration) {
	ticker := time.NewTicker(defaultTicker)
	timeoutTimer := time.NewTicker(timeout)
	for {
		select {
		case <-timeoutTimer.C:
			c.t.Errorf("timeout (%s) reached waiting for sync event", timeout)
			return
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			for _, m := range messages {
				if m.GetEvent().GetSynced() != nil {
					return
				}
			}
		}
	}
}

// WaitForDeploymentEvent waits until sensor process a given deployment
func (c *TestContext) WaitForDeploymentEvent(name string) {
	c.WaitForDeploymentEventWithTimeout(name, defaultWaitTimeout)
}

// WaitForDeploymentEventWithTimeout waits until sensor process a given deployment
func (c *TestContext) WaitForDeploymentEventWithTimeout(name string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(defaultTicker)
	lastErr := errors.Errorf("the deployment %s was not sent", name)
	for {
		select {
		case <-timer.C:
			c.t.Fatalf("timeout reached waiting for deployment: %s", lastErr)
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			lastDeploymentUpdate := GetLastMessageWithDeploymentName(messages, DefaultNamespace, name)
			deployment := lastDeploymentUpdate.GetEvent().GetDeployment()
			if deployment != nil {
				return
			}
		}
	}

}

// LastDeploymentState checks the deployment state similarly to `LastDeploymentStateWithTimeout` with a default 3 seconds timeout.
func (c *TestContext) LastDeploymentState(name string, assertion AssertFunc, message string) {
	c.LastDeploymentStateWithTimeout(name, assertion, message, defaultWaitTimeout)
}

// LastDeploymentStateWithTimeout checks that a deployment reaches a state asserted by `assertion`. If the deployment does not reach
// that state until `timeout` the test fails.
func (c *TestContext) LastDeploymentStateWithTimeout(name string, assertion AssertFunc, message string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(defaultTicker)
	lastErr := errors.New("no deployment found")
	for {
		select {
		case <-timer.C:
			c.t.Errorf("timeout reached waiting for state: (%s): %s", message, lastErr)
			return
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			lastDeploymentUpdate := GetLastMessageWithDeploymentName(messages, DefaultNamespace, name)
			deployment := lastDeploymentUpdate.GetEvent().GetDeployment()
			action := lastDeploymentUpdate.GetEvent().GetAction()
			if deployment != nil {
				if lastErr = assertion(deployment, action); lastErr == nil {
					return
				}
			}
		}
	}
}

// DeploymentCreateReceived checks if a deployment object was received with CREATE action.
func (c *TestContext) DeploymentCreateReceived(name string) {
	c.DeploymentActionReceived(name, central.ResourceAction_CREATE_RESOURCE)
}

// DeploymentActionReceived checks if a deployment object was received with specific action type.
func (c *TestContext) DeploymentActionReceived(name string, expectedAction central.ResourceAction) {
	c.LastDeploymentState(name, func(_ *storage.Deployment, action central.ResourceAction) error {
		if action != expectedAction {
			return errors.Errorf("event action is %s, but expected %s", action, expectedAction)
		}
		return nil
	}, fmt.Sprintf("Deployment %s should be received with action %s", name, expectedAction))
}

// GetLastMessageMatching finds last element in slice matching `matchFn`.
func GetLastMessageMatching(messages []*central.MsgFromSensor, matchFn MatchResource) *central.MsgFromSensor {
	for i := len(messages) - 1; i >= 0; i-- {
		if matchFn(messages[i]) {
			return messages[i]
		}
	}
	return nil
}

// AlertAssertFunc is the alert assertion function signature.
type AlertAssertFunc func(alertResults *central.AlertResults) error

// LastViolationState checks the violation state similarly to `LastViolationStateWithTimeout` with a default 3 seconds timeout.
func (c *TestContext) LastViolationState(name string, assertion AlertAssertFunc, message string) {
	c.LastViolationStateWithTimeout(name, assertion, message, defaultWaitTimeout)
}

// LastViolationStateWithTimeout checks that a violation state for a deployment must match `assertion`. If violation state does not match
// until `timeout` the test fails.
func (c *TestContext) LastViolationStateWithTimeout(name string, assertion AlertAssertFunc, message string, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(defaultTicker)
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

// LastViolationStateByID checks the violation state by deployment ID
func (c *TestContext) LastViolationStateByID(id string, assertion AlertAssertFunc, message string, checkEmptyAlertResults bool) {
	c.LastViolationStateByIDWithTimeout(id, assertion, message, checkEmptyAlertResults, defaultWaitTimeout)
}

// LastViolationStateByIDWithTimeout checks that a violation state for a deployment must match `assertion`. If violation state does not match
// until `timeout` the test fails.
func (c *TestContext) LastViolationStateByIDWithTimeout(id string, assertion AlertAssertFunc, message string, checkEmptyAlertResults bool, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(defaultTicker)
	lastErr := errors.Errorf("no alerts sent for deployment ID %s", id)
	for {
		select {
		case <-timer.C:
			c.t.Fatalf("timeout reached waiting for violation state (%s): %s", message, lastErr)
		case <-ticker.C:
			messages := c.GetFakeCentral().GetAllMessages()
			lastAlert := GetLastAlertsWithDeploymentID(messages, id, checkEmptyAlertResults)
			lastViolationState := lastAlert.GetEvent().GetAlertResults()
			if lastViolationState == nil {
				continue
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
	CertFilePath          string
}

func (c *TestContext) startSensorInstance(env *envconf.Config) {
	c.t.Setenv("ROX_MTLS_CERT_FILE", path.Join(c.config.CertFilePath, "/cert.pem"))
	c.t.Setenv("ROX_MTLS_KEY_FILE", path.Join(c.config.CertFilePath, "/key.pem"))
	c.t.Setenv("ROX_MTLS_CA_FILE", path.Join(c.config.CertFilePath, "/caCert.pem"))
	c.t.Setenv("ROX_MTLS_CA_KEY_FILE", path.Join(c.config.CertFilePath, "/caKey.pem"))

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(client.MustCreateInterfaceFromRest(env.Client().RESTConfig())).
		WithLocalSensor(true).
		WithResyncPeriod(1 * time.Second).
		WithCentralConnectionFactory(c.grpcFactory))

	if err != nil {
		panic(err)
	}

	c.sensorStopped = s.Stopped()
	c.stopFn = func() {
		go s.Stop()
		c.fakeCentral.KillSwitch.Done()
	}

	go s.Start()
	c.fakeCentral.ConnectionStarted.Wait()
}

func createConnectionAndStartServer(fakeCentral *centralDebug.FakeService) (*grpc.ClientConn, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	fakeCentral.ServerPointer = grpc.NewServer()
	central.RegisterSensorServiceServer(fakeCentral.ServerPointer, fakeCentral)

	go func() {
		utils.IgnoreError(func() error {
			return fakeCentral.ServerPointer.Serve(listener)
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
		fakeCentral.ServerPointer.Stop()
	}

	return conn, closeF
}

// ApplyResourceAndWaitNoObject creates a Kubernetes resource using `ApplyResourceAndWait` without requiring an object reference.
// Use this if there is no need to get or manipulate the data in the YAML file.
func (c *TestContext) ApplyResourceAndWaitNoObject(ctx context.Context, ns string, resource K8sResourceInfo, retryFn RetryCallback) (func() error, error) {
	obj := objByKind(resource.Kind)
	return c.ApplyResourceAndWait(ctx, ns, &resource, obj, retryFn)
}

// ApplyResourceAndWait calls ApplyResource and waits for the resource if it's "waitable" (e.g. Deployment or Pod).
func (c *TestContext) ApplyResourceAndWait(ctx context.Context, ns string, resource *K8sResourceInfo, obj k8s.Object, retryFn RetryCallback) (func() error, error) {
	fn, err := c.ApplyResource(ctx, ns, resource, obj, retryFn)
	if err != nil {
		return nil, err
	}

	if resource.Kind == "Deployment" || resource.Kind == "Pod" {
		if err := c.waitForResource(defaultCreationTimeout, deploymentName(obj.GetName())); err != nil {
			return nil, err
		}
	}

	return fn, nil
}

// ApplyResource creates a Kubernetes resource in namespace `ns` from a resource definition (see
// `K8sResourceInfo` for more details). Once the resource is applied, the `obj` will be populated
// with the properties from the resource definition. In case the creation fails (due to the client
// API rejecting the definition), a `RetryCallback` function can be provided to manipulate the
// object prior to the retry.
func (c *TestContext) ApplyResource(ctx context.Context, ns string, resource *K8sResourceInfo, obj k8s.Object, retryFn RetryCallback) (func() error, error) {
	if resource.Obj != nil {
		var ok bool
		obj, ok = resource.Obj.(k8s.Object)
		if !ok {
			return nil, errors.New("invalid k8s.Object")
		}
		resource.Name = obj.GetName()
	} else {
		d := os.DirFS("yaml")
		if err := decoder.DecodeFile(
			d,
			resource.YamlFile,
			obj,
			decoder.MutateNamespace(ns),
		); err != nil {
			return nil, err
		}
		resource.Name = obj.GetName()
	}

	if shouldRetryResource(resource.Kind) || retryFn != nil {
		if err := execWithRetry(defaultCreationTimeout, 5*time.Second, func() error {
			err := c.r.Create(ctx, obj)
			if err != nil && retryFn != nil {
				if retryErr := retryFn(err, obj); retryErr != nil {
					c.t.Fatal(errors.Wrapf(err, "error in retry callback: %s", retryErr))
				}
			}
			return err
		}); err != nil {
			return nil, err
		}
	} else {
		if err := c.r.Create(ctx, obj); err != nil {
			return nil, err
		}
	}

	return func() error {
		if shouldRetryResource(resource.Kind) {
			err := c.r.Delete(ctx, obj)
			if err != nil {
				return err
			}

			// wait for deletion to be finished
			if err := wait.For(conditions.New(c.r).ResourceDeleted(obj)); err != nil {
				c.t.Logf("failed to wait for resource deletion")
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
		if notifyErr != nil {
			return notifyErr
		}
		return backoffErr
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
	ticker := time.NewTicker(defaultTicker)
	for {
		select {
		case <-afterTimeout:
			return errors.New("timeout reached waiting for event")
		case <-ticker.C:
			for _, msg := range c.GetFakeCentral().GetAllMessages() {
				if fn(msg.GetEvent()) {
					return nil
				}
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

// GetLastAlertsWithDeploymentID find most recent alert message by deployment ID. If checkEmptyAlerts is set to true it will also check for AlertResults with no Alerts
func GetLastAlertsWithDeploymentID(messages []*central.MsgFromSensor, id string, checkEmptyAlertResults bool) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i >= 0; i-- {
		if checkEmptyAlertResults && messages[i].GetEvent().GetAlertResults().GetDeploymentId() == id && len(messages[i].GetEvent().GetAlertResults().GetAlerts()) == 0 {
			lastMessage = messages[i]
			break
		}
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

// RetryCallback callback function that will run if the creation of the resources fails.
type RetryCallback func(error, k8s.Object) error

// TestRunFunc options function for the testRun struct.
type TestRunFunc func(*testRun)

// testRun holds all the information about a specific test run. It requires a TestCallback
type testRun struct {
	resources     []K8sResourceInfo
	testCase      TestCallback
	retryCallback RetryCallback
	permutation   bool
}

func (t *testRun) validate() error {
	if t.testCase == nil {
		return errors.New("The testRun needs a TestCallback function")
	}
	return nil
}

func (t *testRun) copy() *testRun {
	newTestRun := &testRun{}
	newTestRun.resources = make([]K8sResourceInfo, len(t.resources))
	copy(newTestRun.resources, t.resources)
	newTestRun.testCase = t.testCase
	newTestRun.retryCallback = t.retryCallback
	newTestRun.permutation = t.permutation
	return newTestRun
}

func newTestRun(options ...TestRunFunc) (*testRun, error) {
	t := &testRun{
		resources:     nil,
		testCase:      nil,
		retryCallback: nil,
		permutation:   false,
	}
	for _, o := range options {
		o(t)
	}

	if err := t.validate(); err != nil {
		return nil, err
	}

	return t, nil
}
