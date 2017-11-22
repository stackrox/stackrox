package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/cis"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	log = logging.New("docker-bench")
)

const (
	apolloEndpointEnv = "ROX_APOLLO_ENDPOINT"
	retries           = 5
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string) (conn *grpc.ClientConn, err error) {
	tlsConfig := &tls.Config{
		// TODO(cgorman) Proper cert management and TLS
		InsecureSkipVerify: true,
	}
	creds := credentials.NewTLS(tlsConfig)
	return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
}

func main() {
	ip := os.Getenv(apolloEndpointEnv)
	if ip == "" {
		log.Fatalf("%v must be specified", apolloEndpointEnv)
	}

	protoStartTime, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		log.Fatalf("Could not compute starting time: %+v", err)
	}
	results := cis.RunCISBenchmark()
	protoEndTime, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		log.Fatalf("Could not conver to proto ending time: %+v", err)
	}
	payload := &v1.BenchmarkPayload{
		Id:        uuid.NewV4().String(),
		Results:   results,
		StartTime: protoStartTime,
		EndTime:   protoEndTime,
		Host:      os.Getenv("HOSTNAME"),
	}
	conn, err := GRPCConnection(ip)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := v1.NewBenchmarkServiceClient(conn)

	fmt.Printf("%+v\n", payload)
	for i := 1; i < retries+1; i++ {
		_, err := client.PostBenchmarkResult(context.Background(), payload)
		if err == nil {
			return
		}
		log.Warnf("Error posting benchmark to %v: %+v", ip, err)
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	log.Error("Timed out posting benchmark back to Apollo")
}
