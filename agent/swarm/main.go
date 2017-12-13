package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitbucket.org/stack-rox/apollo/agent/swarm/benchmarks"
	"bitbucket.org/stack-rox/apollo/agent/swarm/listener"
	"bitbucket.org/stack-rox/apollo/agent/swarm/orchestrator"
	"bitbucket.org/stack-rox/apollo/agent/swarm/service"
	agentPkg "bitbucket.org/stack-rox/apollo/pkg/agent"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"bitbucket.org/stack-rox/apollo/pkg/scheduler"
	"github.com/golang/protobuf/ptypes/empty"
	googleGRPC "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type agent struct {
	server         grpc.API
	listener       listeners.Listener
	benchScheduler *scheduler.BenchmarkSchedulerClient
	orch           orchestrators.Orchestrator

	clusterID          string
	apolloEndpoint     string
	advertisedEndpoint string

	conn *googleGRPC.ClientConn

	logger *logging.Logger
}

func (a *agent) startGRPCServer() {
	a.server = grpc.NewAPI()
	a.server.Register(service.NewBenchmarkRelayService(benchmarks.NewLRURelayer(v1.NewBenchmarkResultsServiceClient(a.conn))))
	a.server.Start()
}

func newAgent() *agent {
	return &agent{
		clusterID:          agentPkg.ClusterID.Setting(),
		apolloEndpoint:     agentPkg.ApolloEndpoint.Setting(),
		advertisedEndpoint: agentPkg.AdvertisedEndpoint.Setting(),

		logger: logging.New("main"),
	}
}

func (a *agent) init() {
	var err error

	a.listener, err = listener.New()
	if err != nil {
		panic(err)
	}

	a.orch, err = orchestrator.New()
	if err != nil {
		panic(err)
	}

	a.benchScheduler = scheduler.NewBenchmarkSchedulerClient(a.orch, a.apolloEndpoint, a.advertisedEndpoint)
}

func (a *agent) start() {
	var err error
	a.logger.Infof("Connecting to Apollo server %s", a.apolloEndpoint)
	a.conn, err = clientconn.GRPCConnection(a.apolloEndpoint)
	if err != nil {
		panic(err)
	}

	a.startGRPCServer()
	go a.listener.Start()
	go a.benchScheduler.Start()

	a.waitUntilApolloIsReady()
	go a.relayEvents()
}

func (a *agent) stop() {
	a.listener.Stop()
	a.benchScheduler.Stop()

	a.conn.Close()
}

func (a *agent) relayEvents() {
	cli := v1.NewAgentEventServiceClient(a.conn)

	for {
		select {
		case ev := <-a.listener.Events():
			if err := a.reportDeploymentEvent(cli, ev); err != nil {
				a.logger.Errorf("Couldn't report event %+v: %+v", ev, err)
			} else {
				a.logger.Infof("Successfully reported event %+v", ev)
			}
		}
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	a := newAgent()
	a.init()
	a.start()

	for {
		select {
		case sig := <-sigs:
			a.logger.Infof("Caught %s signal", sig)
			a.stop()
			a.logger.Infof("Agent terminated")
			return
		}
	}
}

func (a *agent) waitUntilApolloIsReady() {
	pingService := v1.NewPingServiceClient(a.conn)

	_, err := pingService.Ping(context.Background(), &empty.Empty{})
	for err != nil {
		a.logger.Infof("Ping to Apollo failed: %s. Retrying...", err)
		time.Sleep(2 * time.Second)
		_, err = pingService.Ping(context.Background(), &empty.Empty{})
	}
}

func (a *agent) reportDeploymentEvent(cli v1.AgentEventServiceClient, ev *v1.DeploymentEvent) (err error) {
	retries := 0
	_, err = cli.ReportDeploymentEvent(context.Background(), ev)
	errStatus, ok := status.FromError(err)

	for retries <= 5 && err != nil && ok && errStatus.Code() == codes.Unavailable {
		retries++
		time.Sleep(time.Duration(retries) * time.Second)
		_, err = cli.ReportDeploymentEvent(context.Background(), ev)
		errStatus, ok = status.FromError(err)
	}

	return
}
