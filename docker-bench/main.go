package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/cis"
	"bitbucket.org/stack-rox/apollo/docker-bench/common"
)

const (
	apolloEndpoint = "ROX_APOLLO_ENDPOINT"
)

func main() {

	ip := os.Getenv(apolloEndpoint)
	if ip == "" {
		log.Fatalf("%v must be specified", apolloEndpoint)
	}

	startTime := time.Now().UnixNano()
	results := cis.RunCISBenchmark()

	payload := common.BenchmarkPayload{
		Results:   results,
		StartTime: startTime,
		EndTime:   time.Now().UnixNano(),
		Host:      os.Getenv("HOSTNAME"),
	}

	bytes, err := json.Marshal(&payload)
	if err != nil {
		log.Fatalf("Error marshalling benchmark payload: %+v", err)
	}
	fmt.Println(string(bytes))
}
