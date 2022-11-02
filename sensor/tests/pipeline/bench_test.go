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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	fakeClient  *k8s.ClientSet
	fakeCentral *centralDebug.FakeService

	setupOnce sync.Once
)

func init() {
	rand.Seed(time.Now().UnixNano())
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
			message.BaselineSync([]*storage.ProcessBaseline{}))

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

		var objToDelete []*v1.Deployment
		for i := 0; i < 1000; i++ {
			obj := createDeployment(fakeClient, testNamespace, randString(10))
			objToDelete = append(objToDelete, obj)
		}

		// Wait until last deployment is seen by central
		obj := createDeployment(fakeClient, testNamespace, "FINAL")
		objToDelete = append(objToDelete, obj)
		sig.Wait()

		b.StopTimer()
		// cleanup without the timer running
		// We need to delete so sensor data stores don't keep deployment objects lingering, which
		// can cause subsequent runs of the benchmark to yield incorrect results.
		// If this isn't done benchmark results can have a high discrepancy in results (> 10%)
		for _, o := range objToDelete {
			err := fakeClient.Kubernetes().AppsV1().Deployments(testNamespace).Delete(context.Background(), o.GetName(), metav1.DeleteOptions{})
			require.NoError(b, err)
		}
		err = fakeClient.Kubernetes().CoreV1().Namespaces().Delete(context.Background(), testNamespace, metav1.DeleteOptions{})
		require.NoError(b, err)
		b.StartTimer()
	}
}

func createDeployment(fakeClient *k8s.ClientSet, namespace, name string) *v1.Deployment {
	obj, err := fakeClient.Kubernetes().AppsV1().Deployments(namespace).Create(context.Background(),
		&v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
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
					},
				},
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
