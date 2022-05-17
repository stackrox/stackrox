package tests

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/tests/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func createConnectionAndStartServer(t *testing.T, fakeCentral *fakeService) (*grpc.ClientConn, *fakeService, func()) {
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
	require.NoError(t, err)

	closeF := func() {
		utils.IgnoreError(listener.Close)
		server.Stop()
	}

	return conn, fakeCentral, closeF
}

func isHello(m *central.MsgFromSensor) bool {
	return m.GetHello() != nil
}

func atLeastOneMessage(t *testing.T, messages []*central.MsgFromSensor, match func(m *central.MsgFromSensor) bool, msg string) {
	for _, m := range messages {
		if match(m) {
			return
		}
	}
	t.Errorf("No matches: %s", msg)
}

func Test_Sensor(t *testing.T) {
	isolator := envisolator.NewEnvIsolator(t)
	defer isolator.RestoreAll()

	isolator.Setenv("ROX_MTLS_CERT_FILE", "certs/cert.pem")
	isolator.Setenv("ROX_MTLS_KEY_FILE", "certs/key.pem")
	isolator.Setenv("ROX_MTLS_CA_FILE", "certs/caCert.pem")
	isolator.Setenv("ROX_MTLS_CA_KEY_FILE", "certs/caKey.pem")

	fakeCentral := makeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(t, fakeCentral)
	defer shutdownFakeServer()
	fakeClient := makeFakeClient()
	fakeConnectionFactory := makeFakeConnectionFactory(conn)

	fakeClient.setupTestEnvironment(t)

	s, err := sensor.CreateSensor(fakeClient, nil, fakeConnectionFactory, true)
	require.NoError(t, err)

	go s.Start()
	defer s.Stop()

	spyCentral.connectionStarted.Wait()

	time.Sleep(15 * time.Second)
	allMessages := fakeCentral.GetAllMessages()
	assert.GreaterOrEqual(t, len(allMessages), 5)
	atLeastOneMessage(t, allMessages, isHello, "Message is Hello")

	spyCentral.killSwitch.Signal()
}
