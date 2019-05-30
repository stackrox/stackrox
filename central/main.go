package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	alertService "github.com/stackrox/rox/central/alert/service"
	apiTokenService "github.com/stackrox/rox/central/apitoken/service"
	"github.com/stackrox/rox/central/audit"
	authService "github.com/stackrox/rox/central/auth/service"
	"github.com/stackrox/rox/central/auth/userpass"
	authProviderDS "github.com/stackrox/rox/central/authprovider/datastore"
	authproviderService "github.com/stackrox/rox/central/authprovider/service"
	"github.com/stackrox/rox/central/cli"
	clientCAManager "github.com/stackrox/rox/central/clientca/manager"
	clientCAService "github.com/stackrox/rox/central/clientca/service"
	clientCAStore "github.com/stackrox/rox/central/clientca/store"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterService "github.com/stackrox/rox/central/cluster/service"
	clustersZip "github.com/stackrox/rox/central/clusters/zip"
	complianceHandlers "github.com/stackrox/rox/central/compliance/handlers"
	complianceManager "github.com/stackrox/rox/central/compliance/manager"
	complianceManagerService "github.com/stackrox/rox/central/compliance/manager/service"
	complianceService "github.com/stackrox/rox/central/compliance/service"
	configService "github.com/stackrox/rox/central/config/service"
	debugService "github.com/stackrox/rox/central/debug/service"
	deploymentService "github.com/stackrox/rox/central/deployment/service"
	detectionService "github.com/stackrox/rox/central/detection/service"
	developmentService "github.com/stackrox/rox/central/development/service"
	"github.com/stackrox/rox/central/docs"
	"github.com/stackrox/rox/central/ed"
	_ "github.com/stackrox/rox/central/externalbackups/plugins/all" // Import all of the external backup plugins
	backupService "github.com/stackrox/rox/central/externalbackups/service"
	featureFlagService "github.com/stackrox/rox/central/featureflags/service"
	"github.com/stackrox/rox/central/globaldb"
	globaldbHandlers "github.com/stackrox/rox/central/globaldb/handlers"
	graphqlHandler "github.com/stackrox/rox/central/graphql/handler"
	groupService "github.com/stackrox/rox/central/group/service"
	imageService "github.com/stackrox/rox/central/image/service"
	"github.com/stackrox/rox/central/imageintegration"
	iiDatastore "github.com/stackrox/rox/central/imageintegration/datastore"
	iiService "github.com/stackrox/rox/central/imageintegration/service"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/jwt"
	licenseEnforcer "github.com/stackrox/rox/central/license/enforcer"
	licenseService "github.com/stackrox/rox/central/license/service"
	licenseSingletons "github.com/stackrox/rox/central/license/singleton"
	logimbueHandler "github.com/stackrox/rox/central/logimbue/handler"
	metadataService "github.com/stackrox/rox/central/metadata/service"
	namespaceService "github.com/stackrox/rox/central/namespace/service"
	networkFlowService "github.com/stackrox/rox/central/networkflow/service"
	networkPolicyService "github.com/stackrox/rox/central/networkpolicies/service"
	nodeService "github.com/stackrox/rox/central/node/service"
	"github.com/stackrox/rox/central/notifier/processor"
	notifierService "github.com/stackrox/rox/central/notifier/service"
	_ "github.com/stackrox/rox/central/notifiers/all" // These imports are required to register things from the respective packages.
	pingService "github.com/stackrox/rox/central/ping/service"
	policyService "github.com/stackrox/rox/central/policy/service"
	processIndicatorService "github.com/stackrox/rox/central/processindicator/service"
	processWhitelistService "github.com/stackrox/rox/central/processwhitelist/service"
	"github.com/stackrox/rox/central/pruning"
	rbacService "github.com/stackrox/rox/central/rbac/service"
	"github.com/stackrox/rox/central/reprocessor"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/central/role/resources"
	roleService "github.com/stackrox/rox/central/role/service"
	"github.com/stackrox/rox/central/sac/transitional"
	searchService "github.com/stackrox/rox/central/search/service"
	secretService "github.com/stackrox/rox/central/secret/service"
	sensorService "github.com/stackrox/rox/central/sensor/service"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline/all"
	serviceAccountService "github.com/stackrox/rox/central/serviceaccount/service"
	siStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	siService "github.com/stackrox/rox/central/serviceidentities/service"
	summaryService "github.com/stackrox/rox/central/summary/service"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/central/ui"
	userService "github.com/stackrox/rox/central/user/service"
	"github.com/stackrox/rox/central/version"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/auth/authproviders/saml"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
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
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
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

	publicAPIEndpoint     = ":8443"
	insecureLocalEndpoint = "127.0.0.1:8444"
)

