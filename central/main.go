package main

import (
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	alertService "github.com/stackrox/rox/central/alert/service"
	apiTokenService "github.com/stackrox/rox/central/apitoken/service"
	authService "github.com/stackrox/rox/central/auth/service"
	"github.com/stackrox/rox/central/auth/userpass"
	authproviderService "github.com/stackrox/rox/central/authprovider/service"
	authProviderStore "github.com/stackrox/rox/central/authprovider/store"
	benchmarkService "github.com/stackrox/rox/central/benchmark/service"
	brService "github.com/stackrox/rox/central/benchmarkresult/service"
	bsService "github.com/stackrox/rox/central/benchmarkscan/service"
	bshService "github.com/stackrox/rox/central/benchmarkschedule/service"
	btService "github.com/stackrox/rox/central/benchmarktrigger/service"
	"github.com/stackrox/rox/central/cli"
	"github.com/stackrox/rox/central/cluster/datastore"
	clusterService "github.com/stackrox/rox/central/cluster/service"
	clustersZip "github.com/stackrox/rox/central/clusters/zip"
	"github.com/stackrox/rox/central/compliance/manager"
	complianceManagerService "github.com/stackrox/rox/central/compliance/manager/service"
	complianceService "github.com/stackrox/rox/central/compliance/service"
	debugService "github.com/stackrox/rox/central/debug/service"
	deploymentService "github.com/stackrox/rox/central/deployment/service"
	detectionService "github.com/stackrox/rox/central/detection/service"
	"github.com/stackrox/rox/central/docs"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/globaldb"
	globaldbHandlers "github.com/stackrox/rox/central/globaldb/handlers"
	graphqlHandler "github.com/stackrox/rox/central/graphql/handler"
	groupService "github.com/stackrox/rox/central/group/service"
	imageService "github.com/stackrox/rox/central/image/service"
	iiService "github.com/stackrox/rox/central/imageintegration/service"
	interceptorSingletons "github.com/stackrox/rox/central/interceptor/singletons"
	"github.com/stackrox/rox/central/jwt"
	logimbueHandler "github.com/stackrox/rox/central/logimbue/handler"
	metadataService "github.com/stackrox/rox/central/metadata/service"
	"github.com/stackrox/rox/central/metrics"
	networkFlowService "github.com/stackrox/rox/central/networkflow/service"
	networkPolicyService "github.com/stackrox/rox/central/networkpolicies/service"
	nodeService "github.com/stackrox/rox/central/node/service"
	notifierService "github.com/stackrox/rox/central/notifier/service"
	_ "github.com/stackrox/rox/central/notifiers/all" // These imports are required to register things from the respective packages.
	pingService "github.com/stackrox/rox/central/ping/service"
	policyService "github.com/stackrox/rox/central/policy/service"
	processIndicatorService "github.com/stackrox/rox/central/processindicator/service"
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/central/role/resources"
	roleService "github.com/stackrox/rox/central/role/service"
	roleStore "github.com/stackrox/rox/central/role/store"
	searchService "github.com/stackrox/rox/central/search/service"
	secretService "github.com/stackrox/rox/central/secret/service"
	sensorService "github.com/stackrox/rox/central/sensor/service"
	"github.com/stackrox/rox/central/sensor/service/streamer"
	siService "github.com/stackrox/rox/central/serviceidentities/service"
	siStore "github.com/stackrox/rox/central/serviceidentities/store"
	summaryService "github.com/stackrox/rox/central/summary/service"
	userService "github.com/stackrox/rox/central/user/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
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

	authProviderBackendFactories = map[string]authproviders.BackendFactoryCreator{
		oidc.TypeName: oidc.NewFactory,
		"auth0":       oidc.NewFactory, // legacy
		saml.TypeName: saml.NewFactory,
	}
)

const (
	ssoURLPathPrefix     = "/sso/"
	tokenRedirectURLPath = "/auth/response/generic"

	grpcServerWatchdogTimeout = 20 * time.Second
)

func main() {
	central := newCentral()

	go central.startGRPCServer()

	central.processForever()
}

