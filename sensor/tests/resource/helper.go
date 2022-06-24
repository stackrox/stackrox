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
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	k8s2 "github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
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

type yamlTestFile struct {
	kind string
	file string
}

var (
	Nginx            = yamlTestFile{"Deployment", "nginx.yaml"}
	NginxRole        = yamlTestFile{"Role", "nginx-role.yaml"}
	NginxRoleBinding = yamlTestFile{"Binding", "nginx-binding.yaml"}
)

type TestContext struct {
	t               *testing.T
	r               *resources.Resources
	env             *envconf.Config
	fakeCentral     *centralDebug.FakeService
	centralReceived chan *central.MsgFromSensor
}

func NewContext(t *testing.T) (*TestContext, error) {
	envConfig := envconf.New().WithKubeconfigFile(conf.ResolveKubeConfigFile())
	r, err := resources.New(envConfig.Client().RESTConfig())
	if err != nil {
		return nil, err
	}
	fakeCentral := startSensorAndFakeCentral(envConfig)
	ch := make(chan *central.MsgFromSensor, 100)
	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		ch <- msg
	})
	return &TestContext{
		t, r, envConfig, fakeCentral, ch,
	}, nil
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
			fmt.Printf("failed to wait for namespace deletion")
		}
		return nil
	}, nil
}

func (c *TestContext) RunTest(files []yamlTestFile, name string, testCase func(*testing.T, *centralDebug.FakeService)) {
	c.t.Run(name, func(t *testing.T) {
		_, removeNamespace, err := createTestNs(context.Background(), c.r, "sensor-integration")
		defer utils.IgnoreError(removeNamespace)
		if err != nil {
			t.Fatalf("failed to create namespace: %s", err)
		}
		var removeFunctions []func() error
		for _, file := range files {
			removeFn, err := c.applyFileWithWait(context.Background(), "sensor-integration", file)
			if err != nil {
				t.Fatalf("fail to apply resource: %s", err)
			}
			removeFunctions = append(removeFunctions, removeFn)
		}
		defer func() {
			for _, fn := range removeFunctions {
				if err := fn(); err != nil {
					t.Fatalf("failed to remove resource: %s", err)
				}
			}
		}()
		testCase(t, c.fakeCentral)
	})
}


func (c *TestContext) RunPermutationTest(files []yamlTestFile, name string, testCase func(*testing.T, *centralDebug.FakeService)) {
	runPermutation(files, 0, func(f []yamlTestFile) {
		newF := make([]yamlTestFile, len(f))
		copy(newF, f)
		c.RunTest(newF, fmt.Sprintf("%s_Permutation_%s", name, permutationKind(newF)), testCase)
	})
}

func permutationKind(perm []yamlTestFile) string {
	kinds := make([]string, len(perm))
	for i, p := range perm {
		kinds[i] = p.kind
	}
	return strings.Join(kinds, "_")
}

func runPermutation(files []yamlTestFile, i int, cb func([]yamlTestFile)) {
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

func startSensorAndFakeCentral(env *envconf.Config) *centralDebug.FakeService {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../../tools/local-sensor/certs/caKey.pem"))

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	conn, spyCentral, _ := createConnectionAndStartServer(fakeCentral)
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(k8s2.MakeFakeClientFromRest(env.Client().RESTConfig())).
		WithLocalSensor(true).
		WithResyncPeriod(1 * time.Second).
		WithCentralConnectionFactory(fakeConnectionFactory))

	if err != nil {
		panic(err)
	}

	go s.Start()

	spyCentral.ConnectionStarted.Wait()

	return fakeCentral
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

func objByKind(kind string) k8s.Object {
	switch kind {
	case "Deployment":
		return &appsV1.Deployment{}
	case "Role":
		return &v12.Role{}
	case "Binding":
		return &v12.RoleBinding{}
	default:
		log.Fatalf("unrecognized resource kind %s\n", kind)
		return nil
	}
}

func (c *TestContext) applyFileWithWait(ctx context.Context, ns string, file yamlTestFile) (func() error, error) {
	d := os.DirFS("yaml")
	obj := objByKind(file.kind)
	if err := decoder.DecodeFile(
		d,
		file.file,
		obj,
		decoder.MutateNamespace(ns),
	); err != nil {
		return nil, err
	}

	if err := c.r.Create(ctx, obj); err != nil {
		return nil, err
	}

	var cond condition
	switch file.kind {
	case "Deployment":
		cond = deploymentName(obj.GetName())
	case "Role":
		cond = roleName(obj.GetName())
	case "Binding":
		cond = bindingName(obj.GetName())
	default:
		cond = func(event *central.SensorEvent) bool {
			// just don't wait if it's something else
			return true
		}
	}

	if err := c.waitForResource(2 * time.Second, cond); err != nil {
		return nil, err
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

func roleName(s string) condition {
	return func(event *central.SensorEvent) bool {
		return event.GetRole().GetName() == s
	}
}

func bindingName(s string) condition {
	return func(event *central.SensorEvent) bool {
		return event.GetBinding().GetName() == s
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
