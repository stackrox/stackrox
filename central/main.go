package main

import (
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	// These imports are required to register things from the respective packages.
	_ "bitbucket.org/stack-rox/apollo/pkg/auth/authproviders/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/all"
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/all"

	alertService "bitbucket.org/stack-rox/apollo/central/alert/service"
	apiTokenService "bitbucket.org/stack-rox/apollo/central/apitoken/service"
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
	"bitbucket.org/stack-rox/apollo/central/detection"
	dnrIntegrationService "bitbucket.org/stack-rox/apollo/central/dnrintegration/service"
	"bitbucket.org/stack-rox/apollo/central/globaldb"
	globaldbHandlers "bitbucket.org/stack-rox/apollo/central/globaldb/handlers"
	imageService "bitbucket.org/stack-rox/apollo/central/image/service"
	iiService "bitbucket.org/stack-rox/apollo/central/imageintegration/service"
	interceptorSingletons "bitbucket.org/stack-rox/apollo/central/interceptor/singletons"
	logimbueHandler "bitbucket.org/stack-rox/apollo/central/logimbue/handler"
	metadataService "bitbucket.org/stack-rox/apollo/central/metadata/service"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	networkPolicyService "bitbucket.org/stack-rox/apollo/central/networkpolicies/service"
	notifierService "bitbucket.org/stack-rox/apollo/central/notifier/service"
	pingService "bitbucket.org/stack-rox/apollo/central/ping/service"
	policyService "bitbucket.org/stack-rox/apollo/central/policy/service"
	"bitbucket.org/stack-rox/apollo/central/role/resources"
	searchService "bitbucket.org/stack-rox/apollo/central/search/service"
	secretService "bitbucket.org/stack-rox/apollo/central/secret/service"
	seService "bitbucket.org/stack-rox/apollo/central/sensorevent/service"
	siService "bitbucket.org/stack-rox/apollo/central/serviceidentities/service"
	summaryService "bitbucket.org/stack-rox/apollo/central/summary/service"
	"bitbucket.org/stack-rox/apollo/pkg/auth/permissions"
	pkgGRPC "bitbucket.org/stack-rox/apollo/pkg/grpc"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/allow"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/perrpc"
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

	c.server.Register(
		alertService.Singleton(),
		apiTokenService.Singleton(),
		authService.Singleton(),
		authproviderService.Singleton(),
		benchmarkService.Singleton(),
		bsService.Singleton(),
		bshService.Singleton(),
		brService.Singleton(),
		btService.Singleton(),
		clusterService.Singleton(),
		deploymentService.Singleton(),
		dnrIntegrationService.Singleton(),
		imageService.Singleton(),
		iiService.Singleton(),
		metadataService.New(),
		networkPolicyService.Singleton(),
		notifierService.Singleton(),
		pingService.Singleton(),
		policyService.Singleton(),
		searchService.Singleton(),
		secretService.Singleton(),
		seService.Singleton(),
		siService.Singleton(),
		summaryService.Singleton(),
	)

	c.server.Start()
}

// To export the DB, you need to be able to view _everything_.
// As new resource types are added, please ensure that you add
// them to this method.
// TODO(viswa): Figure out a way to make this easier or enforceable.
func dbExportOrBackupAuthorizer() authz.Authorizer {
	return authzUser.With(
		permissions.View(resources.APIToken),
		permissions.View(resources.Alert),
		permissions.View(resources.AuthProvider),
		permissions.View(resources.Benchmark),
		permissions.View(resources.BenchmarkScan),
		permissions.View(resources.BenchmarkSchedule),
		permissions.View(resources.BenchmarkTrigger),
		permissions.View(resources.Cluster),
		permissions.View(resources.DebugMetrics),
		permissions.View(resources.Deployment),
		permissions.View(resources.DNRIntegration),
		permissions.View(resources.Image),
		permissions.View(resources.ImageIntegration),
		permissions.View(resources.ImbuedLogs),
		permissions.View(resources.Notifier),
		permissions.View(resources.Policy),
		permissions.View(resources.Secret),
		permissions.View(resources.ServiceIdentity),
	)
}

func (c *central) customRoutes() (customRoutes []routes.CustomRoute) {
	customRoutes = []routes.CustomRoute{
		{
			Route:           "/",
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   ui.Mux(),
			Compression:     true,
		},
		{
			Route:           "/api/extensions/clusters/zip",
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.With(permissions.View(resources.Cluster), permissions.View(resources.ServiceIdentity)),
			ServerHandler:   clustersZip.Handler(clusterService.Singleton(), siService.Singleton()),
			Compression:     false,
		},

		{
			Route:           "/db/backup",
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      dbExportOrBackupAuthorizer(),
			ServerHandler:   globaldbHandlers.BackupDB(globaldb.GetGlobalDB()),
			Compression:     true,
		},
		{
			Route:           "/db/export",
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      dbExportOrBackupAuthorizer(),
			ServerHandler:   globaldbHandlers.ExportDB(globaldb.GetGlobalDB()),
			Compression:     true,
		},
		{
			Route:           "/metrics",
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      allow.Anonymous(),
			ServerHandler:   promhttp.Handler(),
			Compression:     false,
		},
	}

	logImbueRoute := "/api/logimbue"
	customRoutes = append(customRoutes,
		routes.CustomRoute{
			Route:           logImbueRoute,
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer: perrpc.FromMap(map[authz.Authorizer][]string{
				authzUser.With(permissions.View(resources.ImbuedLogs)): {
					routes.RPCNameForHTTP(logImbueRoute, http.MethodGet),
				},
				authzUser.With(permissions.Modify(resources.ImbuedLogs)): {
					routes.RPCNameForHTTP(logImbueRoute, http.MethodPost),
				},
			}),
			ServerHandler: logimbueHandler.Singleton(),
			Compression:   false,
		},
	)
	customRoutes = append(customRoutes, c.debugRoutes()...)
	return
}

func (c *central) debugRoutes() []routes.CustomRoute {
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

	customRoutes := make([]routes.CustomRoute, 0, len(rs))

	for r, h := range rs {
		customRoutes = append(customRoutes, routes.CustomRoute{
			Route:           r,
			AuthInterceptor: interceptorSingletons.AuthInterceptor().HTTPInterceptor,
			Authorizer:      authzUser.With(permissions.View(resources.DebugMetrics)),
			ServerHandler:   h,
			Compression:     true,
		})
	}
	return customRoutes
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
			detection.GetDetector().Stop()
			globaldb.Close()
			log.Infof("Central terminated")
			return
		}
	}
}
