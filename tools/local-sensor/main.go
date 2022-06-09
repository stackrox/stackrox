package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
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
	Duration            time.Duration
	CentralOutput       string
	ShallRecordK8sInput bool
	RecordK8sFile       string
	ShallReplayK8sTrace bool
	ReplayK8sTraceFile  string
	Verbose             bool
	ResyncPeriod        time.Duration
	CreateMode          k8s.CreateMode
	Delay               time.Duration
}

func mustGetCommandLineArgs() localSensorConfig {
	fsc := localSensorConfig{}
	verboseFlag := flag.Bool("verbose", false, "prints all messages to stdout as well as to the output file")

	durationFlag := flag.Duration("duration", 0, "duration that the scenario should run (leave it empty to run it without timeout)")
	centralOutputFile := flag.String("central-out", "central-out.json", "file to store the events that would be sent to central")

	recordTrace := flag.Bool("record", false, "whether to record a trace with k8s events")
	traceOutFile := flag.String("record-out", "k8s-trace.jsonl", "a file where recorded trace would be stored")
	replayTrace := flag.Bool("replay", false, "whether to reply recorded a trace with k8s events")
	traceInFile := flag.String("replay-in", "k8s-trace.jsonl", "a file where recorded trace would be read from")
	resyncPeriod := flag.Duration("resync", 1*time.Minute, "resync period")
	delay := flag.Duration("delay", 10*time.Second, "create events with a given delay")
	flag.Parse()

	fsc.Duration = *durationFlag
	fsc.CentralOutput = path.Clean(*centralOutputFile)

	if *recordTrace && *replayTrace {
		log.Fatalf("cannot record and replay a trace at the same time. Use either -record or -replay flag")
	}
	fsc.ShallRecordK8sInput = *recordTrace
	if *recordTrace && *traceOutFile == "" {
		log.Printf("trace destination empty. Using default 'k8s-trace.jsonl'\n")
		*traceOutFile = "k8s-trace.jsonl"
	}
	fsc.RecordK8sFile = path.Clean(*traceOutFile)
	if *replayTrace && *traceInFile == "" {
		log.Fatalf("trace source empty")
	}

	fsc.ShallReplayK8sTrace = *replayTrace
	fsc.ReplayK8sTraceFile = path.Clean(*traceInFile)
	fsc.Verbose = *verboseFlag
	fsc.ResyncPeriod = *resyncPeriod
	fsc.Delay = *delay
	fsc.CreateMode = k8s.Delay
	return fsc
}

func registerHostKillSignals(startTime time.Time, fakeCentral *centralDebug.FakeService, outfile string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
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
	sensorConfig := mustGetCommandLineArgs()
	fakeClient, err := k8s.MakeOutOfClusterClient()
	// when replying a trace, there is no need to connect to K8s cluster
	if sensorConfig.ShallReplayK8sTrace {
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

	if sensorConfig.Verbose {
		fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
			log.Printf("MESSAGE RECEIVED: %s\n", msg.String())
		})
	}

	go registerHostKillSignals(startTime, fakeCentral, sensorConfig.CentralOutput)

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	defer shutdownFakeServer()
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	traceRec := &k8s.TraceWriter{
		Destination: path.Clean(sensorConfig.RecordK8sFile),
		Enabled:     sensorConfig.ShallRecordK8sInput,
	}

	trReader := &k8s.TraceReader{
		Source:  path.Clean(sensorConfig.ReplayK8sTraceFile),
		Enabled: sensorConfig.ShallReplayK8sTrace,
	}

	if err := trReader.Init(); err != nil {
		log.Fatalln(err)
	}

	if trReader.Enabled {
		fm := k8s.FakeEventsManager{
			Delay:  sensorConfig.Delay,
			Mode:   sensorConfig.CreateMode,
			Client: fakeClient,
			Reader: trReader,
		}
		if err := fm.CreateEvents(); err != nil {
			log.Fatalln(err)
		}
	}

	s, err := sensor.CreateSensor(sensor.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithCentralConnectionFactory(fakeConnectionFactory).
		WithLocalSensor(true).
		WithResyncPeriod(sensorConfig.ResyncPeriod).WithTraceWriter(traceRec))
	if err != nil {
		panic(err)
	}

	go s.Start()
	defer s.Stop()

	spyCentral.ConnectionStarted.Wait()

	log.Printf("Running scenario for %f minutes\n", sensorConfig.Duration.Minutes())
	<-time.Tick(sensorConfig.Duration)
	endTime := time.Now()
	allMessages := fakeCentral.GetAllMessages()
	dumpMessages(allMessages, startTime, endTime, sensorConfig.CentralOutput)

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
