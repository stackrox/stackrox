package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/utils"
	centralDebug "github.com/stackrox/stackrox/sensor/debugger/central"
	"github.com/stackrox/stackrox/sensor/debugger/k8s"
	"github.com/stackrox/stackrox/sensor/debugger/message"
	"github.com/stackrox/stackrox/sensor/kubernetes/sensor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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

type localSensorConfig struct {
	Duration           time.Duration
	CentralOutput      string
	RecordK8sEnabled   bool
	RecordK8sFile      string
	ReplayK8sEnabled   bool
	ReplayK8sTraceFile string
	Verbose            bool
	ResyncPeriod       time.Duration
	CreateMode         k8s.CreateMode
	Delay              time.Duration
}

func mustGetCommandLineArgs() localSensorConfig {
	sensorConfig := localSensorConfig{
		Verbose:            false,
		Duration:           0,
		CentralOutput:      "central-out.json",
		RecordK8sEnabled:   false,
		RecordK8sFile:      "k8s-trace.jsonl",
		ReplayK8sEnabled:   false,
		ReplayK8sTraceFile: "k8s-trace.jsonl",
		ResyncPeriod:       1 * time.Minute,
		Delay:              5 * time.Second,
		CreateMode:         k8s.Delay,
	}
	flag.BoolVar(&sensorConfig.Verbose, "verbose", sensorConfig.Verbose, "prints all messages to stdout as well as to the output file")
	flag.DurationVar(&sensorConfig.Duration, "duration", sensorConfig.Duration, "duration that the scenario should run (leave it empty to run it without timeout)")
	flag.StringVar(&sensorConfig.CentralOutput, "central-out", sensorConfig.CentralOutput, "file to store the events that would be sent to central")
	flag.BoolVar(&sensorConfig.RecordK8sEnabled, "record", sensorConfig.RecordK8sEnabled, "whether to record a trace with k8s events")
	flag.StringVar(&sensorConfig.RecordK8sFile, "record-out", sensorConfig.RecordK8sFile, "a file where recorded trace would be stored")
	flag.BoolVar(&sensorConfig.ReplayK8sEnabled, "replay", sensorConfig.ReplayK8sEnabled, "whether to reply recorded a trace with k8s events")
	flag.StringVar(&sensorConfig.ReplayK8sTraceFile, "replay-in", sensorConfig.ReplayK8sTraceFile, "a file where recorded trace would be read from")
	flag.DurationVar(&sensorConfig.ResyncPeriod, "resync", sensorConfig.ResyncPeriod, "resync period")
	flag.DurationVar(&sensorConfig.Delay, "delay", sensorConfig.Delay, "create events with a given delay")
	flag.Parse()

	sensorConfig.CentralOutput = path.Clean(sensorConfig.CentralOutput)

	if sensorConfig.ReplayK8sEnabled && sensorConfig.RecordK8sEnabled {
		log.Fatalf("cannot record and replay a trace at the same time. Use either -record or -replay flag")
	}
	if sensorConfig.RecordK8sEnabled && sensorConfig.RecordK8sFile == "" {
		log.Printf("trace destination empty. Using default 'k8s-trace.jsonl'\n")
		sensorConfig.RecordK8sFile = "k8s-trace.jsonl"
	}
	sensorConfig.RecordK8sFile = path.Clean(sensorConfig.RecordK8sFile)
	if sensorConfig.ReplayK8sEnabled && sensorConfig.ReplayK8sTraceFile == "" {
		log.Fatalf("trace source empty")
	}

	sensorConfig.ReplayK8sTraceFile = path.Clean(sensorConfig.ReplayK8sTraceFile)
	return sensorConfig
}

func registerHostKillSignals(startTime time.Time, fakeCentral *centralDebug.FakeService, outfile string, cancelFunc context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	// We cancel the creation of Events
	cancelFunc()
	endTime := time.Now()
	allMessages := fakeCentral.GetAllMessages()
	dumpMessages(allMessages, startTime, endTime, outfile)
	os.Exit(0)
}