func watchdog(signal *concurrency.Signal, timeout time.Duration) {
	if !concurrency.WaitWithTimeout(signal, timeout) {
		log.Errorf("API server failed to start within %v!", timeout)
		log.Errorf("This usually means something is *very* wrong. Terminating ...")
		syscall.Kill(syscall.Getpid(), syscall.SIGABRT)
	}
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
	registry, err := authproviders.NewStoreBackedRegistry(
		ssoURLPathPrefix, tokenRedirectURLPath,
		authProviderStore.New(globaldb.GetGlobalDB()), jwt.IssuerFactorySingleton(),
		mapper.FactorySingleton())

	if err != nil {
		log.Panicf("Could not create auth provider registry: %v", err)
	}

	for typeName, factoryCreator := range authProviderBackendFactories {
		if err := registry.RegisterBackendFactory(typeName, factoryCreator); err != nil {
			log.Panicf("Could not register %s auth provider factory: %v", typeName, err)
		}
	}

	if err := registry.Init(); err != nil {
		log.Panicf("Could not initialize auth provider registry: %v", err)
	}

	idExtractors := []authn.IdentityExtractor{
		service.NewExtractor(), // internal services
		tokenbased.NewExtractor(roleStore.Singleton(), jwt.ValidatorSingleton()), // JWT tokens
	}

	if features.HtpasswdAuth.Enabled() {
		idExtractors = append(idExtractors, userpass.IdentityExtractorOrPanic())
		userpass.RegisterAuthProviderOrPanic(registry)
	}

	config := pkgGRPC.Config{
		CustomRoutes:       c.customRoutes(),
		TLS:                verifier.CA{},
		UnaryInterceptors:  interceptorSingletons.GrpcUnaryInterceptors(),
		StreamInterceptors: interceptorSingletons.GrpcStreamInterceptors(),
		IdentityExtractors: idExtractors,
		AuthProviders:      registry,
	}

	c.server = pkgGRPC.NewAPI(config)

	servicesToRegister := []pkgGRPC.APIService{
		alertService.Singleton(),
		authService.New(),
		apiTokenService.Singleton(),
		authproviderService.New(registry),
		benchmarkService.Singleton(),
		bsService.Singleton(),
		bshService.Singleton(),
		brService.Singleton(),
		btService.Singleton(),
		clusterService.Singleton(),
		debugService.Singleton(),
		deploymentService.Singleton(),
		detectionService.Singleton(),
		groupService.Singleton(),
		imageService.Singleton(),
		iiService.Singleton(),
		metadataService.New(),
		networkFlowService.Singleton(),
		networkPolicyService.Singleton(),
		nodeService.Singleton(),
		notifierService.Singleton(),
		pingService.Singleton(),
		policyService.Singleton(),
		processIndicatorService.Singleton(),
		roleService.Singleton(),
		searchService.Singleton(),
		secretService.Singleton(),
		siService.Singleton(),
		summaryService.Singleton(),
		userService.Singleton(),
		sensorService.New(streamer.ManagerSingleton()),
	}
	if features.Compliance.Enabled() {
		servicesToRegister = append(servicesToRegister,
			complianceService.New(),
			complianceManagerService.Singleton())

		if err := manager.Singleton().Start(); err != nil {
			log.Panicf("could not start compliance manager: %v", err)
		}
	}

	c.server.Register(servicesToRegister...)

	enrichanddetect.GetLoop().Start()
	startedSig := c.server.Start()
	go watchdog(startedSig, grpcServerWatchdogTimeout)
}

// allResourcesViewPermissions returns a slice containing view permissions for all resource types.
func allResourcesViewPermissions() []*v1.Permission {
	resourceLst := resources.ListAll()
	result := make([]*v1.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.View(resource)
	}
	return result
}

// allResourcesViewPermissions returns a slice containing view permissions for all resource types.
func allResourcesModifyPermissions() []*v1.Permission {
	resourceLst := resources.ListAll()
	result := make([]*v1.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.Modify(resource)
	}
	return result
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
			ServerHandler: clustersZip.Handler(datastore.Singleton(), siStore.Singleton()),
			Compression:   false,
		},
		{
			Route:         "/api/cli/download/",
			Authorizer:    authzUser.With(),
			ServerHandler: cli.Handler(),
			Compression:   true,
		},
		{
			Route:         "/db/backup",
			Authorizer:    authzUser.With(allResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.BackupDB(globaldb.GetGlobalDB()),
			Compression:   true,
		},
		{
			Route:         "/db/export",
			Authorizer:    authzUser.With(allResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.ExportDB(globaldb.GetGlobalDB()),
			Compression:   true,
		},
		{
			Route:         "/db/restore",
			Authorizer:    authzUser.With(allResourcesModifyPermissions()...),
			ServerHandler: globaldbHandlers.RestoreDB(globaldb.GetGlobalDB()),
		},
		{
			Route:         "/metrics",
			Authorizer:    allow.Anonymous(),
			ServerHandler: promhttp.Handler(),
			Compression:   false,
		},
		{
			Route:         "/api/docs/swagger",
			Authorizer:    authzUser.With(permissions.View(resources.APIToken)),
			ServerHandler: docs.Swagger(),
			Compression:   true,
		},
		{
			Route:         "/api/graphql",
			Authorizer:    allow.Anonymous(), // graphql enforces permissions internally
			ServerHandler: graphqlHandler.Handler(),
			Compression:   true,
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
	customRoutes := make([]routes.CustomRoute, 0, len(routes.DebugRoutes))

	for r, h := range routes.DebugRoutes {
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
