package tests

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

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

func Test_DeploymentInconsistent(t *testing.T) {
	fakeClient := k8s.MakeFakeClient()

	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem"))

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	defer shutdownFakeServer()
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	s, err := sensor.CreateSensor(fakeClient, nil, fakeConnectionFactory, true)
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	testCases := map[string]struct {
		orderedEvents  []func(t *testing.T, k *k8s.ClientSet)
		deploymentName string
	}{
		// This should not work if re-sync is disabled, because deployment will be
		// sent to central before RBAC is computed fully.
		"Deployment first": {
			orderedEvents: []func(t *testing.T, k *k8s.ClientSet){
				CreateDeployment("dep1", "sa1"),
				CreateRole("r1"),
				CreateRoleBinding("b1", "r1", "sa1"),
			},
			deploymentName: "dep1",
		},
		// This should also not work, if just the role or just the bindings is
		// available, the correct permission level won't be determined correctly.
		"Deployment second": {
			orderedEvents: []func(t *testing.T, k *k8s.ClientSet){
				CreateRole("r2"),
				CreateDeployment("dep2", "sa2"),
				CreateRoleBinding("b2", "r2", "sa2"),
			},
			deploymentName: "dep2",
		},
		// This is the only case that should work if re-sync isn't enabled.
		"Deployment last": {
			orderedEvents: []func(t *testing.T, k *k8s.ClientSet){
				CreateRole("r3"),
				CreateRoleBinding("b3", "r3", "sa3"),
				CreateDeployment("dep3", "sa3"),
			},
			deploymentName: "dep3",
		},
	}

	fakeClient.SetupExampleCluster(t)
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			// Execute each update in order on a clear namespace
			fakeClient.ResetDeployments(t)
			for _, fn := range testCase.orderedEvents {
				// This function will create the fake k8s event and wait for one second.
				// This way we guarantee some level of order for the events. Since sensor
				// processes events in different goroutines, some events take more time to
				// be processed than others. Therefore, even if we send two different events
				// at a particular order, e.g. Deployment and role respectively, it might be
				// that Role finishes first and is sent to Central before deployment gets to
				// the point of checking for permission levels.
				// To force what would happen in conditions where deployment is fully processed
				// before a Role is received, a timer was introduced here as well.
				// TODO: This should be done by checking the received event rather than arbitrarily waiting.
				fn(t, fakeClient)
				time.Sleep(1 * time.Second)
			}
			// Give some time for re-sync to happen (K8s client doesn't allow resync-time to be less than 1s)
			time.Sleep(5 * time.Second)
			allEvents := fakeCentral.GetAllMessages()
			log.Println("EVENTS RECEIVED (in order):")
			for pos, event := range allEvents {
				fmt.Printf("\t%d: %v\n", pos, event)
			}
			eventsFound := getAllDeploymentEventsWithName(allEvents, testCase.deploymentName)
			require.Greater(t, len(eventsFound), 0)
			// Expect last event to have correct permission level
			lastEvent := eventsFound[len(eventsFound)-1]
			assert.Equal(t, storage.PermissionLevel_ELEVATED_IN_NAMESPACE, lastEvent.GetDeployment().GetServiceAccountPermissionLevel())
		})
	}

}

func getAllDeploymentEventsWithName(messages []*central.MsgFromSensor, name string) []*central.SensorEvent {
	var events []*central.SensorEvent
	for _, msg := range messages {
		event := msg.GetEvent()
		if event.GetDeployment() != nil && event.GetDeployment().GetName() == name {
			events = append(events, event)
		}
	}
	return events
}

func CreateRole(id string) func(t *testing.T, k *k8s.ClientSet) {
	return func(t *testing.T, k *k8s.ClientSet) {
		k.MustCreateRole(t, id)
	}
}

func CreateRoleBinding(bindingId, roldId, serviceAccount string) func(t *testing.T, k *k8s.ClientSet) {
	return func(t *testing.T, k *k8s.ClientSet) {
		k.MustCreateRoleBinding(t, bindingId, roldId, serviceAccount)
	}
}

func CreateDeployment(depName, serviceAccount string) func(t *testing.T, k *k8s.ClientSet) {
	return func(t *testing.T, k *k8s.ClientSet) {
		k.MustCreateDeployment(t, depName, k8s.WithServiceAccountName(serviceAccount))
	}
}