func main() {
	ensureDB()

	// Now that we verified that the DB can be loaded, remove the .backup directory
	if err := os.RemoveAll(filepath.Join(migrations.DBMountPath, ".backup")); err != nil {
		log.Errorf("Failed to remove backup DB: %v", err)
	}

	var restartingFlag concurrency.Flag

	licenseMgr := licenseSingletons.ManagerSingleton()
	initialLicense, err := licenseMgr.Initialize(licenseEnforcer.New(&restartingFlag))
	if err != nil {
		log.Fatalf("Could not initialize license manager: %v", err)
	}

	if initialLicense == nil {
		log.Error("*** No valid license found")
		log.Error("*** ")
		log.Error("*** Server starting in limited mode until license activated")
		go startLimitedModeServer(&restartingFlag)
		waitForTerminationSignal()
		return
	}

	log.Info("Extracting StackRox data ...")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := ed.ED(ctx); err != nil {
		log.Fatalf("Could not extract data: %v", err)
	}
	log.Info("Successfully extracted StackRox data")

	go startMainServer(&restartingFlag)

	waitForTerminationSignal()
}

func ensureDB() {
	err := version.Ensure(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB())
	if err != nil {
		log.Panicf("DB version check failed. You may need to run migrations: %v", err)
	}
}

type invalidLicenseFactory struct {
	restartingFlag *concurrency.Flag
}

func (f invalidLicenseFactory) TLSConfigurer() verifier.TLSConfigurer {
	return tlsconfig.NewCentralTLSConfigurer()
}

func (f invalidLicenseFactory) ServicesToRegister(authproviders.Registry) []pkgGRPC.APIService {
	return []pkgGRPC.APIService{
		licenseService.New(true, licenseSingletons.ManagerSingleton()),
		metadataService.New(f.restartingFlag, licenseSingletons.ManagerSingleton()),
		pingService.Singleton(), // required for dev scripts & health checking
	}
}

func (invalidLicenseFactory) StartServices() {
}

func (invalidLicenseFactory) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		uiRoute(ui.RestrictedModeMux()),
	}
}

func startLimitedModeServer(restartingFlag *concurrency.Flag) {
	startGRPCServer(invalidLicenseFactory{
		restartingFlag: restartingFlag,
	})
}

type serviceFactory interface {
	CustomRoutes() (customRoutes []routes.CustomRoute)
	ServicesToRegister(authproviders.Registry) []pkgGRPC.APIService
	TLSConfigurer() verifier.TLSConfigurer
	StartServices()
}

type defaultFactory struct {
	restartingFlag *concurrency.Flag
	caManager      clientCAManager.ClientCAManager
}

func (f defaultFactory) TLSConfigurer() verifier.TLSConfigurer {
	// is nil if feature flag is false
	if f.caManager != nil {
		return f.caManager.TLSConfigurer()
	}
	return tlsconfig.NewCentralTLSConfigurer()
}

func (defaultFactory) StartServices() {
	if err := complianceManager.Singleton().Start(); err != nil {
		log.Panicf("could not start compliance manager: %v", err)
	}
	reprocessor.Singleton().Start()
	pruning.Singleton().Start()

	go registerDelayedIntegrations(iiStore.DelayedIntegrations)
}

