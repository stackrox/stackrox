package main

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"golang.org/x/net/context"
)

var (
	log = logging.New("docker-bench")
)

const (
	retries        = 5
	requestTimeout = 5 * time.Second
)

func main() {
	conn, err := clientconn.GRPCConnection(env.AdvertisedEndpoint.Setting())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := v1.NewBenchmarkRelayServiceClient(conn)

	benchmarkResult := benchmarks.RunBenchmark()
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
}
