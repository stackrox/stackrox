package sensor

import (
	"context"
	"reflect"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/enforcers"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/routes"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"bitbucket.org/stack-rox/apollo/pkg/orchestrators"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
	grpcLib "google.golang.org/grpc"
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
	ImageIntegrationPoller  *sources.Client

	DeploymentCache map[string]*v1.Deployment

	ClusterID          string
	CentralEndpoint    string
	AdvertisedEndpoint string
	Image              string

	Logger *logging.Logger

	eventLimiter *rate.Limiter

	Conn *grpcLib.ClientConn
}

// New returns a new Sensor.
func New(conn *grpcLib.ClientConn) *Sensor {
	return &Sensor{
		DeploymentCache: make(map[string]*v1.Deployment),

		ClusterID:          env.ClusterID.Setting(),
		CentralEndpoint:    env.CentralEndpoint.Setting(),
		AdvertisedEndpoint: env.AdvertisedEndpoint.Setting(),
		Image:              env.Image.Setting(),
		Conn:               conn,

		Logger: logging.NewOrGet("main"),

		eventLimiter: rate.NewLimiter(rate.Every(1*time.Second), 3),
	}
}

func (a *Sensor) customRoutes() map[string]routes.CustomRoute {
	routeMap := map[string]routes.CustomRoute{
		"/metrics": {
			ServerHandler: promhttp.Handler(),
			Compression:   false,
		},
	}
	return routeMap
}

// Start starts all subroutines and the API server.
func (a *Sensor) Start() {
	// Create grpc server with custom routes
	config := grpc.Config{
		TLS:          verifier.NonCA{},
		CustomRoutes: a.customRoutes(),
	}
	a.Server = grpc.NewAPI(config)

	a.Logger.Infof("Connecting to Central server %s", a.CentralEndpoint)
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
	if a.ImageIntegrationPoller != nil {
		go a.ImageIntegrationPoller.Start()
	}

	a.waitUntilCentralIsReady()
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
	if a.ImageIntegrationPoller != nil {
		a.ImageIntegrationPoller.Stop()
	}
}

func (a *Sensor) handleEvent(ev *listeners.DeploymentEventWrap) {

	a.eventLimiter.Wait(context.Background())

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
				AlertID:      resp.GetAlertId(),
			}
		}
	}
}

func (a *Sensor) relayEvents() {
	for {
		select {
		case ev := <-a.Listener.Events():
			if pastDeployment, ok := a.DeploymentCache[ev.Deployment.GetId()]; ok && ev.Action != v1.ResourceAction_REMOVE_RESOURCE {
				pastDeployment.UpdatedAt = ev.Deployment.GetUpdatedAt()
				pastDeployment.Version = ev.Deployment.GetVersion()
				if reflect.DeepEqual(pastDeployment, ev.Deployment) {
					a.Logger.Debugf("De-duping deployment '%s' ('%s') as there have been no tracked changes to the deployment", pastDeployment.GetId(), pastDeployment.GetName())
					continue
				}
			}
			if ev.Action == v1.ResourceAction_REMOVE_RESOURCE {
				delete(a.DeploymentCache, ev.Deployment.GetId())
			} else {
				a.DeploymentCache[ev.Deployment.GetId()] = ev.DeploymentEvent.Deployment
			}
			go a.handleEvent(ev)
		}
	}
}

func (a *Sensor) waitUntilCentralIsReady() {
	pingService := v1.NewPingServiceClient(a.Conn)
	err := pingWithTimeout(pingService)
	for err != nil {
		a.Logger.Infof("Ping to Central failed: %s. Retrying...", err)
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
	cli := v1.NewSensorEventServiceClient(a.Conn)

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
	// TODO(cgorman) can reuse code from central to implement this
	for _, c := range ev.GetDeployment().GetContainers() {
		img := c.GetImage()
		for _, integration := range a.ImageIntegrationPoller.Integrations() {
			registry := integration.Registry
			if registry != nil && registry.Match(img) {
				meta, err := registry.Metadata(img)
				if err != nil {
					a.Logger.Warnf("Couldn't get metadata for %v: %s", img, err)
				}
				img.Metadata = meta
			}
		}
		for _, integration := range a.ImageIntegrationPoller.Integrations() {
			scanner := integration.Scanner
			if scanner != nil && scanner.Match(img) {
				scan, err := scanner.GetLastScan(img)
				if err != nil {
					a.Logger.Warnf("Couldn't get metadata for %v: %s", img, err)
				}
				img.Scan = scan
			}
		}
	}
}

func (a *Sensor) reportWithTimeout(cli v1.SensorEventServiceClient, ev *v1.DeploymentEvent) (resp *v1.DeploymentEventResponse, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	resp, err = cli.ReportDeploymentEvent(ctx, ev)
	return
}
