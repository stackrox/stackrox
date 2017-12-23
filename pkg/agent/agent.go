package agent

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"github.com/golang/protobuf/ptypes/empty"
	googleGRPC "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Agent provides common structure and functionality for agents across various platforms.
type Agent struct {
	Server                  grpc.API
	Listener                listeners.Listener
	BenchScheduler          *benchmarks.SchedulerClient
	Orchestrator            orchestrators.Orchestrator
	ServiceRegistrationFunc func(a *Agent)

	ClusterID          string
	ApolloEndpoint     string
	AdvertisedEndpoint string
	Image              string

	Conn *googleGRPC.ClientConn

	Logger *logging.Logger
}

// New returns a new Agent.
func New() *Agent {
	return &Agent{
		Server: grpc.NewAPI(),

		ClusterID:          ClusterID.Setting(),
		ApolloEndpoint:     ApolloEndpoint.Setting(),
		AdvertisedEndpoint: AdvertisedEndpoint.Setting(),
		Image:              Image.Setting(),

		Logger: logging.New("main"),
	}
}

// Start starts all subroutines and the API server.
func (a *Agent) Start() {
	var err error
	a.Logger.Infof("Connecting to Apollo server %s", a.ApolloEndpoint)
	a.Conn, err = clientconn.GRPCConnection(a.ApolloEndpoint)
	if err != nil {
		panic(err)
	}

	if a.ServiceRegistrationFunc != nil {
		a.ServiceRegistrationFunc(a)
	}

	a.Server.Start()
	if a.Listener != nil {
		go a.Listener.Start()
	}
	if a.BenchScheduler != nil {
		go a.BenchScheduler.Start()
	}

	a.waitUntilApolloIsReady()
	go a.relayEvents()
}

// Stop stops the agent.
func (a *Agent) Stop() {
	if a.Listener != nil {
		a.Listener.Stop()
	}
	if a.BenchScheduler != nil {
		a.BenchScheduler.Stop()
	}

	a.Conn.Close()
}

func (a *Agent) relayEvents() {
	cli := v1.NewAgentEventServiceClient(a.Conn)

	for {
		select {
		case ev := <-a.Listener.Events():
			if err := a.reportDeploymentEvent(cli, ev); err != nil {
				a.Logger.Errorf("Couldn't report event %+v: %+v", ev, err)
			} else {
				a.Logger.Infof("Successfully reported event %+v", ev)
			}
		}
	}
}

func (a *Agent) waitUntilApolloIsReady() {
	pingService := v1.NewPingServiceClient(a.Conn)

	err := pingWithTimeout(pingService)
	for err != nil {
		a.Logger.Infof("Ping to Apollo failed: %s. Retrying...", err)
		time.Sleep(2 * time.Second)
		err = pingWithTimeout(pingService)
	}
}

func pingWithTimeout(svc v1.PingServiceClient) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = svc.Ping(ctx, &empty.Empty{})
	return
}

func (a *Agent) reportDeploymentEvent(cli v1.AgentEventServiceClient, ev *v1.DeploymentEvent) (err error) {
	retries := 0
	err = reportWithTimeout(cli, ev)
	errStatus, ok := status.FromError(err)

	for retries <= 5 && err != nil && ok && errStatus.Code() == codes.Unavailable {
		retries++
		time.Sleep(time.Duration(retries) * time.Second)
		err = reportWithTimeout(cli, ev)
		errStatus, ok = status.FromError(err)
	}

	return
}

func reportWithTimeout(cli v1.AgentEventServiceClient, ev *v1.DeploymentEvent) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = cli.ReportDeploymentEvent(ctx, ev)
	return
}
