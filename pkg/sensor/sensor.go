package sensor

import (
	"context"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/clientconn"
	"bitbucket.org/stack-rox/apollo/pkg/enforcers"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/features"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sensor provides common structure and functionality for sensors across various platforms.
type Sensor struct {
	Server                  grpc.API
	Listener                listeners.Listener
	Enforcer                enforcers.Enforcer
	BenchScheduler          *benchmarks.SchedulerClient
	Orchestrator            orchestrators.Orchestrator
	ServiceRegistrationFunc func(a *Sensor)
	ScannerPoller           *scanners.Client
	RegistryPoller          *registries.Client

	ClusterID          string
	ApolloEndpoint     string
	AdvertisedEndpoint string
	Image              string

	Logger *logging.Logger
}

// New returns a new Sensor.
func New() *Sensor {
	var server grpc.API
	if features.MTLS.Enabled() {
		server = grpc.NewAPI(grpc.Config{TLS: verifier.NonCA{}})
	} else {
		server = grpc.NewAPI(grpc.Config{TLS: verifier.NoMTLS{}})
	}
	return &Sensor{
		Server: server,

		ClusterID:          env.ClusterID.Setting(),
		ApolloEndpoint:     env.ApolloEndpoint.Setting(),
		AdvertisedEndpoint: env.AdvertisedEndpoint.Setting(),
		Image:              env.Image.Setting(),

		Logger: logging.New("main"),
	}
}

// Start starts all subroutines and the API server.
func (a *Sensor) Start() {
	a.Logger.Infof("Connecting to Apollo server %s", a.ApolloEndpoint)
	if a.ServiceRegistrationFunc != nil {
		a.ServiceRegistrationFunc(a)
	}

	a.Server.Start()
	if a.Listener != nil {
		go a.Listener.Start()
	}
	if a.Enforcer != nil {
		go a.Enforcer.Start()
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

// Stop stops the sensor.
func (a *Sensor) Stop() {
	if a.Listener != nil {
		a.Listener.Stop()
	}
	if a.Enforcer != nil {
		a.Enforcer.Stop()
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
}

func (a *Sensor) relayEvents() {
	for {
		select {
		case ev := <-a.Listener.Events():
			if resp, err := a.reportDeploymentEvent(ev.DeploymentEvent); err != nil {
				a.Logger.Errorf("Couldn't report event %s for deployment %s: %+v", ev.GetAction(), ev.GetDeployment().GetName(), err)
			} else {
				a.Logger.Infof("Successfully reported event %s for deployment %s", ev.GetAction(), ev.GetDeployment().GetName())
				if resp.GetEnforcement() != v1.EnforcementAction_UNSET_ENFORCEMENT {
					a.Logger.Infof("Event requested enforcement %s for deployment %s", resp.GetEnforcement(), ev.GetDeployment().GetName())
					a.Enforcer.Actions() <- &enforcers.DeploymentEnforcement{
						Deployment:   ev.GetDeployment(),
						OriginalSpec: ev.OriginalSpec,
						Enforcement:  resp.GetEnforcement(),
					}
				}
			}
		}
	}
}

func (a *Sensor) waitUntilApolloIsReady() {
	conn, err := clientconn.GRPCConnection(a.ApolloEndpoint)
	if err != nil {
		a.Logger.Fatal(err)
	}
	pingService := v1.NewPingServiceClient(conn)
	err = pingWithTimeout(pingService)
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

func (a *Sensor) reportDeploymentEvent(ev *v1.DeploymentEvent) (resp *v1.DeploymentEventResponse, err error) {
	conn, err := clientconn.GRPCConnection(a.ApolloEndpoint)
	if err != nil {
		return nil, err
	}
	cli := v1.NewSensorEventServiceClient(conn)

	a.enrichImages(ev)

	retries := 0
	resp, err = a.reportWithTimeout(cli, ev)
	errStatus, ok := status.FromError(err)

	for retries <= 5 && err != nil && ok && errStatus.Code() == codes.Unavailable {
		retries++
		time.Sleep(time.Duration(retries) * time.Second)
		resp, err = a.reportWithTimeout(cli, ev)
		errStatus, ok = status.FromError(err)
	}

	return
}

func (a *Sensor) enrichImages(ev *v1.DeploymentEvent) {
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

func (a *Sensor) reportWithTimeout(cli v1.SensorEventServiceClient, ev *v1.DeploymentEvent) (resp *v1.DeploymentEventResponse, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err = cli.ReportDeploymentEvent(ctx, ev)
	return
}
