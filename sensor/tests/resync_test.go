package tests

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/utils"
	centralDebug "github.com/stackrox/stackrox/sensor/debugger/central"
	"github.com/stackrox/stackrox/sensor/debugger/k8s"
	"github.com/stackrox/stackrox/sensor/debugger/message"
	"github.com/stackrox/stackrox/sensor/kubernetes/sensor"
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

type resourceCreationFunction func(*testing.T, *k8s.ClientSet, chan *central.MsgFromSensor)

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

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithLocalSensor(true).
		WithResyncPeriod(1 * time.Second).
		WithCentralConnectionFactory(fakeConnectionFactory))

	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	testCases := map[string]struct {
		orderedEvents  []resourceCreationFunction
		deploymentName string
	}{
		// This should not work if re-sync is disabled, because deployment will be
		// sent to central before RBAC is computed fully.
		"Deployment first": {
			orderedEvents: []resourceCreationFunction{
				createDeployment("dep1", "sa1"),
				createRole("r1"),
				createRoleBinding("b1", "r1", "sa1"),
			},
			deploymentName: "dep1",
		},
		// This should also not work, if just the role or just the bindings is
		// available, the correct permission level won't be determined correctly.
		"Deployment second": {
			orderedEvents: []resourceCreationFunction{
				createRole("r2"),
				createDeployment("dep2", "sa2"),
				createRoleBinding("b2", "r2", "sa2"),
			},
			deploymentName: "dep2",
		},
		// This is the only case that should work if re-sync isn't enabled.
		"Deployment last": {
			orderedEvents: []resourceCreationFunction{
				createRole("r3"),
				createRoleBinding("b3", "r3", "sa3"),
				createDeployment("dep3", "sa3"),
			},
			deploymentName: "dep3",
		},
	}

	fakeClient.SetupExampleCluster(t)
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			fakeCentral.ClearReceivedBuffer()
			receivedMessagesCh := make(chan *central.MsgFromSensor, 10)
			fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
				receivedMessagesCh <- msg
			})

			for _, fn := range testCase.orderedEvents {
				// This function will create the fake k8s event and wait for the event to be fully flushed through
				// sensor. This allows the tests to have a little more control in the order that events are being
				// sent. Since each event is processed separately, the order they are received, doesn't necessarily
				// guarantee that they will be fully processed first.
				fn(t, fakeClient, receivedMessagesCh)
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

func createRole(id string) resourceCreationFunction {
	return func(t *testing.T, k *k8s.ClientSet, received chan *central.MsgFromSensor) {
		k.MustCreateRole(t, id)
		require.NoError(t, waitForResource(received, "Role", 2*time.Second))
	}
}

func createRoleBinding(bindingID, roleID, serviceAccount string) resourceCreationFunction {
	return func(t *testing.T, k *k8s.ClientSet, received chan *central.MsgFromSensor) {
		k.MustCreateRoleBinding(t, bindingID, roleID, serviceAccount)
		require.NoError(t, waitForResource(received, "Binding", 2*time.Second))
	}
}

func createDeployment(depName, serviceAccount string) resourceCreationFunction {
	return func(t *testing.T, k *k8s.ClientSet, received chan *central.MsgFromSensor) {
		k.MustCreateDeployment(t, depName, k8s.WithServiceAccountName(serviceAccount))
		require.NoError(t, waitForResource(received, "Deployment", 2*time.Second))
	}
}

func waitForResource(received chan *central.MsgFromSensor, resource string, timeout time.Duration) error {
	afterTimeout := time.After(timeout)
	for {
		select {
		case <-afterTimeout:
			return errors.New("timeout reached waiting for event")
		case d, more := <-received:
			if !more {
				return errors.New("channel closed")
			}
			if d.GetEvent() != nil && d.GetEvent().GetTiming().GetResource() == resource {
				return nil
			}
		}
	}
}
