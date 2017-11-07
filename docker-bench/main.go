package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/stack-rox/apollo/docker-bench/cis"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	apolloEndpointEnv = "ROX_APOLLO_ENDPOINT"
	tlsEnabledEnv     = "ROX_TLS"

	retries = 5
)

// GRPCConnection returns a grpc.ClientConn object.
func GRPCConnection(endpoint string, tlsEnabled bool) (conn *grpc.ClientConn, err error) {
	if tlsEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		creds := credentials.NewTLS(tlsConfig)
		return grpc.Dial(endpoint, grpc.WithTransportCredentials(creds))
	}
	return grpc.Dial(endpoint, grpc.WithInsecure())
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
		Results:   results,
		StartTime: protoStartTime,
		EndTime:   protoEndTime,
		Host:      os.Getenv("HOSTNAME"),
	}

	tlsEnabled := os.Getenv(tlsEnabledEnv) == "1"

	conn, err := GRPCConnection(ip, tlsEnabled)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := v1.NewBenchmarkServiceClient(conn)

	fmt.Printf("\n%+v\n", payload)

	for i := 1; i < retries+1; i++ {
		_, err := client.PostBenchmark(context.Background(), payload)
		if err == nil {
			break
		}
		log.Printf("Error posting benchmark to %v: %+v", ip, err)
		time.Sleep(time.Duration(i*2) * time.Second)
	}
}
