package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/central/globaldb"
	groupDS "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/logimbue/store"
	"github.com/stackrox/rox/central/logimbue/writer"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/postgres/stats"
	"github.com/stackrox/rox/pkg/prometheusutil"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

type logsMode int

// telemetryMode specifies how to use sensor/central telemetry to gather diagnostics.
// 0 - don't collect any telemetry data
// 1 - collect telemetry data for central only
// 2 - collect telemetry data from sensors and central
type telemetryMode int

const (
	cpuProfileDuration = 30 * time.Second

	noLogs logsMode = iota
	localLogs
	fullK8sIntrospectionData

	noTelemetry telemetryMode = iota
	telemetryCentralOnly
	telemetryCentralAndSensors

	centralClusterPrefix = "_central-cluster"

	pgStatStatementsMax = 1000

	layout    = "2006-01-02T15:04:05.000Z"
	logWindow = 20 * time.Minute
	// This timeout is safety net to prevent request from running forever.
	// We don't expect it to ever actually be reached. Actual timeout should be set on the client side.
	debugDumpHardTimeout = 1 * time.Hour
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.DebugService/GetLogLevel",
			"/v1.DebugService/StreamAuthzTraces",
		},
		user.With(permissions.Modify(resources.Administration)): {
			"/v1.DebugService/SetLogLevel",
		},
	})

	mainClusterConfig = k8sintrospect.DefaultConfig()
)

func init() {
	mainClusterConfig.PathPrefix = centralClusterPrefix
}

// Service provides the interface to the gRPC service for debugging
type Service interface {
	grpcPkg.APIServiceWithCustomRoutes
	v1.DebugServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	InternalDiagnosticsHandler() http.HandlerFunc
}

// New returns a Service that implements v1.DebugServiceServer
func New(clusters datastore.DataStore, sensorConnMgr connection.Manager, telemetryGatherer *gatherers.RoxGatherer,
	store store.Store, authzTraceSink observe.AuthzTraceSink, authProviderRegistry authproviders.Registry,
	groupDataStore groupDS.DataStore, roleDataStore roleDS.DataStore, configDataStore configDS.DataStore,
	notifierDataStore notifierDS.DataStore) Service {
	return &serviceImpl{
		clusters:             clusters,
		sensorConnMgr:        sensorConnMgr,
		telemetryGatherer:    telemetryGatherer,
		store:                store,
		authzTraceSink:       authzTraceSink,
		authProviderRegistry: authProviderRegistry,
		groupDataStore:       groupDataStore,
		roleDataStore:        roleDataStore,
		configDataStore:      configDataStore,
		notifierDataStore:    notifierDataStore,
	}
}

type serviceImpl struct {
	v1.UnimplementedDebugServiceServer

	sensorConnMgr        connection.Manager
	clusters             datastore.DataStore
	telemetryGatherer    *gatherers.RoxGatherer
	store                store.Store
	authzTraceSink       observe.AuthzTraceSink
	authProviderRegistry authproviders.Registry
	groupDataStore       groupDS.DataStore
	roleDataStore        roleDS.DataStore
	configDataStore      configDS.DataStore
	notifierDataStore    notifierDS.DataStore
}

// InternalDiagnosticsHandler returns handler to be served on "cluster-internal" port.
// Cluster-internal port is not exposed via k8s Service and only accessible to callers with k8s/Openshift cluster access.
// This handler shouldn't be exposed to other callers as it has no authorization and can elevate customer permissions.
func (s *serviceImpl) InternalDiagnosticsHandler() http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, r *http.Request) {
		// Adding scope checker as no authorizer is used, ergo no identity in context by default.
		ctx := sac.WithGlobalAccessScopeChecker(r.Context(), sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS)))
		s.getDiagnosticDumpWithCentral(responseWriter, r.WithContext(ctx), true)
	}
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterDebugServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterDebugServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetLogLevel returns a v1.LogLevelResponse object.
func (s *serviceImpl) GetLogLevel(_ context.Context, req *v1.GetLogLevelRequest) (*v1.LogLevelResponse, error) {
	resp := &v1.LogLevelResponse{}
	var unknownModules []string
	var forEachModule func(name string, m *logging.Module)

	// If the request is global, then return all modules who have a log level that does not match the global level
	if len(req.GetModules()) == 0 {
		level := logging.GetGlobalLogLevel()
		resp.Level = logging.LabelForLevelOrInvalid(level)
		forEachModule = func(name string, m *logging.Module) {
			moduleLevel := m.GetLogLevel()
			if moduleLevel != level {
				resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{
					Module: name, Level: logging.LabelForLevelOrInvalid(moduleLevel),
				})
			}
		}
	} else {
		forEachModule = func(name string, m *logging.Module) {
			if m == nil {
				unknownModules = append(unknownModules, name)
			} else {
				resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{
					Module: name, Level: logging.LabelForLevelOrInvalid(m.GetLogLevel()),
				})
			}
		}
	}

	logging.ForEachModule(forEachModule, req.GetModules())

	if len(unknownModules) > 0 {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}

	return resp, nil
}

