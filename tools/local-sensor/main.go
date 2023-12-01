package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // #nosec G108
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/centralclient"
	commonSensor "github.com/stackrox/rox/sensor/common/sensor"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/testutils"
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
	OutputFormat       string
	CentralOutput      string
	RecordK8sEnabled   bool
	RecordK8sFile      string
	ReplayK8sEnabled   bool
	ReplayK8sTraceFile string
	Verbose            bool
	ResyncPeriod       time.Duration
	CreateMode         k8s.CreateMode
	Delay              time.Duration
	PoliciesFile       string
	FakeWorkloadFile   string
	WithMetrics        bool
	NoCPUProfile       bool
	NoMemProfile       bool
	PprofServer        bool
	CentralEndpoint    string
}

const (
	jsonFormat string = "json"
	rawFormat  string = "raw"
)

func writeOutputInJSONFormat(messages []*central.MsgFromSensor, start, end time.Time, outfile string) {
	dateFormat := "02.01.15 11:06:39"
	data, err := json.Marshal(&sensorMessageJSONOutput{
		ScenarioStart:      start.Format(dateFormat),
		ScenarioEnd:        end.Format(dateFormat),
		MessagesFromSensor: messages,
	})
	utils.CrashOnError(err)
	utils.CrashOnError(os.WriteFile(outfile, data, 0644))
}

func writeOutputInBinaryFormat(messages []*central.MsgFromSensor, _, _ time.Time, outfile string) {
	file, err := os.OpenFile(outfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		utils.CrashOnError(file.Close())
	}()
	utils.CrashOnError(err)
	for _, m := range messages {
		d, err := m.Marshal()
		utils.CrashOnError(err)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(len(d)))
		_, err = file.Write(buf)
		utils.CrashOnError(err)
		_, err = file.Write(d)
		utils.CrashOnError(err)
	}
	if outfile != "/dev/null" {
		utils.CrashOnError(file.Sync())
	}
}

var validFormats = map[string]func([]*central.MsgFromSensor, time.Time, time.Time, string){
	jsonFormat: writeOutputInJSONFormat,
	rawFormat:  writeOutputInBinaryFormat,
}

func isValidOutputFormat(format string) bool {
	_, ok := validFormats[format]
	return ok
}

func mustGetCommandLineArgs() localSensorConfig {
	sensorConfig := localSensorConfig{
		Verbose:            false,
		Duration:           0,
		OutputFormat:       "json",
		CentralOutput:      "central-out.json",
		RecordK8sEnabled:   false,
		RecordK8sFile:      "k8s-trace.jsonl",
		ReplayK8sEnabled:   false,
		ReplayK8sTraceFile: "k8s-trace.jsonl",
		ResyncPeriod:       1 * time.Minute,
		Delay:              5 * time.Second,
		CreateMode:         k8s.Delay,
		PoliciesFile:       "",
		FakeWorkloadFile:   "",
		WithMetrics:        false,
		NoCPUProfile:       false,
		NoMemProfile:       false,
		PprofServer:        false,
		CentralEndpoint:    "",
	}
	flag.BoolVar(&sensorConfig.NoCPUProfile, "no-cpu-prof", sensorConfig.NoCPUProfile, "disables producing CPU profile for performance analysis")
	flag.BoolVar(&sensorConfig.NoMemProfile, "no-mem-prof", sensorConfig.NoMemProfile, "disables producing memory profile for performance analysis")

	flag.BoolVar(&sensorConfig.Verbose, "verbose", sensorConfig.Verbose, "prints all messages to stdout as well as to the output file")
	flag.DurationVar(&sensorConfig.Duration, "duration", sensorConfig.Duration, "duration that the scenario should run (leave it empty to run it without timeout)")
	flag.StringVar(&sensorConfig.CentralOutput, "central-out", sensorConfig.CentralOutput, "file to store the events that would be sent to central")
	flag.StringVar(&sensorConfig.OutputFormat, "format", sensorConfig.OutputFormat, "format of sensor's events file: 'raw' or 'json'")
	flag.BoolVar(&sensorConfig.RecordK8sEnabled, "record", sensorConfig.RecordK8sEnabled, "whether to record a trace with k8s events")
	flag.StringVar(&sensorConfig.RecordK8sFile, "record-out", sensorConfig.RecordK8sFile, "a file where recorded trace would be stored")
	flag.BoolVar(&sensorConfig.ReplayK8sEnabled, "replay", sensorConfig.ReplayK8sEnabled, "whether to reply recorded a trace with k8s events")
	flag.StringVar(&sensorConfig.ReplayK8sTraceFile, "replay-in", sensorConfig.ReplayK8sTraceFile, "a file where recorded trace would be read from")
	flag.DurationVar(&sensorConfig.ResyncPeriod, "resync", sensorConfig.ResyncPeriod, "resync period")
	flag.DurationVar(&sensorConfig.Delay, "delay", sensorConfig.Delay, "create events with a given delay")
	flag.StringVar(&sensorConfig.PoliciesFile, "with-policies", sensorConfig.PoliciesFile, " a file containing a list of policies")
	flag.StringVar(&sensorConfig.FakeWorkloadFile, "with-fakeworkload", sensorConfig.FakeWorkloadFile, " a file containing a FakeWorkload definition")
	flag.BoolVar(&sensorConfig.WithMetrics, "with-metrics", sensorConfig.WithMetrics, "enables the metric server")
	flag.BoolVar(&sensorConfig.PprofServer, "with-pprof-server", sensorConfig.PprofServer, "enables the pprof server on port :6060")
	flag.StringVar(&sensorConfig.CentralEndpoint, "connect-central", sensorConfig.CentralEndpoint, "connects to a Central instance rather than a fake Central")
	flag.Parse()

	sensorConfig.CentralOutput = path.Clean(sensorConfig.CentralOutput)

	if sensorConfig.ReplayK8sEnabled && sensorConfig.RecordK8sEnabled {
		log.Fatalf("cannot record and replay a trace at the same time. Use either -record or -replay flag")
	}
	if sensorConfig.ReplayK8sEnabled && sensorConfig.FakeWorkloadFile != "" {
		log.Fatalf("cannot replay a trace and use fake workloads at the same time. Use either -replay or -record -with-fakeworkload")
	}
	if sensorConfig.RecordK8sEnabled && sensorConfig.RecordK8sFile == "" {
		log.Printf("trace destination empty. Using default 'k8s-trace.jsonl'\n")
		sensorConfig.RecordK8sFile = "k8s-trace.jsonl"
	}
	sensorConfig.RecordK8sFile = path.Clean(sensorConfig.RecordK8sFile)
	if sensorConfig.ReplayK8sEnabled && sensorConfig.ReplayK8sTraceFile == "" {
		log.Fatalf("trace source empty")
	}

	if !isValidOutputFormat(sensorConfig.OutputFormat) {
		log.Fatalf("invalid format '%s'", sensorConfig.OutputFormat)
	}

	sensorConfig.ReplayK8sTraceFile = path.Clean(sensorConfig.ReplayK8sTraceFile)
	return sensorConfig
}

