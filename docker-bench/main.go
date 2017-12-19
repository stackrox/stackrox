package main

import (
	"fmt"
	"os"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/cis"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/docker"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
)

var (
	log = logging.New("docker-bench")
)

const (
	apolloEndpointEnv = "ROX_APOLLO_POST_ENDPOINT"
	retries           = 5
	requestTimeout    = 5 * time.Second
)

func main() {
	ip := os.Getenv(apolloEndpointEnv)
	if ip == "" {
		log.Fatalf("%v must be specified", apolloEndpointEnv)
	}

	hostname, err := getHostname()
	if err != nil {
		log.Fatalf("Could not find this node's hostname: %+v", err)
	}
	protoStartTime, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		log.Fatalf("Could not convert starting time to proto: %+v", err)
	}
	results := cis.RunCISBenchmark()
	protoEndTime, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		log.Fatalf("Could not convert ending time to proto: %+v", err)
	}
	payload := &v1.BenchmarkResult{
		Id:        uuid.NewV4().String(),
		Results:   results,
		StartTime: protoStartTime,
		EndTime:   protoEndTime,
		Host:      hostname,
		ScanId:    os.Getenv("ROX_APOLLO_SCAN_ID"),
	}
	conn, err := clientconn.GRPCConnection(ip)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := v1.NewBenchmarkRelayServiceClient(conn)

	fmt.Printf("%+v\n", payload)
	for i := 1; i < retries+1; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		_, err := client.PostBenchmarkResult(ctx, payload)
		if err == nil {
			return
		}
		log.Warnf("Error posting benchmark to %v: %+v", ip, err)
		log.Infof("Sleeping for %v before retrying", (time.Duration(i*2) * time.Second).Seconds())
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	log.Error("Timed out posting benchmark back to Apollo")
}

func getHostname() (string, error) {
	cli, err := docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("docker client setup: %s", err)
	}
	ctx, cancel := docker.TimeoutContext()
	defer cancel()
	info, err := cli.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("docker info: %s", err)
	}
	return info.Name, nil
}
