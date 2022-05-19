package integration_tests

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)


type fakeService struct{
	stream central.SensorService_CommunicateServer
	connectionStarted concurrency.Signal
}

func (s *fakeService) Communicate(msg central.SensorService_CommunicateServer) error {
	fmt.Println("@ Received Communicate call")
	md := metautils.NiceMD{}
	md.Set(centralsensor.SensorHelloMetadataKey, "true")
	err := msg.SetHeader(metadata.MD(md))
	if err != nil {
		return err
	}

	go func () {
		if received, err := msg.Recv(); err != nil {
			fmt.Printf("Error: %s\n", err)
		} else {
			fmt.Printf("MESSAGE RECEIVED FROM CENTRAL: %s\n", received.String())
		}
	}()

	// s.stream = msg
	return nil
}

func createConnectionAndStartServer(t *testing.T) (*grpc.ClientConn, *fakeService, func()) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)


	server := grpc.NewServer()
	fakeCentral := &fakeService{
		connectionStarted: concurrency.NewSignal(),
	}
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

func Test_Sensor(t *testing.T) {
	isolator := envisolator.NewEnvIsolator(t)
	defer isolator.RestoreAll()

	isolator.Setenv("ROX_MTLS_CERT_FILE", "certs/cert.pem")
	isolator.Setenv("ROX_MTLS_KEY_FILE", "certs/key.pem")
	isolator.Setenv("ROX_MTLS_CA_FILE", "certs/caCert.pem")
	isolator.Setenv("ROX_MTLS_CA_KEY_FILE", "certs/caKey.pem")

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(t)
	defer shutdownFakeServer()
	fakeClient := makeFakeClient()
	fakeConnectionFactory := makeFakeConnectionFactory(conn)

	fakeClient.setupTestEnvironment(t)

	s, err := sensor.CreateSensor(fakeClient, nil, fakeConnectionFactory, true)
	require.NoError(t, err)

	fmt.Println("Starting sensor")
	go s.Start()
	go func() {
		if b := fakeConnectionFactory.okSig.Signal(); !b {
			t.Fatal("Couldn't send signal to okSig from connection factory")
		}
	}()
	spyCentral.connectionStarted.Wait()
	fmt.Println("Received first communication")
	time.Sleep(20 * time.Second)
	//msg, err := spyCentral.stream.Recv()
	//require.NoError(t, err)
	//require.Equal(t, msg.String(), "asd")
	s.Stop()
}
