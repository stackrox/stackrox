package run

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/common/clusterid"
	commonSensor "github.com/stackrox/rox/sensor/common/sensor"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/certs"
	"github.com/stackrox/rox/sensor/debugger/collector"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stackrox/rox/sensor/debugger/message"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
	"github.com/stackrox/rox/sensor/kubernetes/sensor"
	"github.com/stackrox/rox/sensor/testutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Handle is the live in-process sensor session returned by Run.
type Handle struct {
	Stop            func()
	MetricsURL      string
	FakeCentral     *centralDebug.FakeService
	WaitInitialSync func(ctx context.Context) error
	Stopped         concurrency.ReadOnlyErrorSignal
}

// Run starts an in-process Sensor with fake or real Central. It does not wait for cfg.Duration;
// the caller owns timing and must call Handle.Stop when finished.
func Run(ctx context.Context, cfg Config) (*Handle, error) {
	if cfg.MetricsEnabled {
		if cfg.MetricsPort != "" {
			if err := os.Setenv(env.MetricsPort.EnvVar(), cfg.MetricsPort); err != nil {
				return nil, errors.Wrap(err, "setting metrics port")
			}
		}
		metrics.NewServer(metrics.SensorSubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
		metrics.GatherThrottleMetricsForever(metrics.SensorSubsystem.String())
	}

	runCtx, cancelRun := context.WithCancel(ctx)

	var k8sClient client.Interface
	if cfg.ReplayK8s {
		k8sClient = k8s.MakeFakeClient()
	}

	var (
		workloadManager *fake.WorkloadManager
		processPipeline sensor.ProcessPipelineHandle
	)

	if cfg.FakeWorkloadFile != "" {
		if _, err := os.Stat(cfg.FakeWorkloadFile); err != nil {
			if os.IsNotExist(err) {
				return nil, errors.Errorf("fake workload profile %q not found", cfg.FakeWorkloadFile)
			}
			return nil, errors.Wrapf(err, "unable to access fake workload profile %q", cfg.FakeWorkloadFile)
		}
		workloadManager = fake.NewWorkloadManager(fake.ConfigDefaults().
			WithWorkloadFile(cfg.FakeWorkloadFile))
		if workloadManager == nil {
			return nil, errors.Errorf("failed to initialize fake workload manager from workload profile %q", cfg.FakeWorkloadFile)
		}
		k8sClient = workloadManager.Client()
	}

	if k8sClient == nil {
		var err error
		k8sClient, err = k8s.MakeOutOfClusterClient()
		if err != nil {
			cancelRun()
			return nil, err
		}
	}

	if cfg.PprofServer {
		go func() {
			log.Printf("Started pprof server in port :6060\n")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				log.Fatalf("%s\n", err)
			}
		}()
	}

	startTime := time.Now()
	isFakeCentral := cfg.CentralEndpoint == ""

	var connection centralclient.CentralConnectionFactory
	var certLoader centralclient.CertLoader
	var spyCentral *centralDebug.FakeService
	clusterIDHandler := clusterid.NewHandler()

	if isFakeCentral {
		var err error
		connection, certLoader, spyCentral, err = setupCentralWithFakeConnection(cfg)
		if err != nil {
			cancelRun()
			return nil, err
		}
	} else {
		var err error
		connection, certLoader, err = setupCentralWithRealConnection(k8sClient, cfg)
		if err != nil {
			cancelRun()
			return nil, err
		}
	}

	if spyCentral != nil {
		spyCentral.SetMessageRecording(!cfg.SkipCentralOutput)
	}

	var traceCloser io.Closer
	sensorConfig := sensor.ConfigWithDefaults().
		WithClusterIDHandler(clusterIDHandler).
		WithK8sClient(k8sClient).
		WithCentralConnectionFactory(connection).
		WithCertLoader(certLoader).
		WithLocalSensor(true).
		WithWorkloadManager(workloadManager).
		WithProcessPipelineObserver(func(p sensor.ProcessPipelineHandle) {
			processPipeline = p
		})

	if !isFakeCentral {
		deploymentID := createDeploymentIdentificationWithNamespace(cfg.Namespace)
		sensorConfig = sensorConfig.WithDeploymentIdentification(deploymentID)
	}

	if cfg.FakeCollector {
		acceptAnyFn := func(ctx context.Context, _ string) (context.Context, error) {
			return ctx, nil
		}
		sensorConfig.WithSignalServiceAuthFuncOverride(acceptAnyFn).
			WithNetworkFlowServiceAuthFuncOverride(acceptAnyFn)
	}

	if cfg.RecordK8s {
		traceRec, err := k8s.NewTraceWriter(path.Clean(cfg.RecordK8sFile))
		if err != nil {
			cancelRun()
			return nil, err
		}
		traceCloser = traceRec
		sensorConfig.WithTraceWriter(traceRec)
	}

	if cfg.ReplayK8s {
		trReader := &k8s.TraceReader{
			Source: path.Clean(cfg.ReplayK8sFile),
		}
		if err := trReader.Init(); err != nil {
			cancelRun()
			return nil, err
		}

		fm := k8s.FakeEventsManager{
			Delay:   cfg.Delay,
			Mode:    k8s.Delay,
			Client:  k8sClient,
			Reader:  trReader,
			Verbose: cfg.Verbose,
		}
		min, doneSignal := fm.CreateEvents(runCtx)
		select {
		case <-doneSignal.Done():
			cancelRun()
			if err := doneSignal.Err(); err != nil {
				return nil, err
			}
			return nil, errors.New("the minimum resources to start sensor were not created")
		case <-min.WaitC():
		}
		go func() {
			select {
			case <-doneSignal.Done():
				if err := doneSignal.Err(); err != nil {
					cancelRun()
					log.Fatalln(err)
				}
			case <-runCtx.Done():
			}
		}()
	}

	s, err := sensor.CreateSensor(sensorConfig)
	if err != nil {
		cancelRun()
		return nil, err
	}

	go s.Start()

	if spyCentral != nil {
		spyCentral.ConnectionStarted.Wait()
	}

	if cfg.FakeCollector {
		fakeCollector := collector.NewFakeCollector(collector.WithDefaultConfig())
		if err := fakeCollector.Start(); err != nil {
			cancelRun()
			return nil, err
		}
	}

	stopOnce := false
	stopFn := func() {
		if stopOnce {
			return
		}
		stopOnce = true

		cancelRun()
		stopSensorAndWorkload(workloadManager, s, processPipeline)
		if spyCentral != nil {
			log.Printf("Stopping spyCentral")
			if !cfg.SkipCentralOutput {
				spyCentral.DumpAllMessages(startTime, time.Now(), cfg.CentralOutput, cfg.OutputFormat)
			}
			spyCentral.KillSwitch.Signal()
			spyCentral.Stop()
		}
		if traceCloser != nil {
			utils.IgnoreError(traceCloser.Close)
		}
	}

	return &Handle{
		Stop:            stopFn,
		MetricsURL:      metricsURL(cfg),
		FakeCentral:     spyCentral,
		WaitInitialSync: newWaitInitialSync(spyCentral),
		Stopped:         s.Stopped(),
	}, nil
}

