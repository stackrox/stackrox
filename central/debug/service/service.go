package service

import (
	"bytes"
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
	concPool "github.com/sourcegraph/conc/pool"
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
	"github.com/stackrox/rox/pkg/concurrency"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			"/v1.DebugService/ResetDBStats",
		},
	})

	mainClusterConfig = k8sintrospect.DefaultConfig()
)

func init() {
	mainClusterConfig.PathPrefix = centralClusterPrefix
	// For the main cluster (i.e. the collection for the Central cluster) we explicitly ignore the log file limits.
	// The limitation is not required since the GRPC message isn't affected by it, and has proven to be unhelpful
	// in cases where the logs of Central are quite big (e.g. in larger scale environments).
	mainClusterConfig.IgnoreLogLimits = true
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

// ResetDBStats resets pg_stat_statements in order to allow new metrics to be accumulated.
func (s *serviceImpl) ResetDBStats(ctx context.Context, _ *v1.Empty) (*v1.Empty, error) {
	if pgconfig.IsExternalDatabase() {
		return nil, status.Error(codes.InvalidArgument, "cannot reset DB stats on an external database")
	}
	err := stats.ResetPGStatStatements(ctx, globaldb.GetPostgres())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

// InternalDiagnosticsHandler returns handler to be served on "cluster-internal" port.
// Cluster-internal port is not exposed via k8s Service and only accessible to callers with k8s/Openshift cluster access.
// This handler shouldn't be exposed to other callers as it has no authorization and can elevate customer permissions.
func (s *serviceImpl) InternalDiagnosticsHandler() http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, r *http.Request) {
		// Adding scope checker as no authorizer is used, ergo no identity in context by default.
		ctx := sac.WithGlobalAccessScopeChecker(r.Context(),
			sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS)))
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
		return nil, errors.Wrapf(errox.InvalidArgs, "Unknown module(s): %s",
			strings.Join(unknownModules, ", "))
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
		return nil, errors.Wrapf(errox.InvalidArgs, "Unknown module(s): %s",
			strings.Join(unknownModules, ", "))
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

func fetchAndAddJSONToZip(ctx context.Context, zipWriter *zipWriter, fileName string,
	fetchData func(ctx context.Context) (interface{}, error)) {
	jsonObj, errFetchData := fetchData(ctx)
	if errFetchData != nil {
		log.Error(errFetchData)

		return
	}

	if errAddToZip := addJSONToZip(zipWriter, fileName, jsonObj); errAddToZip != nil {
		log.Error(errAddToZip)
	}
}

func addJSONToZip(zipWriter *zipWriter, fileName string, jsonObj interface{}) error {
	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock(fileName)
	if err != nil {
		return errors.Wrapf(err, "unable to create zip file %q", fileName)
	}

	jsonEnc := json.NewEncoder(w)
	jsonEnc.SetIndent("", "  ")

	return jsonEnc.Encode(jsonObj)
}

