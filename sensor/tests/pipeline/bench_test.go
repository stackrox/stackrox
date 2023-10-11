package pipeline

import (
	"context"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	v1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	v12 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	fakeClient  *k8s.ClientSet
	fakeCentral *centralDebug.FakeService

	setupOnce sync.Once
)

var (
	serviceAccounts []string
	appNames        []string
)

func init() {
	rand.Seed(time.Now().UnixNano())

	serviceAccounts = make([]string, 1000)
	for i := 0; i < 1000; i++ {
		serviceAccounts[i] = randString(5)
	}

	appNames = make([]string, 1000)
	for i := 0; i < 1000; i++ {
		appNames[i] = randString(5)
	}
}

func randomSA() string {
	return serviceAccounts[rand.Intn(len(serviceAccounts))]
}

func randomAppName() map[string]string {
	return map[string]string{
		"app": appNames[rand.Intn(len(appNames))],
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func Benchmark_Pipeline(b *testing.B) {
	b.StopTimer()

	setupOnce.Do(func() {
		fakeClient = k8s.MakeFakeClient()

		b.Setenv("ROX_MTLS_CERT_FILE", "../../../tools/local-sensor/certs/cert.pem")
		b.Setenv("ROX_MTLS_KEY_FILE", "../../../tools/local-sensor/certs/key.pem")
		b.Setenv("ROX_MTLS_CA_FILE", "../../../tools/local-sensor/certs/caCert.pem")
		b.Setenv("ROX_MTLS_CA_KEY_FILE", "../../../tools/local-sensor/certs/caKey.pem")

		fakeCentral = centralDebug.MakeFakeCentralWithInitialMessages(
			message.SensorHello("00000000-0000-4000-A000-000000000000"),
			message.ClusterConfig(),
			message.PolicySync([]*storage.Policy{}),
			message.BaselineSync([]*storage.ProcessBaseline{}),
			message.NetworkBaselineSync([]*storage.NetworkBaseline{}))

		// No resync, we just want to test how long it takes for messages to go through the pipeline
		setupSensor(fakeCentral, fakeClient, 0)
	})

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		testNamespace := randString(10)
		_, err := fakeClient.Kubernetes().CoreV1().Namespaces().Create(context.Background(), &core.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}, metav1.CreateOptions{})

		require.NoError(b, err)
		sig := concurrency.NewSignal()
		fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
			if msg.GetEvent().GetDeployment() == nil {
				return
			}
			deployment := msg.GetEvent().GetDeployment()
			if deployment.GetNamespace() == testNamespace && deployment.GetName() == "FINAL" {
				sig.Signal()
			}
		})

		deletion := map[string][]string{
			"deployment": {},
			"role":       {},
			"binding":    {},
			"service":    {},
		}

		for i := 0; i < 100; i++ {
			roleName := randString(10)
			createRole(fakeClient, testNamespace, roleName)
			bindingName := randString(10)
			createBinding(fakeClient, testNamespace, bindingName, roleName)
			deletion["role"] = append(deletion["role"], roleName)
			deletion["binding"] = append(deletion["binding"], bindingName)
		}

		for i := 0; i < 100; i++ {
			serviceName := randString(10)
			createService(fakeClient, testNamespace, serviceName)
			deletion["service"] = append(deletion["service"], serviceName)
		}

		for i := 0; i < 1000; i++ {
			deploymentName := randString(10)
			createDeployment(fakeClient, testNamespace, deploymentName, appNames[i], serviceAccounts[i])
			deletion["deployment"] = append(deletion["deployment"], deploymentName)
		}

		// Wait until last deployment is seen by central
		createDeployment(fakeClient, testNamespace, "FINAL", "", "")
		deletion["deployment"] = append(deletion["deployment"], "FINAL")
		sig.Wait()

		b.StopTimer()
		// cleanup without the timer running
		// We need to delete so sensor data stores don't keep deployment objects lingering, which
		// can cause subsequent runs of the benchmark to yield incorrect results.
		// If this isn't done benchmark results can have a high discrepancy in results (> 10%)
		wg := sync.WaitGroup{}
		wg.Add(4)

		go utils.CrashOnError(deleteAll(fakeClient, "deployment", testNamespace, deletion["deployment"], &wg))
		go utils.CrashOnError(deleteAll(fakeClient, "role", testNamespace, deletion["role"], &wg))
		go utils.CrashOnError(deleteAll(fakeClient, "binding", testNamespace, deletion["binding"], &wg))
		go utils.CrashOnError(deleteAll(fakeClient, "service", testNamespace, deletion["service"], &wg))

		wg.Wait()
		err = fakeClient.Kubernetes().CoreV1().Namespaces().Delete(context.Background(), testNamespace, metav1.DeleteOptions{})
		require.NoError(b, err)
		b.StartTimer()
	}
}