func metricsURL(cfg Config) string {
	if !cfg.MetricsEnabled {
		return ""
	}
	port := cfg.MetricsPort
	if port == "" {
		port = env.MetricsPort.Setting()
	}
	if port == "" || port == "disabled" {
		return ""
	}
	hostPort := port
	if strings.HasPrefix(port, ":") {
		hostPort = "localhost" + port
	}
	return fmt.Sprintf("http://%s/metrics", hostPort)
}

func stopSensorAndWorkload(workloadManager *fake.WorkloadManager, sensor *commonSensor.Sensor, pipeline sensor.ProcessPipelineHandle) {
	if workloadManager != nil {
		workloadManager.Stop()
	}
	if sensor != nil {
		sensor.Stop()
	}
	if pipeline != nil {
		if err := pipeline.WaitForShutdown(); err != nil {
			log.Printf("warning: waiting for process pipeline shutdown failed: %v", err)
		}
	}
}

func createDeploymentIdentificationWithNamespace(namespace string) *storage.SensorDeploymentIdentification {
	return &storage.SensorDeploymentIdentification{
		AppNamespace: namespace,
	}
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

func setupCentralWithRealConnection(cli client.Interface, cfg Config) (centralclient.CentralConnectionFactory, centralclient.CertLoader, error) {
	certFetcherOpts := []certs.OptionFunc{
		certs.WithOutputDir("tmp/"),
		certs.WithNamespace(cfg.Namespace),
	}
	if cfg.OperatorInstall {
		certFetcherOpts = append(certFetcherOpts, certs.WithClusterName("", "", ""))
	}
	certFetcher := certs.NewCertificateFetcher(cli, certFetcherOpts...)
	if err := certFetcher.FetchCertificatesAndSetEnvironment(); err != nil {
		return nil, nil, errors.Wrap(err, "failed to retrieve sensor's certificates")
	}
	if err := os.Setenv("ROX_CERTIFICATE_CACHE_DIR", "tmp/.local-sensor-cache"); err != nil {
		return nil, nil, err
	}
	if err := os.Setenv("ROX_CENTRAL_ENDPOINT", cfg.CentralEndpoint); err != nil {
		return nil, nil, err
	}

	clientconn.SetUserAgent(clientconn.Sensor)

	centralClient, err := centralclient.NewClient(env.CentralEndpoint.Setting())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "sensor failed to start while initializing HTTP client to endpoint %s", env.CentralEndpoint.Setting())
	}
	centralConnFactory := centralclient.NewCentralConnectionFactory(centralClient)
	centralCertLoader := centralclient.RemoteCertLoader(centralClient)

	return centralConnFactory, centralCertLoader, nil
}

