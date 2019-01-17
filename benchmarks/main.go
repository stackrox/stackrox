package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/jsonpb"
	_ "github.com/stackrox/rox/benchmarks/checks"
	_ "github.com/stackrox/rox/benchmarks/checks/all"
	"github.com/stackrox/rox/benchmarks/runner"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/net/context"
)

var (
	log = logging.LoggerForModule()
)

const (
	retries        = 5
	requestTimeout = 5 * time.Second
)

func runBenchmark(jsonOnly bool) {
	conn, err := clientconn.AuthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting(), clientconn.Sensor)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	cli := v1.NewBenchmarkResultsServiceClient(conn)

	benchmarkResult := runner.RunBenchmark()
	if jsonOnly {
		marshaler := jsonpb.Marshaler{}
		buf := new(bytes.Buffer)
		if err := marshaler.Marshal(buf, benchmarkResult); err != nil {
			log.Fatal(err)
		}
		fmt.Println(buf.String())
		return
	}
	fmt.Printf("%+v\n", benchmarkResult)
	for i := 1; i < retries+1; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		_, err := cli.PostBenchmarkResult(ctx, benchmarkResult)
		cancel()
		if err == nil {
			return
		}
		log.Warnf("Error posting benchmark to %v: %+v", env.AdvertisedEndpoint.Setting(), err)
		log.Infof("Sleeping for %v before retrying", (time.Duration(i*2) * time.Second).Seconds())
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	log.Error("Timed out posting benchmark back to Sensor")
}

func main() {
	var (
		jsonOnly bool
	)
	flag.BoolVar(&jsonOnly, "json", false, "specify --json will print the payload out in json format and not try to post it")
	flag.Parse()

	runBenchmark(jsonOnly)

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	if env.BenchmarkCompletion.Setting() == "true" {
		<-signalsC
	}
}