// SetLogLevel implements v1.DebugServiceServer, and it sets the log level for StackRox services.
func (s *serviceImpl) SetLogLevel(_ context.Context, req *v1.LogLevelRequest) (*types.Empty, error) {
	levelStr := req.GetLevel()
	zapLevel, ok := logging.LevelForLabel(levelStr)
	if !ok {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unknown log level %s", levelStr)
	}

	// If this is a global request, then set the global level and return
	if len(req.GetModules()) == 0 {
		logging.SetGlobalLogLevel(zapLevel)
		return &types.Empty{}, nil
	}

	var unknownModules []string
	logging.ForEachModule(func(name string, m *logging.Module) {
		if m == nil {
			unknownModules = append(unknownModules, name)
		} else {
			m.SetLogLevel(zapLevel)
		}
	}, req.GetModules())

	if len(unknownModules) > 0 {
		return nil, errors.Wrapf(errox.InvalidArgs, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}

	return &types.Empty{}, nil
}

func (s *serviceImpl) StreamAuthzTraces(_ *v1.Empty, stream v1.DebugService_StreamAuthzTracesServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	traceC := s.authzTraceSink.Subscribe(ctx)
	for {
		select {
		case trace, ok := <-traceC:
			if !ok {
				return nil
			}
			err := stream.Send(trace)
			if err != nil {
				if err != io.EOF {
					log.Warnf("Error during authz trace streaming: %s", err.Error())
				}
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func fetchAndAddJSONToZip(ctx context.Context, zipWriter *zip.Writer, fileName string, fetchData func(ctx context.Context) (interface{}, error)) {
	jsonObj, errFetchData := fetchData(ctx)
	if errFetchData != nil {
		log.Error(errFetchData)

		return
	}

	if errAddToZip := addJSONToZip(zipWriter, fileName, jsonObj); errAddToZip != nil {
		log.Error(errAddToZip)
	}
}

func addJSONToZip(zipWriter *zip.Writer, fileName string, jsonObj interface{}) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, fileName)
	if err != nil {
		return errors.Wrapf(err, "unable to create zip file %q", fileName)
	}

	jsonEnc := json.NewEncoder(w)
	jsonEnc.SetIndent("", "  ")

	return jsonEnc.Encode(jsonObj)
}

func zipPGStatStatements(zipWriter *zip.Writer, name string) error {
	metricsWriter, err := zipWriterWithCurrentTimestamp(zipWriter, name)
	if err != nil {
		return err
	}

	return prometheusutil.ExportText(metricsWriter)
}
func zipPrometheusMetrics(zipWriter *zip.Writer, name string) error {
	metricsWriter, err := zipWriterWithCurrentTimestamp(zipWriter, name)
	if err != nil {
		return err
	}
	return prometheusutil.ExportText(metricsWriter)
}

func getMemory(zipWriter *zip.Writer) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "heap.tar.gz")
	if err != nil {
		return err
	}
	return pprof.WriteHeapProfile(w)
}

func getCPU(ctx context.Context, zipWriter *zip.Writer, duration time.Duration) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "cpu.tar.gz")
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(w); err != nil {
		return err
	}
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
	pprof.StopCPUProfile()
	return nil
}

func getBlock(zipWriter *zip.Writer) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "block.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("block")
	return p.WriteTo(w, 0)
}

func getMutex(zipWriter *zip.Writer) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "mutex.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("mutex")
	return p.WriteTo(w, 0)
}

func getGoroutines(zipWriter *zip.Writer) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "goroutine.txt")
	if err != nil {
		return err
	}
	p := pprof.Lookup("goroutine")
	return p.WriteTo(w, 2)
}