func writeMemoryProfile() {
	f, err := os.Create(fmt.Sprintf("local-sensor-mem-%s.prof", time.Now().UTC().Format(time.RFC3339)))
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer utils.IgnoreError(f.Close)
	runtime.GC()
	if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	log.Printf("Wrote memory profile")
}

func registerHostKillSignals(startTime time.Time, fakeCentral *centralDebug.FakeService, writeMemProfile bool, outfile string, outputFormat string, cancelFunc context.CancelFunc, sensor *commonSensor.Sensor) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	// We cancel the creation of Events
	cancelFunc()
	endTime := time.Now()
	if writeMemProfile {
		writeMemoryProfile()
	}
	sensor.Stop()
	pprof.StopCPUProfile()
	if fakeCentral != nil {
		allMessages := fakeCentral.GetAllMessages()
		dumpMessages(allMessages, startTime, endTime, outfile, outputFormat)
	}
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
	if localConfig.WithMetrics {
		// Start the prometheus metrics server
		metrics.NewServer(metrics.SensorSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
		metrics.GatherThrottleMetricsForever(metrics.SensorSubsystem.String())
	}
	var fakeClient client.Interface
	fakeClient, err := k8s.MakeOutOfClusterClient()
	// when replying a trace, there is no need to connect to K8s cluster
	if localConfig.ReplayK8sEnabled {
		fakeClient = k8s.MakeFakeClient()
	}
	var workloadManager *fake.WorkloadManager
	// if we are using a fake workload we don't want to connect to a real K8s cluster
	if localConfig.FakeWorkloadFile != "" {
		workloadManager = fake.NewWorkloadManager(fake.ConfigDefaults().
			WithWorkloadFile(localConfig.FakeWorkloadFile))
		fakeClient = workloadManager.Client()
	}
	utils.CrashOnError(err)
	if !localConfig.NoCPUProfile {
		f, err := os.Create(fmt.Sprintf("local-sensor-cpu-%s.prof", time.Now().UTC().Format(time.RFC3339)))
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer utils.IgnoreError(f.Close)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	if localConfig.PprofServer {
		go func() {
			log.Printf("Started pprof server in port :6060\n")
			err := http.ListenAndServe("localhost:6060", nil)
			if err != nil {
				log.Fatalf("%s\n", err)
			}
		}()
	}

	startTime := time.Now()

	isFakeCentral := localConfig.CentralEndpoint == ""

	var connection centralclient.CentralConnectionFactory
	var spyCentral *centralDebug.FakeService
	if isFakeCentral {
		connection, spyCentral = setupCentralWithFakeConnection(localConfig)
		defer spyCentral.Stop()
	} else {
		connection = setupCentralWithRealConnection(localConfig)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	sensorConfig := sensor.ConfigWithDefaults().
		WithK8sClient(fakeClient).
		WithCentralConnectionFactory(connection).
		WithLocalSensor(true).
		WithResyncPeriod(localConfig.ResyncPeriod).
		WithWorkloadManager(workloadManager)

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
			Delay:   localConfig.Delay,
			Mode:    localConfig.CreateMode,
			Client:  fakeClient,
			Reader:  trReader,
			Verbose: localConfig.Verbose,
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
	go registerHostKillSignals(startTime, spyCentral, !localConfig.NoMemProfile, localConfig.CentralOutput, localConfig.OutputFormat, cancelFunc, s)

	if spyCentral != nil {
		spyCentral.ConnectionStarted.Wait()
	}

	log.Printf("Running scenario for %f minutes\n", localConfig.Duration.Minutes())
	select {
	case <-time.Tick(localConfig.Duration):
		s.Stop()
		break
	case <-s.Stopped().Done():
		break
	}

	if spyCentral != nil {
		endTime := time.Now()
		allMessages := spyCentral.GetAllMessages()
		dumpMessages(allMessages, startTime, endTime, localConfig.CentralOutput, localConfig.OutputFormat)

		spyCentral.KillSwitch.Signal()
	}
}

func setupCentralWithRealConnection(localConfig localSensorConfig) centralclient.CentralConnectionFactory {
	// These files depend on running `tools/local-sensor/scripts/fetch-certs.sh`.
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "tmp/sensor-cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "tmp/sensor-key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "tmp/ca.pem"))

	utils.CrashOnError(os.Setenv("ROX_HELM_CONFIG_FILE_OVERRIDE", "tmp/helm-config.yaml"))
	utils.CrashOnError(os.Setenv("ROX_HELM_CLUSTER_NAME_FILE_OVERRIDE", "tmp/helm-name.yaml"))

	utils.CrashOnError(os.Setenv("ROX_CERTIFICATE_CACHE_DIR", "tmp/.local-sensor-cache"))

	utils.CrashOnError(os.Setenv("ROX_CENTRAL_ENDPOINT", localConfig.CentralEndpoint))

	clientconn.SetUserAgent(clientconn.Sensor)
	centralConnFactory, err := centralclient.NewCentralConnectionFactory(env.CentralEndpoint.Setting())
	if err != nil {
		utils.CrashOnError(errors.Wrapf(err, "sensor failed to start while initializing gRPC client to endpoint %s", env.CentralEndpoint.Setting()))
	}

	return centralConnFactory
}

