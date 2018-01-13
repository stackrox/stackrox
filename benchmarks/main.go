package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	_ "bitbucket.org/stack-rox/apollo/pkg/checks"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/jsonpb"
	"golang.org/x/net/context"
)

var (
	log = logging.New("benchmarks")
)

const (
	retries        = 5
	requestTimeout = 5 * time.Second
)

func main() {
	var (
		jsonOnly bool
	)
	flag.BoolVar(&jsonOnly, "json", false, "specify --json will print the payload out in json format and not try to post it")
	flag.Parse()

	conn, err := clientconn.UnauthenticatedGRPCConnection(env.AdvertisedEndpoint.Setting())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := v1.NewBenchmarkRelayServiceClient(conn)

	benchmarkResult := RunBenchmark()
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
		defer cancel()
		_, err := client.PostBenchmarkResult(ctx, benchmarkResult)
		if err == nil {
			return
		}
		log.Warnf("Error posting benchmark to %v: %+v", env.AdvertisedEndpoint.Setting(), err)
		log.Infof("Sleeping for %v before retrying", (time.Duration(i*2) * time.Second).Seconds())
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	log.Error("Timed out posting benchmark back to Apollo")

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	if env.BenchmarkCompletion.Setting() == "true" {
		<-signalsC
	}
}