func setupCentralWithFakeConnection(cfg Config) (centralclient.CentralConnectionFactory, centralclient.CertLoader, *centralDebug.FakeService, error) {
	if err := os.Setenv("ROX_MTLS_CERT_FILE", "tools/local-sensor/certs/cert.pem"); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Setenv("ROX_MTLS_KEY_FILE", "tools/local-sensor/certs/key.pem"); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Setenv("ROX_MTLS_CA_FILE", "tools/local-sensor/certs/caCert.pem"); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Setenv("ROX_MTLS_CA_KEY_FILE", "tools/local-sensor/certs/caKey.pem"); err != nil {
		return nil, nil, nil, err
	}

	var policies []*storage.Policy
	if cfg.PoliciesFile != "" {
		var err error
		policies, err = testutils.GetPoliciesFromFile(cfg.PoliciesFile)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	initialMessages := []*central.MsgToSensor{
		message.SensorHello("00000000-0000-4000-A000-000000000000", string(centralsensor.VirtualMachinesSupported)),
		message.ClusterConfig(),
		message.PolicySync(policies),
		message.BaselineSync([]*storage.ProcessBaseline{}),
		message.NetworkBaselineSync([]*storage.NetworkBaseline{}),
		message.DeduperState(nil, 1, 1),
	}

	fakeCentral := centralDebug.MakeFakeCentralWithInitialMessages(initialMessages...)

	if cfg.Verbose {
		fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
			log.Printf("MESSAGE RECEIVED: %s\n", msg.String())
		})
	}

	conn, spyCentral, shutdownFakeServer := createConnectionAndStartServer(fakeCentral)
	fakeCentral.OnShutdown(shutdownFakeServer)
	fakeConnectionFactory := centralDebug.MakeFakeConnectionFactory(conn)

	return fakeConnectionFactory, centralclient.EmptyCertLoader(), spyCentral, nil
}
