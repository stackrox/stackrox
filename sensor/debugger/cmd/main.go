package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
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

// Exporter works as an utility tool to export all events sent from sensor
// given a set of k8s events published.

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

func main() {
	var realCluster bool
	var fakeClient *k8s.ClientSet
	log.Println(os.Args)
	if len(os.Args) >= 2 && os.Args[1] == "k8s" {
		log.Printf("Using real k8s cluster")
		realCluster = true
		var err error
		fakeClient, err = k8s.MakeOutOfClusterClient()
		utils.CrashOnError(err)
	} else {
		log.Println("Creating fake k8s client")
		fakeClient = k8s.MakeFakeClient()
	}

	startTime := time.Now()
	os.Setenv("ROX_MTLS_CERT_FILE", "sensor/debugger/certs/cert.pem")
	os.Setenv("ROX_MTLS_KEY_FILE", "sensor/debugger/certs/key.pem")
	os.Setenv("ROX_MTLS_CA_FILE", "sensor/debugger/certs/caCert.pem")
	os.Setenv("ROX_MTLS_CA_KEY_FILE", "sensor/debugger/certs/caKey.pem")

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		log.Printf("MESSAGE SENT: %s\n", msg.String())
	})

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	defer shutdownFakeServer()
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	if !realCluster {
		utils.CrashOnError(fakeClient.SetupTestEnvironment())
		utils.CrashOnError(fakeClient.SetupNamespace("default"))
	}

	s, err := sensor.CreateSensor(fakeClient, nil, fakeConnectionFactory, true)
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	if !realCluster {
		for i := 0; i < 10; i++ {
			time.Sleep(20 * time.Second)
			depName := fmt.Sprintf("dep-%d", i)
			log.Printf("EXPORTER: creating deployment: %s\n", depName)
			utils.CrashOnError(fakeClient.SetupNginxDeployment(depName))
		}
	}



	time.Sleep(2 * time.Minute)
	endTime := time.Now()
	allMessages := fakeCentral.GetAllMessages()

	dateFormat := "02.01.15 11:06:39"
	dumpMessages(allMessages, startTime.Format(dateFormat), endTime.Format(dateFormat), "./outfile.json")

	spyCentral.KillSwitch.Signal()
}

type SensorMessagesOut struct {
	ScenarioStart string `json:"scenario_start"`
	ScenarioEnd string `json:"scenario_end"`
	MessagesFromSensor []*central.MsgFromSensor `json:"messages_from_sensor"`
}


func dumpMessages(messages []*central.MsgFromSensor, start, end, outfile string) {
	log.Printf("Dumping all sensor messages to file: %s\n", outfile)
	data, err := json.Marshal(&SensorMessagesOut{
		ScenarioStart:      start,
		ScenarioEnd:        end,
		MessagesFromSensor: messages,
	})
	utils.CrashOnError(err)
	utils.CrashOnError(ioutil.WriteFile(outfile, data, 0644))
}