func getLogs(zipWriter *zip.Writer) error {
	if err := getLogFile(zipWriter, "central.log", logging.LoggingPath); err != nil {
		return err
	}
	return nil
}

func getLogFile(zipWriter *zip.Writer, targetPath string, sourcePath string) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, targetPath)
	if err != nil {
		return err
	}

	logFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, logFile)
	return err
}

func getVersion(ctx context.Context, zipWriter *zip.Writer) error {
	versions := buildVersions(ctx)

	return addJSONToZip(zipWriter, "versions.json", versions)
}

func writeTelemetryData(zipWriter *zip.Writer, telemetryInfo *data.TelemetryData) error {
	if telemetryInfo == nil {
		return errors.New("no telemetry data provided")
	}

	return addJSONToZip(zipWriter, "telemetry-data.json", telemetryInfo)
}

type dbExtension struct {
	ExtensionName    string `json:"ExtensionName"`
	ExtensionVersion string `json:"ExtensionVersion"`
}

// centralDBDiagnosticData represents a collection of various pieces of central db config information.
type centralDBDiagnosticData struct {
	// The Database versioning needs to be added by the caller due to scoping issues of config availabilty
	Database              string        `json:"Database,omitempty"`
	DatabaseClientVersion string        `json:"DatabaseClientVersion,omitempty"`
	DatabaseServerVersion string        `json:"DatabaseServerVersion,omitempty"`
	DatabaseExtensions    []dbExtension `json:"DatabaseExtensions,omitempty"`
	DatabaseConnectString string        `json:"DatabaseConnectString,omitempty"`
}

func getCentralDBData(ctx context.Context, zipWriter *zip.Writer) error {
	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Warnf("Could not parse postgres config: %v", err)
		return err
	}

	db := globaldb.GetPostgres()
	dbDiagnosticData := buildDBDiagnosticData(ctx, dbConfig, db)
	if err := addJSONToZip(zipWriter, "central-db.json", dbDiagnosticData); err != nil {
		return err
	}
	statements := stats.GetPGStatStatements(ctx, db, pgStatStatementsMax)
	if statements.Error != "" {
		log.Errorf("error retrieving pg_stat_statements: %s", statements.Error)
	}
	return addJSONToZip(zipWriter, "central-db-pg-stats.json", statements)
}

func (s *serviceImpl) getLogImbue(ctx context.Context, zipWriter *zip.Writer) error {
	w, err := zipWriterWithCurrentTimestamp(zipWriter, "logimbue-data.json")
	if err != nil {
		return err
	}
	logs, err := s.store.GetAll(ctx)
	if err != nil {
		return err
	}
	err = writer.WriteLogs(w, logs)
	return err
}

func (s *serviceImpl) getAuthProviders(_ context.Context) (interface{}, error) {
	authProviders := s.authProviderRegistry.GetProviders(nil, nil)

	var storageAuthProviders []*storage.AuthProvider
	for _, authProvider := range authProviders {
		storageAuthProviders = append(storageAuthProviders, authProvider.StorageView())
	}

	return storageAuthProviders, nil
}

func (s *serviceImpl) getGroups(_ context.Context) (interface{}, error) {
	accessGroupsCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	return s.groupDataStore.GetAll(accessGroupsCtx)
}

type diagResolvedRole struct {
	Role          *storage.Role              `json:"role,omitempty"`
	PermissionSet map[string]string          `json:"permission_set,omitempty"`
	AccessScope   *storage.SimpleAccessScope `json:"access_scope,omitempty"`
}

func (s *serviceImpl) getRoles(_ context.Context) (interface{}, error) {
	accessRolesCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	roles, errGetRoles := s.roleDataStore.GetAllRoles(accessRolesCtx)
	if errGetRoles != nil {
		return nil, errGetRoles
	}

	var resolvedRoles []*diagResolvedRole
	for _, role := range roles {
		diagRole := diagResolvedRole{
			Role: role,
		}

		if resolvedRole, err := s.roleDataStore.GetAndResolveRole(accessRolesCtx, role.Name); err == nil && resolvedRole != nil {
			// Get better formatting of permission sets.
			diagRole.PermissionSet = map[string]string{}
			for permName, accessRight := range resolvedRole.GetPermissions() {
				diagRole.PermissionSet[permName] = accessRight.String()
			}
			diagRole.AccessScope = resolvedRole.GetAccessScope()
		}

		resolvedRoles = append(resolvedRoles, &diagRole)
	}

	return resolvedRoles, nil
}