func zipPrometheusMetrics(ctx context.Context, zipWriter *zipWriter, name string) error {
	// Write to the buffer first instead of directly to the zip writer, this way we hold the lock _only_ for the copy
	// time.
	buf := &bytes.Buffer{}
	if err := prometheusutil.ExportText(ctx, buf); err != nil {
		return err
	}
	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	metricsWriter, err := zipWriter.writerWithCurrentTimestampNoLock(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(metricsWriter, buf)
	return err
}

func getMemory(zipWriter *zipWriter) error {
	// Write to the buffer first instead of directly to the zip writer, this way we hold the lock _only_ for the copy
	// time.
	buf := &bytes.Buffer{}
	if err := pprof.WriteHeapProfile(buf); err != nil {
		return err
	}
	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock("heap.pb.gz")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

func getCPU(ctx context.Context, zipWriter *zipWriter, duration time.Duration) error {
	// Write to the buffer first instead of directly to the zip writer, this way we hold the lock _only_ for the copy
	// time.
	buf := &bytes.Buffer{}
	if err := pprof.StartCPUProfile(buf); err != nil {
		return err
	}
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
	pprof.StopCPUProfile()
	if concurrency.IsDone(ctx) {
		return nil
	}

	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock("cpu.pb.gz")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

func getMutex(zipWriter *zipWriter) error {
	// Write to the buffer first instead of directly to the zip writer, this way we hold the lock _only_ for the copy
	// time.
	buf := &bytes.Buffer{}
	p := pprof.Lookup("mutex")
	if err := p.WriteTo(buf, 0); err != nil {
		return err
	}

	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock("mutex.pb.gz")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

func getGoroutines(zipWriter *zipWriter) error {
	// Write to the buffer first instead of directly to the zip writer, this way we hold the lock _only_ for the copy
	// time.
	buf := &bytes.Buffer{}
	p := pprof.Lookup("goroutine")
	if err := p.WriteTo(buf, 2); err != nil {
		return err
	}
	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock("goroutine.txt")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

func getLogs(zipWriter *zipWriter) error {
	if err := getLogFile(zipWriter, "central.log", logging.LoggingPath); err != nil {
		return err
	}
	return nil
}

func getLogFile(zipWriter *zipWriter, targetPath string, sourcePath string) error {
	logFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock(targetPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, logFile)
	return err
}

func getVersion(ctx context.Context, zipWriter *zipWriter) error {
	versions := buildVersions(ctx)

	return addJSONToZip(zipWriter, "versions.json", versions)
}

func writeTelemetryData(zipWriter *zipWriter, telemetryInfo *data.TelemetryData) error {
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

func getCentralDBData(ctx context.Context, zipWriter *zipWriter) error {
	_, dbConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		log.Warnw("Could not parse postgres config", logging.Err(err))
		return err
	}

	db := globaldb.GetPostgres()
	dbDiagnosticData := buildDBDiagnosticData(ctx, dbConfig, db)
	if err := addJSONToZip(zipWriter, "central-db.json", dbDiagnosticData); err != nil {
		return err
	}
	statements := stats.GetPGStatStatements(ctx, db, pgStatStatementsMax)
	if statements.Error != "" {
		log.Errorw("error retrieving pg_stat_statements", logging.Err(errors.New(statements.Error)))
	}
	if err := addJSONToZip(zipWriter, "central-db-pg-stats.json", statements); err != nil {
		return err
	}

	// Get the analyze stats
	analyzeStats := stats.GetPGAnalyzeStats(ctx, db, pgStatStatementsMax)
	if analyzeStats.Error != "" {
		log.Errorw("error retrieving pg_stat_statements", logging.Err(errors.New(analyzeStats.Error)))
	}
	if err := addJSONToZip(zipWriter, "central-db-pg-analyze-stats.json", analyzeStats); err != nil {
		return err
	}

	// Get the dead tuple stats
	tuples := stats.GetPGTupleStats(ctx, db, pgStatStatementsMax)
	if tuples.Error != "" {
		log.Errorw("error retrieving pg_stat_user_tables", logging.Err(errors.New(tuples.Error)))
	}
	return addJSONToZip(zipWriter, "central-db-pg-tuples.json", tuples)
}

func (s *serviceImpl) getLogImbue(ctx context.Context, zipWriter *zipWriter) error {
	logs, err := s.store.GetAll(ctx)
	if err != nil {
		return err
	}

	zipWriter.LockWrite()
	defer zipWriter.UnlockWrite()
	w, err := zipWriter.writerWithCurrentTimestampNoLock("logimbue-data.json")
	if err != nil {
		return err
	}

	return writer.WriteLogs(w, logs)
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

		if resolvedRole, err := s.roleDataStore.GetAndResolveRole(accessRolesCtx,
			role.Name); err == nil && resolvedRole != nil {
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

func (s *serviceImpl) writeZippedDebugDump(ctx context.Context, w http.ResponseWriter, filename string,
	opts debugDumpOptions) {
	debugDumpCtx, cancel := context.WithTimeout(ctx, debugDumpHardTimeout)
	defer cancel()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	zipWriter := newZipWriter(w)

	// Defer closing the zip writer since we short-circuit in case of context cancellations.
	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Errorw("Failed closing the ZIP writer", logging.Err(err))
		}
	}()

	if err := getVersion(ctx, zipWriter); err != nil {
		log.Errorw("Failed getting Central's version", logging.Err(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	diagBundleTasks := concPool.New().WithContext(debugDumpCtx)
	if opts.withCentral {
		diagBundleTasks.Go(func(ctx context.Context) error {
			return zipPrometheusMetrics(ctx, zipWriter,
				"metrics-1")
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			return getMemory(zipWriter)
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			return getGoroutines(zipWriter)
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			return getMutex(zipWriter)
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			return getCentralDBData(ctx, zipWriter)
		})
		if opts.withCPUProfile {
			diagBundleTasks.Go(func(ctx context.Context) error {
				return getCPU(ctx, zipWriter, cpuProfileDuration)
			})
			diagBundleTasks.Go(func(ctx context.Context) error {
				return zipPrometheusMetrics(ctx, zipWriter, "metrics-2")
			})
		}
	}

	var failureDuringDiagnostics bool
	if opts.logs == fullK8sIntrospectionData {
		diagBundleTasks.Go(func(ctx context.Context) error {
			// In case we fail to fetch K8S diagnostics, which also includes the collection of logs for Central,
			// ensure we later on attempt to collect local logs as a safety net to at the very least have the
			// Central logs contained in the diagnostic bundle.
			if err := s.getK8sDiagnostics(ctx, zipWriter, opts); err != nil {
				failureDuringDiagnostics = true
				return err
			}
			return nil
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			if err := s.pullSensorMetrics(ctx, zipWriter, opts); err != nil {
				return err
			}
			return nil
		})
	}
	if s.telemetryGatherer != nil && opts.telemetryMode > noTelemetry {
		diagBundleTasks.Go(func(ctx context.Context) error {
			telemetryData := s.telemetryGatherer.Gather(ctx, opts.telemetryMode >= telemetryCentralAndSensors,
				opts.withCentral)
			return writeTelemetryData(zipWriter, telemetryData)
		})
	}
	if opts.withAccessControl {
		diagBundleTasks.Go(func(ctx context.Context) error {
			fetchAndAddJSONToZip(ctx, zipWriter, "auth-providers.json", s.getAuthProviders)
			return nil
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			fetchAndAddJSONToZip(ctx, zipWriter, "auth-provider-groups.json", s.getGroups)
			return nil
		})
		diagBundleTasks.Go(func(ctx context.Context) error {
			fetchAndAddJSONToZip(ctx, zipWriter, "access-control-roles.json", s.getRoles)
			return nil
		})
	}
	if opts.withNotifiers {
		diagBundleTasks.Go(func(ctx context.Context) error {
			fetchAndAddJSONToZip(ctx, zipWriter, "notifiers.json", s.getNotifiers)
			return nil
		})
	}
	diagBundleTasks.Go(func(ctx context.Context) error {
		fetchAndAddJSONToZip(ctx, zipWriter, "system-configuration.json", s.getConfig)
		return nil
	})
	if opts.withCentral && opts.withLogImbue {
		diagBundleTasks.Go(func(ctx context.Context) error {
			return s.getLogImbue(ctx, zipWriter)
		})
	}

	// Wait for the "all the rest" part of the tasks to construct the diagnostic bundle.
	// This also respects context cancellations and returns any potential errors that occurred.
	err := diagBundleTasks.Wait()
	if err != nil {
		// Short-circuit in case the context has been cancelled.
		if concurrency.IsDone(debugDumpCtx) {
			log.Warn("The context for collecting diagnostic bundle data has been cancelled")
			return
		}
		log.Errorw("Failures during gathering diagnostic bundle contents", logging.Err(err))
	}
	log.Info("Finished writing data to the diagnostic bundle")

	// Get logs last to also catch logs made during creation of diag bundle.
	if opts.withCentral && (opts.logs == localLogs || failureDuringDiagnostics) {
		if err := getLogs(zipWriter); err != nil {
			log.Error(err)
		}
	}
}

func (s *serviceImpl) getVersionsJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %q, only GET requests are allowed",
			r.Method)
		return
	}

	versions := buildVersions(r.Context())

	versionsJSON, err := json.Marshal(&versions)
	if err != nil {
		httputil.WriteErrorf(w, http.StatusInternalServerError, "could not marshal version info to JSON: %v",
			err)
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
		telemetryMode:     noTelemetry,
		withCPUProfile:    true,
		withLogImbue:      true,
		withAccessControl: true,
		withNotifiers:     true,
		withCentral:       env.EnableCentralDiagnostics.BooleanSetting(),
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
