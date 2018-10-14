package main

import (
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	// These imports are required to register things from the respective packages.
	_ "github.com/stackrox/rox/pkg/auth/authproviders/all"
	_ "github.com/stackrox/rox/pkg/notifiers/all"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	alertService "github.com/stackrox/rox/central/alert/service"
	apiTokenService "github.com/stackrox/rox/central/apitoken/service"
	authService "github.com/stackrox/rox/central/auth/service"
	"github.com/stackrox/rox/central/authprovider/cachedstore"
	authproviderService "github.com/stackrox/rox/central/authprovider/service"
	benchmarkService "github.com/stackrox/rox/central/benchmark/service"
	brService "github.com/stackrox/rox/central/benchmarkresult/service"
	bsService "github.com/stackrox/rox/central/benchmarkscan/service"
	bshService "github.com/stackrox/rox/central/benchmarkschedule/service"
	btService "github.com/stackrox/rox/central/benchmarktrigger/service"
	clusterService "github.com/stackrox/rox/central/cluster/service"
	clustersZip "github.com/stackrox/rox/central/clusters/zip"
	deploymentService "github.com/stackrox/rox/central/deployment/service"
	detectionService "github.com/stackrox/rox/central/detection/service"
	enforcementService "github.com/stackrox/rox/central/enforcement/service"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/globaldb"
	globaldbHandlers "github.com/stackrox/rox/central/globaldb/handlers"
	imageService "github.com/stackrox/rox/central/image/service"
	iiService "github.com/stackrox/rox/central/imageintegration/service"
	interceptorSingletons "github.com/stackrox/rox/central/interceptor/singletons"
	"github.com/stackrox/rox/central/jwt"
	logimbueHandler "github.com/stackrox/rox/central/logimbue/handler"
	metadataService "github.com/stackrox/rox/central/metadata/service"
	"github.com/stackrox/rox/central/metrics"
	networkFlowService "github.com/stackrox/rox/central/networkflow/service"
	networkPolicyService "github.com/stackrox/rox/central/networkpolicies/service"
	notifierService "github.com/stackrox/rox/central/notifier/service"
	pingService "github.com/stackrox/rox/central/ping/service"
	policyService "github.com/stackrox/rox/central/policy/service"
	"github.com/stackrox/rox/central/role/resources"
	roleService "github.com/stackrox/rox/central/role/service"
	roleStore "github.com/stackrox/rox/central/role/store"
	searchService "github.com/stackrox/rox/central/search/service"
	secretService "github.com/stackrox/rox/central/secret/service"
	seService "github.com/stackrox/rox/central/sensorevent/service"
	"github.com/stackrox/rox/central/sensornetworkflow"
	siService "github.com/stackrox/rox/central/serviceidentities/service"
	summaryService "github.com/stackrox/rox/central/summary/service"
	"github.com/stackrox/rox/central/user/mapper"
	"github.com/stackrox/rox/pkg/auth/permissions"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authn/tokenbased"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	authzUser "github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/ui"
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
	signal.Notify(central.signalsC, syscall.SIGINT, syscall.SIGTERM)

	return central
}

func (c *central) startGRPCServer() {
	config := pkgGRPC.Config{
		CustomRoutes:       c.customRoutes(),
		TLS:                verifier.CA{},
		UnaryInterceptors:  interceptorSingletons.GrpcUnaryInterceptors(),
		StreamInterceptors: interceptorSingletons.GrpcStreamInterceptors(),
		IdentityExtractors: []authn.IdentityExtractor{
			service.NewExtractor(), // internal services
			tokenbased.NewExtractor(roleStore.Singleton(), jwt.ValidatorSingleton()), // JWT tokens (new)
			tokenbased.NewLegacyExtractor(cachedstore.Singleton(), usermapper.New(roleStore.Singleton())),
		},
		AuthProviders: cachedstore.Singleton(),
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
		detectionService.Singleton(),
		enforcementService.Singleton(),
		imageService.Singleton(),
		iiService.Singleton(),
		metadataService.New(),
		networkFlowService.Singleton(),
		networkPolicyService.Singleton(),
		notifierService.Singleton(),
		pingService.Singleton(),
		policyService.Singleton(),
		roleService.Singleton(),
		searchService.Singleton(),
		secretService.Singleton(),
		seService.Singleton(),
		siService.Singleton(),
		summaryService.Singleton(),
		sensornetworkflow.Singleton(),
	)

	enrichanddetect.GetLoop().Start()
	c.server.Start()
}

// allResourcesViewPermissions returns a slice containing view permissions for all resource types.
func allResourcesViewPermissions() []permissions.Permission {
	resourceLst := resources.ListAll()
	result := make([]permissions.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.View(resource)
	}
	return result
}

// To export the DB, you need to be able to view _everything_.
func dbExportOrBackupAuthorizer() authz.Authorizer {
	return authzUser.With(allResourcesViewPermissions()...)
}

func (c *central) customRoutes() (customRoutes []routes.CustomRoute) {
	customRoutes = []routes.CustomRoute{
		{
			Route:         "/",
			Authorizer:    allow.Anonymous(),
			ServerHandler: ui.Mux(),
			Compression:   true,
		},
		{
			Route:         "/api/extensions/clusters/zip",
			Authorizer:    authzUser.With(permissions.View(resources.Cluster), permissions.View(resources.ServiceIdentity)),
			ServerHandler: clustersZip.Handler(clusterService.Singleton(), siService.Singleton()),
			Compression:   false,
		},

		{
			Route:         "/db/backup",
			Authorizer:    dbExportOrBackupAuthorizer(),
			ServerHandler: globaldbHandlers.BackupDB(globaldb.GetGlobalDB()),
			Compression:   true,
		},
		{
			Route:         "/db/export",
			Authorizer:    dbExportOrBackupAuthorizer(),
			ServerHandler: globaldbHandlers.ExportDB(globaldb.GetGlobalDB()),
			Compression:   true,
		},
		{
			Route:         "/metrics",
			Authorizer:    allow.Anonymous(),
			ServerHandler: promhttp.Handler(),
			Compression:   false,
		},
	}

	logImbueRoute := "/api/logimbue"
	customRoutes = append(customRoutes,
		routes.CustomRoute{
			Route: logImbueRoute,
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
			Route:         r,
			Authorizer:    authzUser.With(permissions.View(resources.DebugMetrics)),
			ServerHandler: h,
			Compression:   true,
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
			enrichanddetect.GetLoop().Stop()
			globaldb.Close()
			log.Infof("Central terminated")
			return
		}
	}
}
