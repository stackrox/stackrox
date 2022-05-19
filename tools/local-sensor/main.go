package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// local-sensor is an application that allows you to run sensor in your host machine, while mocking a
// gRPC connection to central. This was introduced for testing and debugging purposes. At its current form,
// it does not connect to a real central, but instead it dumps all gRPC messages that would be sent to central in a file.

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

func mustGetCommandLineArgs() (time.Duration, string) {
	if len(os.Args) != 3 {
		fmt.Println("USAGE:")
		fmt.Println("  local-sensor <minutes> <output file path>")
		log.Fatalf("Incorrect number of arguments, expected 2 but found: %d", len(os.Args) - 1)
	}

	i, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("First parameter must be a valid integer")
	}

	return time.Duration(i) * time.Minute, path.Clean(os.Args[2])
}

func registerHostKillSignals(startTime time.Time, fakeCentral *centralDebug.FakeService, outfile string) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		endTime := time.Now()
		allMessages := fakeCentral.GetAllMessages()
		dumpMessages(allMessages, startTime, endTime, outfile)
		os.Exit(0)
	}()
}


// Args:
//   local-sensor <minutes> <output file path>
//
// local-sensor will run for <minutes> receiving events from a k8s cluster based on local environment.
// If a KUBECONFIG file is provided, then local-sensor will use that file to connect to a remote cluster.
// <output file path> specifies where should local-sensor store all the serialized messages sent to central.
// Outgoing messages will also show up in stdout
func main() {
	scenarioDuration, outfile := mustGetCommandLineArgs()

	fakeClient, err := k8s.MakeOutOfClusterClient()

	startTime := time.Now()
	os.Setenv("ROX_MTLS_CERT_FILE", "tools/local-sensor/certs/cert.pem")
	os.Setenv("ROX_MTLS_KEY_FILE", "tools/local-sensor/certs/key.pem")
	os.Setenv("ROX_MTLS_CA_FILE", "tools/local-sensor/certs/caCert.pem")
	os.Setenv("ROX_MTLS_CA_KEY_FILE", "tools/local-sensor/certs/caKey.pem")

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		// log.Printf("MESSAGE RECEIVED: %s\n", msg.String())
	})

	registerHostKillSignals(startTime, fakeCentral, outfile)

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

	log.Printf("Running scenario for %f minutes\n", scenarioDuration.Minutes())
	time.Sleep(scenarioDuration)
	endTime := time.Now()
	allMessages := fakeCentral.GetAllMessages()
	dumpMessages(allMessages, startTime, endTime, outfile)

	spyCentral.KillSwitch.Signal()
}

type SensorMessagesOut struct {
	ScenarioStart string `json:"scenario_start"`
	ScenarioEnd string `json:"scenario_end"`
	MessagesFromSensor []*central.MsgFromSensor `json:"messages_from_sensor"`
}

func dumpMessages(messages []*central.MsgFromSensor, start, end time.Time, outfile string) {
	dateFormat := "02.01.15 11:06:39"
	log.Printf("Dumping all sensor messages to file: %s\n", outfile)
	data, err := json.Marshal(&SensorMessagesOut{
		ScenarioStart:      start.Format(dateFormat),
		ScenarioEnd:        end.Format(dateFormat),
		MessagesFromSensor: messages,
	})
	utils.CrashOnError(err)
	utils.CrashOnError(ioutil.WriteFile(outfile, data, 0644))
}