func (s *serviceImpl) getNotifiers(_ context.Context) (interface{}, error) {
	accessNotifierCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	return s.notifierDataStore.GetScrubbedNotifiers(accessNotifierCtx)
}

func (s *serviceImpl) getConfig(_ context.Context) (interface{}, error) {
	accessConfigCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	return s.configDataStore.GetConfig(accessConfigCtx)
}

// CustomRoutes returns route-handler pairs to be served on HTTP port.
func (s *serviceImpl) CustomRoutes() []routes.CustomRoute {
	customRoutes := []routes.CustomRoute{
		{
			Route:         "/debug/dump",
			Authorizer:    user.With(permissions.View(resources.Administration)),
			ServerHandler: http.HandlerFunc(s.getDebugDump),
			Compression:   true,
		},
		{
			Route:         "/api/extensions/diagnostics",
			Authorizer:    user.With(permissions.View(resources.Administration)),
			ServerHandler: http.HandlerFunc(s.getDiagnosticDump),
			Compression:   true,
		},
		{
			Route:         "/debug/versions.json",
			Authorizer:    user.With(permissions.View(resources.Administration)),
			ServerHandler: http.HandlerFunc(s.getVersionsJSON),
		},
	}

	return customRoutes
}

type debugDumpOptions struct {
	logs              logsMode
	telemetryMode     telemetryMode
	withCPUProfile    bool
	withLogImbue      bool
	withAccessControl bool
	withNotifiers     bool
	withCentral       bool
	clusters          []string
	since             time.Time
}

