package agent

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
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
	ScannerPoller           *scanners.Client
	RegistryPoller          *registries.Client

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

		ClusterID:          env.ClusterID.Setting(),
		ApolloEndpoint:     env.ApolloEndpoint.Setting(),
		AdvertisedEndpoint: env.AdvertisedEndpoint.Setting(),
		Image:              env.Image.Setting(),

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
	if a.ScannerPoller != nil {
		go a.ScannerPoller.Start()
	}
	if a.RegistryPoller != nil {
		go a.RegistryPoller.Start()
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
	if a.ScannerPoller != nil {
		a.ScannerPoller.Stop()
	}
	if a.RegistryPoller != nil {
		a.RegistryPoller.Stop()
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
	a.enrichImages(ev)

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

func (a *Agent) enrichImages(ev *v1.DeploymentEvent) {
	for _, c := range ev.GetDeployment().GetContainers() {
		img := c.GetImage()
		for _, r := range a.RegistryPoller.Registries() {
			if r.Match(img) {
				meta, err := r.Metadata(img)
				if err != nil {
					a.Logger.Warnf("Couldn't get metadata for %v: %s", img, err)
				}
				img.Metadata = meta
			}
		}
		for _, s := range a.ScannerPoller.Scanners() {
			if s.Match(img) {
				scan, err := s.GetLastScan(img)
				if err != nil {
					a.Logger.Warnf("Couldn't get last scan for %v: %s", img, err)
				}
				img.Scan = scan
			}
		}
	}
}

func reportWithTimeout(cli v1.AgentEventServiceClient, ev *v1.DeploymentEvent) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = cli.ReportDeploymentEvent(ctx, ev)
	return
}