func setupCentralWithFakeConnection(localConfig localSensorConfig) (centralclient.CentralConnectionFactory, *centralDebug.FakeService) {
	utils.CrashOnError(os.Setenv("ROX_MTLS_CERT_FILE", "tools/local-sensor/certs/cert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_KEY_FILE", "tools/local-sensor/certs/key.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_FILE", "tools/local-sensor/certs/caCert.pem"))
	utils.CrashOnError(os.Setenv("ROX_MTLS_CA_KEY_FILE", "tools/local-sensor/certs/caKey.pem"))

	var policies []*storage.Policy
	var err error
	if localConfig.PoliciesFile != "" {
		policies, err = testutils.GetPoliciesFromFile(localConfig.PoliciesFile)
		if err != nil {
			log.Fatalln(err)
		}
	}

	initialMessages := []*central.MsgToSensor{
		message.SensorHello("00000000-0000-4000-A000-000000000000"),
		message.ClusterConfig(),
		message.PolicySync(policies),
		message.BaselineSync([]*storage.ProcessBaseline{}),
		message.NetworkBaselineSync([]*storage.NetworkBaseline{}),
	}

	if features.SensorReconciliationOnReconnect.Enabled() {
		initialMessages = append(initialMessages, message.DeduperState(nil, 1, 1))
	}

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(initialMessages...)

	if localConfig.Verbose {
		fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
			log.Printf("MESSAGE RECEIVED: %s\n", msg.String())
		})
	}

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	fakeCentral.OnShutdown(shutdownFakeServer)
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	return fakeConnectionFactory, spyCentral
}

type sensorMessageJSONOutput struct {
	ScenarioStart      string                   `json:"scenario_start"`
	ScenarioEnd        string                   `json:"scenario_end"`
	MessagesFromSensor []*central.MsgFromSensor `json:"messages_from_sensor"`
}

func dumpMessages(messages []*central.MsgFromSensor, start, end time.Time, outfile string, outputFormat string) {
	log.Printf("Dumping all sensor messages to file: %s\n", outfile)
	f, ok := validFormats[outputFormat]
	if !ok {
		log.Fatalf("invalid format '%s'", outputFormat)
	}
	f(messages, start, end, outfile)
}