func (s *serviceImpl) writeZippedDebugDump(ctx context.Context, w http.ResponseWriter, filename string, opts debugDumpOptions) {
	debugDumpCtx, cancel := context.WithTimeout(ctx, debugDumpHardTimeout)
	defer cancel()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	zipWriter := zip.NewWriter(w)

	if err := getVersion(ctx, zipWriter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if opts.withCentral {
		if err := zipPrometheusMetrics(zipWriter, "metrics-1"); err != nil {
			log.Error(err)
		}

		if err := getMemory(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getGoroutines(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getBlock(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getMutex(zipWriter); err != nil {
			log.Error(err)
		}

		if opts.withCPUProfile {
			if err := getCPU(debugDumpCtx, zipWriter, cpuProfileDuration); err != nil {
				log.Error(err)
			}

			if err := zipPrometheusMetrics(zipWriter, "metrics-2"); err != nil {
				log.Error(err)
			}
		}

		if err := getCentralDBData(ctx, zipWriter); err != nil {
			log.Error(err)
		}

		log.Info("Finished writing Central data to diagnostic bundle")
	}

	if opts.logs == fullK8sIntrospectionData {
		if err := s.getK8sDiagnostics(debugDumpCtx, zipWriter, opts); err != nil {
			log.Errorf("Could not get K8s diagnostics: %+q", err)
			opts.logs = localLogs // fallback to local logs
		}
		if err := s.pullSensorMetrics(debugDumpCtx, zipWriter, opts); err != nil {
			log.Errorf("Could not get sensor metrics: %+q", err)
		}
	}

	if s.telemetryGatherer != nil && opts.telemetryMode > noTelemetry {
		telemetryData := s.telemetryGatherer.Gather(debugDumpCtx, opts.telemetryMode >= telemetryCentralAndSensors, opts.withCentral)
		if err := writeTelemetryData(zipWriter, telemetryData); err != nil {
			log.Error(err)
		}
	}

	if opts.withAccessControl {
		fetchAndAddJSONToZip(debugDumpCtx, zipWriter, "auth-providers.json", s.getAuthProviders)
		fetchAndAddJSONToZip(debugDumpCtx, zipWriter, "auth-provider-groups.json", s.getGroups)
		fetchAndAddJSONToZip(debugDumpCtx, zipWriter, "access-control-roles.json", s.getRoles)
	}

	if opts.withNotifiers {
		fetchAndAddJSONToZip(debugDumpCtx, zipWriter, "notifiers.json", s.getNotifiers)
	}

	fetchAndAddJSONToZip(debugDumpCtx, zipWriter, "system-configuration.json", s.getConfig)

	// Get logs last to also catch logs made during creation of diag bundle.
	if opts.withCentral && opts.logs == localLogs {
		if err := getLogs(zipWriter); err != nil {
			log.Error(err)
		}
	}

	if opts.withCentral && opts.withLogImbue {
		if err := s.getLogImbue(debugDumpCtx, zipWriter); err != nil {
			log.Error(err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		log.Error(err)
	}
}

func (s *serviceImpl) getVersionsJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %q, only GET requests are allowed", r.Method)
		return
	}

	versions := buildVersions(r.Context())

	versionsJSON, err := json.Marshal(&versions)
	if err != nil {
		httputil.WriteErrorf(w, http.StatusInternalServerError, "could not marshal version info to JSON: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(versionsJSON)))
	_, _ = w.Write(versionsJSON)
}

// getDebugDump aims to be a more involved version of getDiagnosticDump. For
// instance, it records CPU profile live. Also getDebugDump focuses primarily
// on Central and might not include all we know about secured clusters.
func (s *serviceImpl) getDebugDump(w http.ResponseWriter, r *http.Request) {
	opts := debugDumpOptions{
		logs:              localLogs,
		withCPUProfile:    true,
		withLogImbue:      true,
		withAccessControl: true,
		withNotifiers:     true,
		withCentral:       env.EnableCentralDiagnostics.BooleanSetting(),
		telemetryMode:     noTelemetry,
	}

	query := r.URL.Query()
	for _, p := range query["logs"] {
		v, err := strconv.ParseBool(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid log value: %q\n", p)
			return
		}
		if v {
			opts.logs = localLogs
		} else {
			opts.logs = noLogs
		}
	}

	telemetryModeStr := query.Get("telemetry")
	if telemetryModeStr != "" {
		var err error
		telemetryModeInt, err := strconv.Atoi(telemetryModeStr)
		opts.telemetryMode = telemetryMode(telemetryModeInt)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid telemetry mode value: %q\n", telemetryModeStr)
			return
		}
	}
	filename := time.Now().Format("stackrox_debug_2006_01_02_15_04_05.zip")

	s.writeZippedDebugDump(r.Context(), w, filename, opts)
}

// getDiagnosticDump aims to provide a snapshot of some state information for
// triaging. The size and download times of this dump shall stay reasonable.
func (s *serviceImpl) getDiagnosticDump(w http.ResponseWriter, r *http.Request) {
	s.getDiagnosticDumpWithCentral(w, r, env.EnableCentralDiagnostics.BooleanSetting())
}

func (s *serviceImpl) getDiagnosticDumpWithCentral(w http.ResponseWriter, r *http.Request, withCentral bool) {
	filename := time.Now().Format("stackrox_diagnostic_2006_01_02_15_04_05.zip")

	opts := debugDumpOptions{
		logs:              fullK8sIntrospectionData,
		telemetryMode:     telemetryCentralAndSensors,
		withCPUProfile:    false,
		withLogImbue:      true,
		withAccessControl: true,
		withCentral:       withCentral,
		withNotifiers:     true,
	}

	err := getOptionalQueryParams(&opts, r.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	log.Infof("Started writing diagnostic bundle %q with options: %+v", filename, opts)

	s.writeZippedDebugDump(r.Context(), w, filename, opts)
}

func getOptionalQueryParams(opts *debugDumpOptions, u *url.URL) error {
	values := u.Query()

	clusters := values["cluster"]
	if len(clusters) > 0 {
		opts.clusters = clusters
	}

	timeSince := values.Get("since")
	if timeSince != "" {
		t, err := time.Parse(layout, timeSince)
		if err != nil {
			return errors.Wrapf(err, "invalid timestamp value: %q\n", t)
		}
		opts.since = t
	} else {
		opts.since = time.Now().Add(-logWindow)
	}
	return nil
}

func buildVersions(ctx context.Context) version.Versions {
	versions := version.GetAllVersionsDevelopment()
	if buildinfo.ReleaseBuild {
		versions = version.GetAllVersionsUnified()
	}
	// Add the database version if Postgres
	versions.Database = "PostgresDB"
	versions.DatabaseServerVersion = globaldb.GetPostgresVersion(ctx, globaldb.GetPostgres())

	return versions
}
