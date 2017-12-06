package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/stack-rox/apollo/agent/swarm/listener"
	"bitbucket.org/stack-rox/apollo/agent/swarm/orchestrator"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/scheduler"
)

var (
	logger = logging.New("main")
)

const (
	// ApolloEndpointEnvVar is consulted to determine where to look for the central Apollo server.
	ApolloEndpointEnvVar = "APOLLO_ENDPOINT"

	defaultApolloEndpoint = "apollo.stackrox:8080"
)

func apolloEndpoint() string {
	host := os.Getenv(ApolloEndpointEnvVar)
	if len(host) != 0 {
		return host
	}
	return defaultApolloEndpoint
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	listener, err := listener.New()
	if err != nil {
		panic(err)
	}

	endpoint := apolloEndpoint()
	logger.Infof("Connecting to Apollo server %s", endpoint)
	conn, err := clientconn.GRPCConnection(endpoint)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	cli := v1.NewAgentEventServiceClient(conn)

	go listener.Start()

	orch, err := orchestrator.New()
	if err != nil {
		panic(err)
	}
	bench := scheduler.NewBenchmarkSchedulerClient(orch, apolloEndpoint())
	go bench.Start()

	for {
		select {
		case ev := <-listener.Events():
			_, err := cli.ReportDeploymentEvent(context.Background(), ev)
			if err != nil {
				logger.Errorf("Couldn't report event %+v: %+v", ev, err)
			} else {
				logger.Infof("Successfully reported event %+v", ev)
			}
		case sig := <-sigs:
			logger.Infof("Caught %s signal", sig)
			listener.Stop()
			bench.Stop()
			logger.Infof("Agent terminated")
			return
		}
	}
}
