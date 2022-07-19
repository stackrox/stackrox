package resource

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

type YamlTestFile struct {
	Kind string
	File string
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
	default:
		log.Fatalf("unrecognized resource kind %s\n", kind)
		return nil
	}
}

type TestCallback func(t *testing.T, testContext *TestContext, objects map[string]k8s.Object)

type TestContext struct {
	t               *testing.T
	r               *resources.Resources
	env             *envconf.Config
	fakeCentral     *centralDebug.FakeService
	centralReceived chan *central.MsgFromSensor
	stopFn          func()
}

func NewContext(t *testing.T) (*TestContext, error) {
	envConfig := envconf.New().WithKubeconfigFile(conf.ResolveKubeConfigFile())
	r, err := resources.New(envConfig.Client().RESTConfig())
	if err != nil {
		return nil, err
	}
	fakeCentral, startFn, stopFn := startSensorAndFakeCentral(envConfig)
	ch := make(chan *central.MsgFromSensor, 100)
	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		ch <- msg
	})

	startFn()
	return &TestContext{
		t, r, envConfig, fakeCentral, ch, stopFn,
	}, nil
}

func (c *TestContext) Stop() {
	c.stopFn()
}

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

func (c *TestContext) GetFakeCentral() *centralDebug.FakeService {
	return c.fakeCentral
}

func (c *TestContext) RunWithResources(files []YamlTestFile, testCase TestCallback) {
	_, removeNamespace, err := createTestNs(context.Background(), c.r, "sensor-integration")
	defer utils.IgnoreError(removeNamespace)
	if err != nil {
		c.t.Fatalf("failed to create namespace: %s", err)
	}
	var removeFunctions []func() error
	fileToObj := map[string]k8s.Object{}
	for _, file := range files {
		obj := objByKind(file.Kind)
		removeFn, err := c.ApplyFile(context.Background(), "sensor-integration", file, obj)
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

func (c *TestContext) RunBare(name string, testCase TestCallback) {
	c.t.Run(name, func(t *testing.T) {
		_, removeNamespace, err := createTestNs(context.Background(), c.r, "sensor-integration")
		defer utils.IgnoreError(removeNamespace)
		if err != nil {
			t.Fatalf("failed to create namespace: %s", err)
		}
		testCase(t, c, nil)
	})
}

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

func startSensorAndFakeCentral(env *envconf.Config) (*centralDebug.FakeService, func(), func()) {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../../../tools/local-sensor/certs/caKey.pem"))

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
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

func (c *TestContext) ApplyFileNoObject(ctx context.Context, ns string, file YamlTestFile) (func() error, error) {
	obj := objByKind(file.Kind)
	return c.ApplyFile(ctx, ns, file, obj)
}

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

	if err := c.r.Create(ctx, obj); err != nil {
		return nil, err
	}

	if file.Kind == "Deployment" || file.Kind == "Pod" {
		if err := c.waitForResource(5*time.Second, deploymentName(obj.GetName())); err != nil {
			return nil, err
		}
	}

	return func() error {
		return c.r.Delete(ctx, obj)
	}, nil
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

func GetLastMessageWithDeploymentName(messages []*central.MsgFromSensor, ns, name string) *central.MsgFromSensor {
	var lastMessage *central.MsgFromSensor
	for i := len(messages) - 1; i > 0; i-- {
		deployment := messages[i].GetEvent().GetDeployment()
		if deployment.GetName() == name && deployment.GetNamespace() == ns {
			lastMessage = messages[i]
			break
		}
	}
	return lastMessage
}

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