func (f defaultFactory) ServicesToRegister(registry authproviders.Registry) []pkgGRPC.APIService {
	servicesToRegister := []pkgGRPC.APIService{
		alertService.Singleton(),
		authService.New(),
		apiTokenService.Singleton(),
		authproviderService.New(registry),
		backupService.Singleton(),
		clusterService.Singleton(),
		complianceService.Singleton(),
		complianceManagerService.Singleton(),
		configService.Singleton(),
		debugService.Singleton(),
		deploymentService.Singleton(),
		detectionService.Singleton(),
		featureFlagService.Singleton(),
		groupService.Singleton(),
		imageService.Singleton(),
		iiService.Singleton(),
		metadataService.New(f.restartingFlag, licenseSingletons.ManagerSingleton()),
		namespaceService.Singleton(),
		networkFlowService.Singleton(),
		networkPolicyService.Singleton(),
		nodeService.Singleton(),
		notifierService.Singleton(),
		pingService.Singleton(),
		policyService.Singleton(),
		processIndicatorService.Singleton(),
		processWhitelistService.Singleton(),
		roleService.Singleton(),
		rbacService.Singleton(),
		searchService.Singleton(),
		secretService.Singleton(),
		serviceAccountService.Singleton(),
		siService.Singleton(),
		summaryService.Singleton(),
		userService.Singleton(),
		sensorService.New(connection.ManagerSingleton(), all.Singleton(), clusterDataStore.Singleton()),
		licenseService.New(false, licenseSingletons.ManagerSingleton()),
	}

	if env.DevelopmentBuild.Setting() == "true" {
		servicesToRegister = append(servicesToRegister, developmentService.Singleton())
	}

	if features.ClientCAAuth.Enabled() {
		servicesToRegister = append(servicesToRegister, clientCAService.New(f.caManager))
	}
	return servicesToRegister
}

func startMainServer(restartingFlag *concurrency.Flag) {
	factory := defaultFactory{
		restartingFlag: restartingFlag,
	}
	if features.ClientCAAuth.Enabled() {
		factory.caManager = clientCAManager.New(clientCAStore.Singleton())
		utils.Must(factory.caManager.Initialize())
	}
	startGRPCServer(factory)
}

func watchdog(signal *concurrency.Signal, timeout time.Duration) {
	if !concurrency.WaitWithTimeout(signal, timeout) {
		log.Errorf("API server failed to start within %v!", timeout)
		log.Errorf("This usually means something is *very* wrong. Terminating ...")
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGABRT); err != nil {
			panic(err)
		}
	}
}

func startGRPCServer(factory serviceFactory) {
	// Temporarily elevate permissions to modify auth providers.
	authProviderRegisteringCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.AuthProvider)))

	// Create the registry of applied auth providers.
	registry, err := authproviders.NewStoreBackedRegistry(
		ssoURLPathPrefix, tokenRedirectURLPath,
		authProviderDS.Singleton(), jwt.IssuerFactorySingleton(),
		mapper.FactorySingleton())
	if err != nil {
		log.Panicf("Could not create auth provider registry: %v", err)
	}

	for typeName, factoryCreator := range authProviderBackendFactories {
		if err := registry.RegisterBackendFactory(authProviderRegisteringCtx, typeName, factoryCreator); err != nil {
			log.Panicf("Could not register %s auth provider factory: %v", typeName, err)
		}
	}
	if err := registry.Init(); err != nil {
		log.Panicf("Could not initialize auth provider registry: %v", err)
	}

	userpass.RegisterAuthProviderOrPanic(authProviderRegisteringCtx, registry)

	idExtractors := []authn.IdentityExtractor{
		service.NewExtractor(), // internal services
		tokenbased.NewExtractor(roleDataStore.Singleton(), jwt.ValidatorSingleton()), // JWT tokens
		userpass.IdentityExtractorOrPanic(),
	}

	config := pkgGRPC.Config{
		CustomRoutes:          factory.CustomRoutes(),
		TLS:                   factory.TLSConfigurer(),
		IdentityExtractors:    idExtractors,
		AuthProviders:         registry,
		InsecureLocalEndpoint: insecureLocalEndpoint,
		PublicEndpoint:        publicAPIEndpoint,
	}

	if features.AuditLogging.Enabled() {
		config.Auditor = audit.New(processor.Singleton())
	}

	log.Infof("Scoped access control enabled: %v", features.ScopedAccessControl.Enabled())
	if features.ScopedAccessControl.Enabled() {
		config.ContextEnrichers = append(config.ContextEnrichers, transitional.LegacyAccessScopesContextEnricher)
		config.UnaryInterceptors = append(config.UnaryInterceptors, transitional.VerifySACScopeChecksInterceptor)
	}

	server := pkgGRPC.NewAPI(config)
	server.Register(factory.ServicesToRegister(registry)...)

	factory.StartServices()
	startedSig := server.Start()

	go watchdog(startedSig, grpcServerWatchdogTimeout)
}

