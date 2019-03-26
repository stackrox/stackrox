package main

import (
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
	authproviderService "github.com/stackrox/rox/central/authprovider/service"
	authProviderStore "github.com/stackrox/rox/central/authprovider/store"
	"github.com/stackrox/rox/central/cli"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterService "github.com/stackrox/rox/central/cluster/service"
	clustersZip "github.com/stackrox/rox/central/clusters/zip"
	complianceHandlers "github.com/stackrox/rox/central/compliance/handlers"
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
	"github.com/stackrox/rox/central/imageintegration"
	iiDatastore "github.com/stackrox/rox/central/imageintegration/datastore"
	iiService "github.com/stackrox/rox/central/imageintegration/service"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/jwt"
	licenseService "github.com/stackrox/rox/central/license/service"
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
	"github.com/stackrox/rox/central/role/mapper"
	"github.com/stackrox/rox/central/role/resources"
	roleService "github.com/stackrox/rox/central/role/service"
	roleStore "github.com/stackrox/rox/central/role/store"
	searchService "github.com/stackrox/rox/central/search/service"
	secretService "github.com/stackrox/rox/central/secret/service"
	sensorService "github.com/stackrox/rox/central/sensor/service"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/sensor/service/pipeline/all"
	serviceAccountService "github.com/stackrox/rox/central/serviceaccount/service"
	siService "github.com/stackrox/rox/central/serviceidentities/service"
	siStore "github.com/stackrox/rox/central/serviceidentities/store"
	summaryService "github.com/stackrox/rox/central/summary/service"
	userService "github.com/stackrox/rox/central/user/service"
	"github.com/stackrox/rox/central/version"
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
	"github.com/stackrox/rox/pkg/migrations"
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
	ensureDB()

	// Now that we verified that the DB can be loaded, remove the .backup directory
	if err := os.RemoveAll(filepath.Join(migrations.DBMountPath, ".backup")); err != nil {
		log.Errorf("Failed to remove backup DB: %v", err)
	}

	go startGRPCServer()

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM)
	waitForTerminationSignal(signalsC)
}

func ensureDB() {
	err := version.Ensure(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB())
	if err != nil {
		log.Panicf("DB version check failed. You may need to run migrations: %v", err)
	}
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

func startGRPCServer() {
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

	userpass.RegisterAuthProviderOrPanic(registry)

	idExtractors := []authn.IdentityExtractor{
		service.NewExtractor(), // internal services
		tokenbased.NewExtractor(roleStore.Singleton(), jwt.ValidatorSingleton()), // JWT tokens
		userpass.IdentityExtractorOrPanic(),
	}

	config := pkgGRPC.Config{
		CustomRoutes:       customRoutes(),
		TLS:                verifier.CA{},
		IdentityExtractors: idExtractors,
		AuthProviders:      registry,
	}

	if features.AuditLogging.Enabled() {
		config.Auditor = audit.New(processor.Singleton())
	}

	server := pkgGRPC.NewAPI(config)

	servicesToRegister := []pkgGRPC.APIService{
		alertService.Singleton(),
		authService.New(),
		apiTokenService.Singleton(),
		authproviderService.New(registry),
		clusterService.Singleton(),
		complianceService.Singleton(),
		complianceManagerService.Singleton(),
		debugService.Singleton(),
		deploymentService.Singleton(),
		detectionService.Singleton(),
		groupService.Singleton(),
		imageService.Singleton(),
		iiService.Singleton(),
		metadataService.New(),
		namespaceService.Singleton(),
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
		serviceAccountService.Singleton(),
		siService.Singleton(),
		summaryService.Singleton(),
		userService.Singleton(),
		sensorService.New(connection.ManagerSingleton(), all.Singleton(), clusterDataStore.Singleton()),
	}

	if features.LicenseEnforcement.Enabled() {
		servicesToRegister = append(servicesToRegister, licenseService.Singleton())
	}

	if err := manager.Singleton().Start(); err != nil {
		log.Panicf("could not start compliance manager: %v", err)
	}

	server.Register(servicesToRegister...)

	enrichanddetect.GetLoop().Start()
	startedSig := server.Start()

	go registerDelayedIntegrations(iiStore.DelayedIntegrations)
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

func registerDelayedIntegrations(integrationsInput []iiStore.DelayedIntegration) {
	integrations := make(map[int]iiStore.DelayedIntegration, len(integrationsInput))
	for k, v := range integrationsInput {
		integrations[k] = v
	}
	ds := iiDatastore.Singleton()
	for len(integrations) > 0 {
		for idx, integration := range integrations {
			_, exists, _ := ds.GetImageIntegration(integration.Integration.GetId())
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
				err = ds.UpdateImageIntegration(integration.Integration)
				if err != nil {
					// so, we added the integration to the set but we weren't able to save it.
					// This is ok -- the image scanner will "work" and after a restart we'll try to save it again.
					log.Errorf("We added the %q integration, but saving it failed with: %v. We'll try again next restart", integration.Integration.GetName(), err)
				} else {
					log.Infof("Registered integration %q", integration.Integration.GetName())
				}
				enrichanddetect.GetLoop().ShortCircuit()
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

// allResourcesViewPermissions returns a slice containing view permissions for all resource types.
func allResourcesModifyPermissions() []*v1.Permission {
	resourceLst := resources.ListAll()
	result := make([]*v1.Permission, len(resourceLst))
	for i, resource := range resourceLst {
		result[i] = permissions.Modify(resource)
	}
	return result
}

func customRoutes() (customRoutes []routes.CustomRoute) {
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
			Authorizer:    authzUser.With(allResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.BackupDB(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB()),
			Compression:   true,
		},
		{
			Route:         "/db/export",
			Authorizer:    authzUser.With(allResourcesViewPermissions()...),
			ServerHandler: globaldbHandlers.ExportDB(globaldb.GetGlobalDB(), globaldb.GetGlobalBadgerDB()),
			Compression:   true,
		},
		{
			Route:         "/db/restore",
			Authorizer:    authzUser.With(allResourcesModifyPermissions()...),
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

func waitForTerminationSignal(signalsC <-chan os.Signal) {
	sig := <-signalsC
	log.Infof("Caught %s signal", sig)
	enrichanddetect.GetLoop().Stop()
	globaldb.Close()
	log.Infof("Central terminated")
	return
}