func deleteAll(fakeClient *k8s.ClientSet, kind, namespace string, names []string, wg *sync.WaitGroup) error {
	defer wg.Done()
	for _, name := range names {
		switch kind {
		case "deployment":
			if err := fakeClient.Kubernetes().AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		case "role":
			if err := fakeClient.Kubernetes().RbacV1().Roles(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		case "binding":
			if err := fakeClient.Kubernetes().RbacV1().RoleBindings(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		case "service":
			if err := fakeClient.Kubernetes().CoreV1().Services(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func createDeployment(fakeClient *k8s.ClientSet, namespace, name, appName, serviceAccount string) *v1.Deployment {
	obj, err := fakeClient.Kubernetes().AppsV1().Deployments(namespace).Create(context.Background(),
		&v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app": appName,
				},
			},
			Spec: v1.DeploymentSpec{
				Template: core.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: core.PodSpec{
						Containers: []core.Container{
							{
								Name:  "foo",
								Image: "bar",
							},
						},
						ServiceAccountName: serviceAccount,
					},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return obj
}

func createRole(fakeClient *k8s.ClientSet, namespace, name string) *v12.Role {
	obj, err := fakeClient.Kubernetes().RbacV1().Roles(namespace).Create(context.Background(),
		&v12.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Rules: []v12.PolicyRule{
				{
					Verbs:     []string{"list"},
					APIGroups: []string{"deployments"},
				},
			},
		}, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return obj
}

func createBinding(fakeClient *k8s.ClientSet, namespace, name, ref string) *v12.RoleBinding {
	obj, err := fakeClient.Kubernetes().RbacV1().RoleBindings(namespace).Create(context.Background(),
		&v12.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Subjects: []v12.Subject{
				{
					Kind: "ServiceAccount",
					Name: randomSA(),
				},
			},
			RoleRef: v12.RoleRef{
				Kind: "Role",
				Name: ref,
			},
		}, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return obj
}

func createService(fakeClient *k8s.ClientSet, namespace, name string) *core.Service {
	obj, err := fakeClient.Kubernetes().CoreV1().Services(namespace).Create(context.Background(),
		&core.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: core.ServiceSpec{
				Ports: []core.ServicePort{
					{
						Name:       "some.service",
						Protocol:   "TCP",
						Port:       80,
						TargetPort: intstr.IntOrString{IntVal: 8080},
					},
				},
				Selector: randomAppName(),
				Type:     "LoadBalancer",
			},
		}, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return obj
}

func setupSensor(fakeCentral *centralDebug.FakeService, fakeClient *k8s.ClientSet, resyncTime time.Duration) {
	conn, spyCentral, _ := createConnectionAndStartServer(fakeCentral)
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithLocalSensor(true).
		WithResyncPeriod(resyncTime).
		WithCentralConnectionFactory(fakeConnectionFactory))

	if err != nil {
		panic(err)
	}

	go s.Start()

	spyCentral.ConnectionStarted.Wait()
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