func registerDelayedIntegrations(integrationsInput []iiStore.DelayedIntegration) {
	integrations := make(map[int]iiStore.DelayedIntegration, len(integrationsInput))
	for k, v := range integrationsInput {
		integrations[k] = v
	}
	ds := iiDatastore.Singleton()
	for len(integrations) > 0 {
		for idx, integration := range integrations {
			_, exists, _ := ds.GetImageIntegration(context.TODO(), integration.Integration.GetId())
			if exists {
				delete(integrations, idx)
				continue
			}
			ready := integration.Trigger()
			if !ready {
				continue
			}
			// add the integration first, which is more likely to fail. If it does, no big deal -- you can still try to
			// manually add it and get the error message.
			err := imageintegration.ToNotify().NotifyUpdated(integration.Integration)
			if err == nil {
				err = ds.UpdateImageIntegration(context.TODO(), integration.Integration)
				if err != nil {
					// so, we added the integration to the set but we weren't able to save it.
					// This is ok -- the image scanner will "work" and after a restart we'll try to save it again.
					log.Errorf("We added the %q integration, but saving it failed with: %v. We'll try again next restart", integration.Integration.GetName(), err)
				} else {
					log.Infof("Registered integration %q", integration.Integration.GetName())
				}
				reprocessor.Singleton().ShortCircuit()
			} else {
				log.Errorf("Unable to register integration %q: %v", integration.Integration.GetName(), err)
			}
			// either way, time to stop watching this entry
			delete(integrations, idx)
		}
		time.Sleep(5 * time.Second)
	}
	log.Debugf("All dynamic integrations registered, exiting")
}

func uiRoute(uiHandler http.Handler) routes.CustomRoute {
	return routes.CustomRoute{
		Route:         "/",
		Authorizer:    allow.Anonymous(),
		ServerHandler: uiHandler,
		Compression:   true,
	}

}

func (defaultFactory) CustomRoutes() (customRoutes []routes.CustomRoute) {
	customRoutes = []routes.CustomRoute{
		uiRoute(ui.Mux()),
		{
			Route:         "/api/extensions/clusters/zip",
			Authorizer:    authzUser.With(permissions.View(resources.Cluster), permissions.View(resources.ServiceIdentity)),
			ServerHandler: clustersZip.Handler(clusterDataStore.Singleton(), siStore.Singleton()),
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
			Authorizer:    authzUser.With(resources.AllResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.BackupDB(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB()),
			Compression:   true,
		},
		{
			Route:         "/db/export",
			Authorizer:    authzUser.With(resources.AllResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.ExportDB(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB()),
			Compression:   true,
		},
		{
			Route:         "/db/restore",
			Authorizer:    authzUser.With(resources.AllResourcesModifyPermissions()...),
			ServerHandler: globaldbHandlers.RestoreDB(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB()),
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
		{
			Route:         "/api/compliance/export/csv",
			Authorizer:    authzUser.With(permissions.View(resources.Compliance)),
			ServerHandler: complianceHandlers.CSVHandler(),
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

	customRoutes = append(customRoutes, debugRoutes()...)
	return
}

func debugRoutes() []routes.CustomRoute {
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

func waitForTerminationSignal() {
	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
	reprocessor.Singleton().Stop()
	log.Infof("Stopped reprocessor loop")
	pruning.Singleton().Stop()
	log.Infof("Stopped garbage collector")
	globaldb.Close()
	log.Infof("Central terminated")
}
