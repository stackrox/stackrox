package main

import (
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	_ "bitbucket.org/stack-rox/apollo/pkg/authproviders/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/all"

	clustersZip "bitbucket.org/stack-rox/apollo/central/clusters/zip"
	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/db/boltdb"
	"bitbucket.org/stack-rox/apollo/central/db/inmem"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/enrichment"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/notifications"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/central/search/blevesearch"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	pkgGRPC "bitbucket.org/stack-rox/apollo/pkg/grpc"
	authnUser "bitbucket.org/stack-rox/apollo/pkg/grpc/authn/user"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	authzUser "bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/clusters"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/routes"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"bitbucket.org/stack-rox/apollo/pkg/ui"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	central := newCentral()

	persistence, err := boltdb.NewWithDefaults(env.DBPath.Setting())
	if err != nil {
		panic(err)
	}
	database := inmem.New(persistence)

	indexer, err := blevesearch.NewIndexer()
	if err != nil {
		panic(err)
	}

	central.datastore, err = datastore.NewDataStore(database, indexer)
	if err != nil {
		panic(err)
	}

	central.notificationProcessor, err = notifications.NewNotificationProcessor(central.datastore)
	if err != nil {
		panic(err)
	}
	go central.notificationProcessor.Start()

	central.scorer = risk.NewScorer(central.datastore)
	if central.enricher, err = enrichment.New(central.datastore, central.scorer); err != nil {
		panic(err)
	}

	central.detector, err = detection.New(central.datastore, central.enricher, central.notificationProcessor)
	if err != nil {
		panic(err)
	}

	go central.startGRPCServer()

	central.processForever()
}

type central struct {
	signalsC              chan os.Signal
	detector              *detection.Detector
	enricher              *enrichment.Enricher
	notificationProcessor *notifications.Processor
	datastore             *datastore.DataStore
	server                pkgGRPC.API
	scorer                *risk.Scorer
}

func newCentral() *central {
	central := &central{}

	central.signalsC = make(chan os.Signal, 1)
	signal.Notify(central.signalsC, os.Interrupt)
	signal.Notify(central.signalsC, syscall.SIGINT, syscall.SIGTERM)

	return central
}

func (c *central) startGRPCServer() {
	idService := service.NewServiceIdentityService(c.datastore)
	clusterService := service.NewClusterService(c.datastore)
	clusterWatcher := clusters.NewClusterWatcher(c.datastore)
	userAuth := authnUser.NewAuthInterceptor(c.datastore)

	config := pkgGRPC.Config{
		CustomRoutes: c.customRoutes(userAuth, clusterService, idService),
		TLS:          verifier.CA{},
		UnaryInterceptors: []grpc.UnaryServerInterceptor{
			userAuth.UnaryInterceptor(),
			clusterWatcher.UnaryInterceptor(),
		},
		StreamInterceptors: []grpc.StreamServerInterceptor{
			userAuth.StreamInterceptor(),
			clusterWatcher.StreamInterceptor(),
		},
	}

	c.server = pkgGRPC.NewAPI(config)
	c.server.Register(service.NewAlertService(c.datastore))
	c.server.Register(service.NewAuthService())
	c.server.Register(service.NewAuthProviderService(c.datastore, userAuth))
	c.server.Register(service.NewBenchmarkService(c.datastore))
	c.server.Register(service.NewBenchmarkScansService(c.datastore))
	c.server.Register(service.NewBenchmarkScheduleService(c.datastore))
	c.server.Register(service.NewBenchmarkResultsService(c.datastore, c.notificationProcessor))
	c.server.Register(service.NewBenchmarkTriggerService(c.datastore))
	c.server.Register(clusterService)
	c.server.Register(service.NewDeploymentService(c.datastore, c.enricher))
	c.server.Register(service.NewImageService(c.datastore))
	c.server.Register(service.NewImageIntegrationService(c.datastore, c.detector))
	c.server.Register(service.NewNotifierService(c.datastore, c.notificationProcessor, c.detector))
	c.server.Register(service.NewPingService())
	c.server.Register(service.NewPolicyService(c.datastore, c.detector))
	c.server.Register(service.NewSearchService(c.datastore))
	c.server.Register(idService)
	c.server.Register(service.NewSensorEventService(c.detector, c.datastore, c.scorer))
	c.server.Register(service.NewSummaryService(c.datastore))
	c.server.Start()
}

func (c *central) customRoutes(userAuth *authnUser.AuthInterceptor, clusterService *service.ClusterService, idService *service.IdentityService) (routeMap map[string]routes.CustomRoute) {
	routeMap = map[string]routes.CustomRoute{
		"/": {
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   ui.Mux(),
			Compression:     true,
		},
		"/api/extensions/clusters/zip": {
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   clustersZip.Handler(clusterService, idService),
			Compression:     false,
		},
		"/db/backup": {
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   c.datastore.BackupHandler(),
			Compression:     true,
		},
		"/db/export": {
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   c.datastore.ExportHandler(),
			Compression:     true,
		},
		"/metrics": {
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   promhttp.Handler(),
			Compression:     false,
		},
	}

	c.addDebugRoutes(routeMap, userAuth)

	return
}

func (c *central) addDebugRoutes(routeMap map[string]routes.CustomRoute, userAuth *authnUser.AuthInterceptor) {
	rs := map[string]http.Handler{
		"/debug/pprof":         http.HandlerFunc(pprof.Index),
		"/debug/pprof/cmdline": http.HandlerFunc(pprof.Cmdline),
		"/debug/pprof/profile": http.HandlerFunc(pprof.Profile),
		"/debug/pprof/symbol":  http.HandlerFunc(pprof.Symbol),
		"/debug/pprof/trace":   http.HandlerFunc(pprof.Trace),
		"/debug/block":         pprof.Handler(`block`),
		"/debug/goroutine":     pprof.Handler(`goroutine`),
		"/debug/heap":          pprof.Handler(`heap`),
		"/debug/mutex":         pprof.Handler(`mutex`),
		"/debug/threadcreate":  pprof.Handler(`threadcreate`),
	}

	for r, h := range rs {
		routeMap[r] = routes.CustomRoute{
			AuthInterceptor: userAuth.HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   h,
			Compression:     true,
		}
	}
}

func (c *central) processForever() {
	defer func() {
		if r := recover(); r != nil {
			metrics.IncrementPanicCounter(getPanicFunc())
			log.Errorf("Caught panic in process loop; restarting. Stack: %s", string(debug.Stack()))
			c.processForever()
		}
	}()

	for {
		select {
		case sig := <-c.signalsC:
			log.Infof("Caught %s signal", sig)
			c.detector.Stop()
			c.datastore.Close()
			log.Infof("Central terminated")
			return
		}
	}
}