// local-sensor adds three new flags to sensor:
// -duration: specifies how long should the scenario run for (e.g. 10m)
// -output: once the scenario finishes (or gets killed) all messages sent to the fake central will be stored in this file.
// -verbose: other than storing messages to files, local-sensor will also send them to stdout
//
// If a KUBECONFIG file is provided, then local-sensor will use that file to connect to a remote cluster.
func main() {
	localConfig := mustGetCommandLineArgs()
	fakeClient, err := k8s.MakeOutOfClusterClient()
	// when replying a trace, there is no need to connect to K8s cluster
	if localConfig.ReplayK8sEnabled {
		fakeClient = k8s.MakeFakeClient()
	}
	utils.CrashOnError(err)

	startTime := time.Now()
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "tools/local-sensor/certs/caKey.pem"))

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(
		message.SensorHello("1234"),
		message.ClusterConfig(),
		message.PolicySync([]*storage.Policy{}),
		message.BaselineSync([]*storage.ProcessBaseline{}))

	if localConfig.Verbose {
		fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
			log.Printf("MESSAGE RECEIVED: %s\n", msg.String())
		})
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	go registerHostKillSignals(startTime, fakeCentral, localConfig.CentralOutput, cancelFunc)

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	defer shutdownFakeServer()
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	sensorConfig := sensor.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithCentralConnectionFactory(fakeConnectionFactory).
		WithLocalSensor(true).
		WithResyncPeriod(localConfig.ResyncPeriod)

	if localConfig.RecordK8sEnabled {
		traceRec := &k8s.TraceWriter{
			Destination: path.Clean(localConfig.RecordK8sFile),
		}
		if err := traceRec.Init(); err != nil {
			log.Fatalln(err)
		}
		sensorConfig.WithTraceWriter(traceRec)
	}

	if localConfig.ReplayK8sEnabled {
		trReader := &k8s.TraceReader{
			Source: path.Clean(localConfig.ReplayK8sTraceFile),
		}
		if err := trReader.Init(); err != nil {
			log.Fatalln(err)
		}

		fm := k8s.FakeEventsManager{
			Delay:  localConfig.Delay,
			Mode:   localConfig.CreateMode,
			Client: fakeClient,
			Reader: trReader,
		}
		min, errCh := fm.CreateEvents(ctx)
		select {
		case err := <-errCh:
			if err != nil {
				cancelFunc()
				log.Fatalln(err)
			}
			// If the errCh is closed but err == nil we know we are done creating resources,
			// but we did not reach the minimum resources to start sensor
			log.Fatalln(errors.New("the minimum resources to start sensor were not created"))
		case <-min.WaitC():
			break
		}
		// in case there are errors after we received the minimum resources signal
		go func() {
			for e := range errCh {
				cancelFunc()
				log.Fatalln(e)
			}
		}()
	}

	s, err := sensor.CreateSensor(sensorConfig)
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	log.Printf("Running scenario for %f minutes\n", localConfig.Duration.Minutes())
	<-time.Tick(localConfig.Duration)
	endTime := time.Now()
	allMessages := fakeCentral.GetAllMessages()
	dumpMessages(allMessages, startTime, endTime, localConfig.CentralOutput)

	spyCentral.KillSwitch.Signal()
}

type sensorMessagesJSONOutput struct {
	ScenarioStart      string                   `json:"scenario_start"`
	ScenarioEnd        string                   `json:"scenario_end"`
	MessagesFromSensor []*central.MsgFromSensor `json:"messages_from_sensor"`
}

func dumpMessages(messages []*central.MsgFromSensor, start, end time.Time, outfile string) {
	dateFormat := "02.01.15 11:06:39"
	log.Printf("Dumping all sensor messages to file: %s\n", outfile)
	data, err := json.Marshal(&sensorMessagesJSONOutput{
		ScenarioStart:      start.Format(dateFormat),
		ScenarioEnd:        end.Format(dateFormat),
		MessagesFromSensor: messages,
	})
	utils.CrashOnError(err)
	utils.CrashOnError(os.WriteFile(outfile, data, 0644))
}
