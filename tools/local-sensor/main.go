package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/stackrox/rox/pkg/continuousprofiling"
	"github.com/stackrox/rox/pkg/utils"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
	"github.com/stackrox/rox/sensor/debugger/certs"
	"github.com/stackrox/rox/tools/local-sensor/run"
	"k8s.io/apimachinery/pkg/util/validation"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// local-sensor is an application that allows you to run sensor on your host machine for testing and
// debugging purposes. It can either connect to a real Central instance using the -connect-central flag,
// or use a fake Central that dumps all gRPC messages to a file.

type localSensorConfig struct {
	Duration           time.Duration
	OutputFormat       string
	CentralOutput      string
	SkipCentralOutput  bool
	RecordK8sEnabled   bool
	RecordK8sFile      string
	ReplayK8sEnabled   bool
	ReplayK8sTraceFile string
	Verbose            bool
	Delay              time.Duration
	PoliciesFile       string
	FakeWorkloadFile   string
	WithMetrics        bool
	NoCPUProfile       bool
	NoMemProfile       bool
	PprofServer        bool
	CentralEndpoint    string
	FakeCollector      bool
	Namespace          string
	OperatorInstall    bool
}

func mustGetCommandLineArgs() localSensorConfig {
	sensorConfig := localSensorConfig{
		Verbose:            false,
		Duration:           0,
		OutputFormat:       "json",
		CentralOutput:      "central-out.json",
		SkipCentralOutput:  false,
		RecordK8sEnabled:   false,
		RecordK8sFile:      "k8s-trace.jsonl",
		ReplayK8sEnabled:   false,
		ReplayK8sTraceFile: "k8s-trace.jsonl",
		Delay:              5 * time.Second,
		PoliciesFile:       "",
		FakeWorkloadFile:   "",
		WithMetrics:        false,
		NoCPUProfile:       false,
		NoMemProfile:       false,
		PprofServer:        false,
		CentralEndpoint:    "",
		FakeCollector:      false,
		Namespace:          certs.DefaultNamespace,
		OperatorInstall:    false,
	}
	flag.BoolVar(&sensorConfig.NoCPUProfile, "no-cpu-prof", sensorConfig.NoCPUProfile, "disables producing CPU profile for performance analysis")
	flag.BoolVar(&sensorConfig.NoMemProfile, "no-mem-prof", sensorConfig.NoMemProfile, "disables producing memory profile for performance analysis")

	flag.BoolVar(&sensorConfig.Verbose, "verbose", sensorConfig.Verbose, "prints all messages to stdout as well as to the output file")
	flag.DurationVar(&sensorConfig.Duration, "duration", sensorConfig.Duration, "duration that the scenario should run (leave it empty to run it without timeout)")
	flag.StringVar(&sensorConfig.CentralOutput, "central-out", sensorConfig.CentralOutput, "file to store the events that would be sent to central")
	flag.BoolVar(&sensorConfig.SkipCentralOutput, "skip-central-output", sensorConfig.SkipCentralOutput, "disables recording fake central messages and writing central output files")
	flag.StringVar(&sensorConfig.OutputFormat, "format", sensorConfig.OutputFormat, "format of sensor's events file: 'raw' or 'json'")
	flag.BoolVar(&sensorConfig.RecordK8sEnabled, "record", sensorConfig.RecordK8sEnabled, "whether to record a trace with k8s events")
	flag.StringVar(&sensorConfig.RecordK8sFile, "record-out", sensorConfig.RecordK8sFile, "a file where recorded trace would be stored")
	flag.BoolVar(&sensorConfig.ReplayK8sEnabled, "replay", sensorConfig.ReplayK8sEnabled, "whether to reply recorded a trace with k8s events")
	flag.StringVar(&sensorConfig.ReplayK8sTraceFile, "replay-in", sensorConfig.ReplayK8sTraceFile, "a file where recorded trace would be read from")
	flag.DurationVar(&sensorConfig.Delay, "delay", sensorConfig.Delay, "create events with a given delay")
	flag.StringVar(&sensorConfig.PoliciesFile, "with-policies", sensorConfig.PoliciesFile, " a file containing a list of policies")
	flag.StringVar(&sensorConfig.FakeWorkloadFile, "with-fakeworkload", sensorConfig.FakeWorkloadFile, " a file containing a FakeWorkload definition")
	flag.BoolVar(&sensorConfig.WithMetrics, "with-metrics", sensorConfig.WithMetrics, "enables the metric server")
	flag.BoolVar(&sensorConfig.PprofServer, "with-pprof-server", sensorConfig.PprofServer, "enables the pprof server on port :6060")
	flag.StringVar(&sensorConfig.CentralEndpoint, "connect-central", sensorConfig.CentralEndpoint, "connects to a Central instance rather than a fake Central")
	flag.StringVar(&sensorConfig.Namespace, "namespace", sensorConfig.Namespace, "namespace where sensor is deployed (used for certificate generation when connecting to real Central)")
	flag.BoolVar(&sensorConfig.FakeCollector, "with-fake-collector", sensorConfig.FakeCollector, "enables sensor to allow connections from a fake collector")
	flag.BoolVar(&sensorConfig.OperatorInstall, "operator-install", sensorConfig.OperatorInstall, "use together with connect-central, indicates that the remote ACS was installed with the Operator")
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

	if !centralDebug.IsValidOutputFormat(sensorConfig.OutputFormat) {
		log.Fatalf("invalid format '%s'", sensorConfig.OutputFormat)
	}

	if sensorConfig.CentralEndpoint != "" && sensorConfig.SkipCentralOutput {
		log.Fatalf("-skip-central-output cannot be used together with -connect-central")
	}

	if errs := validation.IsDNS1123Label(sensorConfig.Namespace); len(errs) > 0 {
		log.Fatalf("invalid namespace '%s': %s", sensorConfig.Namespace, errs[0])
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

func toRunConfig(localConfig localSensorConfig) run.Config {
	return run.Config{
		Duration:          localConfig.Duration,
		CentralEndpoint:   localConfig.CentralEndpoint,
		FakeWorkloadFile:  localConfig.FakeWorkloadFile,
		PoliciesFile:      localConfig.PoliciesFile,
		RecordK8s:         localConfig.RecordK8sEnabled,
		RecordK8sFile:     localConfig.RecordK8sFile,
		ReplayK8s:         localConfig.ReplayK8sEnabled,
		ReplayK8sFile:     localConfig.ReplayK8sTraceFile,
		Delay:             localConfig.Delay,
		Verbose:           localConfig.Verbose,
		MetricsEnabled:    localConfig.WithMetrics,
		SkipCentralOutput: localConfig.SkipCentralOutput,
		CentralOutput:     localConfig.CentralOutput,
		OutputFormat:      localConfig.OutputFormat,
		NoCPUProfile:      localConfig.NoCPUProfile,
		NoMemProfile:      localConfig.NoMemProfile,
		PprofServer:       localConfig.PprofServer,
		FakeCollector:     localConfig.FakeCollector,
		Namespace:         localConfig.Namespace,
		OperatorInstall:   localConfig.OperatorInstall,
	}
}

// local-sensor adds three new flags to sensor:
// -duration: specifies how long should the scenario run for (e.g. 10m)
// -output: once the scenario finishes (or gets killed) all messages sent to the fake central will be stored in this file.
// -verbose: other than storing messages to files, local-sensor will also send them to stdout
//
// If a KUBECONFIG file is provided, then local-sensor will use that file to connect to a remote cluster.
func main() {
	if err := continuousprofiling.SetupClient(continuousprofiling.DefaultConfig(),
		continuousprofiling.WithDefaultAppName("sensor")); err != nil {
		log.Printf("unable to start continuous profiling: %v", err)
	}

	localConfig := mustGetCommandLineArgs()
	cfg := toRunConfig(localConfig)

	if !cfg.NoCPUProfile {
		f, err := os.Create(fmt.Sprintf("local-sensor-cpu-%s.prof", time.Now().UTC().Format(time.RFC3339)))
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer utils.IgnoreError(f.Close)
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
	}

	handle, err := run.Run(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	signal.Ignore(syscall.SIGPIPE)

	log.Printf("Running scenario for %f minutes\n", cfg.Duration.Minutes())
	select {
	case <-time.Tick(cfg.Duration):
	case <-handle.Stopped.Done():
	case sig := <-sigCh:
		log.Printf("Received %s, starting graceful shutdown (press Ctrl-C again to force exit)", sig)
		go func() {
			forceSig := <-sigCh
			log.Printf("Received %s during graceful shutdown, exiting immediately", forceSig)
			os.Exit(130)
		}()
	}

	if !cfg.NoMemProfile {
		writeMemoryProfile()
	}
	log.Printf("Stopping sensor and workload manager...")
	handle.Stop()
	pprof.StopCPUProfile()
}
