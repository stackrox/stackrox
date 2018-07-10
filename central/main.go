package main

import (
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	// These imports are required to register things from the respective packages.
	_ "bitbucket.org/stack-rox/apollo/pkg/authproviders/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/all"

	alertService "bitbucket.org/stack-rox/apollo/central/alert/service"
	authService "bitbucket.org/stack-rox/apollo/central/auth/service"
	authproviderService "bitbucket.org/stack-rox/apollo/central/authprovider/service"
	benchmarkService "bitbucket.org/stack-rox/apollo/central/benchmark/service"
	brService "bitbucket.org/stack-rox/apollo/central/benchmarkresult/service"
	bsService "bitbucket.org/stack-rox/apollo/central/benchmarkscan/service"
	bshService "bitbucket.org/stack-rox/apollo/central/benchmarkschedule/service"
	btService "bitbucket.org/stack-rox/apollo/central/benchmarktrigger/service"
	clusterService "bitbucket.org/stack-rox/apollo/central/cluster/service"
	clustersZip "bitbucket.org/stack-rox/apollo/central/clusters/zip"
	deploymentService "bitbucket.org/stack-rox/apollo/central/deployment/service"
	detectionSingletons "bitbucket.org/stack-rox/apollo/central/detection/singletons"
	dnrIntegrationService "bitbucket.org/stack-rox/apollo/central/dnrintegration/service"
	"bitbucket.org/stack-rox/apollo/central/globaldb"
	globaldbSingletons "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	imageService "bitbucket.org/stack-rox/apollo/central/image/service"
	iiService "bitbucket.org/stack-rox/apollo/central/imageintegration/service"
	interceptorSingletons "bitbucket.org/stack-rox/apollo/central/interceptor/singletons"
	logimbueHandler "bitbucket.org/stack-rox/apollo/central/logimbue/handler"
	metadataService "bitbucket.org/stack-rox/apollo/central/metadata/service"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	notifierService "bitbucket.org/stack-rox/apollo/central/notifier/service"
	pingService "bitbucket.org/stack-rox/apollo/central/ping/service"
	policyService "bitbucket.org/stack-rox/apollo/central/policy/service"
	searchService "bitbucket.org/stack-rox/apollo/central/search/service"
	secretService "bitbucket.org/stack-rox/apollo/central/secret/service"
	seService "bitbucket.org/stack-rox/apollo/central/sensorevent/service"
	siService "bitbucket.org/stack-rox/apollo/central/serviceidentities/service"
	summaryService "bitbucket.org/stack-rox/apollo/central/summary/service"
	pkgGRPC "bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	authzUser "bitbucket.org/stack-rox/apollo/pkg/grpc/authz/user"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/routes"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/mtls/verifier"
	"bitbucket.org/stack-rox/apollo/pkg/ui"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	central := newCentral()

	go central.startGRPCServer()

	central.processForever()
}

type central struct {
	signalsC chan os.Signal
	server   pkgGRPC.API
}

func newCentral() *central {
	central := &central{}

	central.signalsC = make(chan os.Signal, 1)
	signal.Notify(central.signalsC, os.Interrupt)
	signal.Notify(central.signalsC, syscall.SIGINT, syscall.SIGTERM)

	return central
}

func (c *central) startGRPCServer() {
	config := pkgGRPC.Config{
		CustomRoutes:       c.customRoutes(),
		TLS:                verifier.CA{},
		UnaryInterceptors:  interceptorSingletons.GrpcUnaryInterceptors(),
		StreamInterceptors: interceptorSingletons.GrpsStreamInterceptors(),
	}

	c.server = pkgGRPC.NewAPI(config)

	c.server.Register(alertService.Singleton())
	c.server.Register(authService.Singleton())
	c.server.Register(authproviderService.Singleton())
	c.server.Register(benchmarkService.Singleton())
	c.server.Register(bsService.Singleton())
	c.server.Register(bshService.Singleton())
	c.server.Register(brService.Singleton())
	c.server.Register(btService.Singleton())
	c.server.Register(clusterService.Singleton())
	c.server.Register(deploymentService.Singleton())
	c.server.Register(dnrIntegrationService.Singleton())
	c.server.Register(imageService.Singleton())
	c.server.Register(iiService.Singleton())
	c.server.Register(metadataService.New())
	c.server.Register(notifierService.Singleton())
	c.server.Register(pingService.Singleton())
	c.server.Register(policyService.Singleton())
	c.server.Register(searchService.Singleton())
	c.server.Register(secretService.Singleton())
	c.server.Register(siService.Singleton())
	c.server.Register(seService.Singleton())
	c.server.Register(summaryService.Singleton())

	c.server.Start()
}

func (c *central) customRoutes() (routeMap map[string]routes.CustomRoute) {
	routeMap = map[string]routes.CustomRoute{
		"/": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   ui.Mux(),
			Compression:     true,
		},
		"/api/extensions/clusters/zip": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   clustersZip.Handler(clusterService.Singleton(), siService.Singleton()),
			Compression:     false,
		},
		"/api/logimbue": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   logimbueHandler.Singleton(),
			Compression:     false,
		},
		"/db/backup": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   globaldb.BackupHandler(globaldbSingletons.GetGlobalDB()),
			Compression:     true,
		},
		"/db/export": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.Any(),
			ServerHandler:   globaldb.ExportHandler(globaldbSingletons.GetGlobalDB()),
			Compression:     true,
		},
		"/metrics": {
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   promhttp.Handler(),
			Compression:     false,
		},
	}

	c.addDebugRoutes(routeMap)

	return
}

func (c *central) addDebugRoutes(routeMap map[string]routes.CustomRoute) {
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
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
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
			detectionSingletons.GetDetector().Stop()
			globaldbSingletons.Close()
			log.Infof("Central terminated")
			return
		}
	}
}
