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
	groupDS "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/central/logimbue/store"
	"github.com/stackrox/rox/central/logimbue/writer"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/prometheusutil"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

type logsMode int

const (
	cpuProfileDuration = 30 * time.Second

	noLogs logsMode = iota
	localLogs
	fullK8sIntrospectionData

	centralClusterPrefix = "_central-cluster"

	metricsPullTimeout     = 10 * time.Second
	diagnosticsPullTimeout = 10 * time.Second
	layout                 = "2006-01-02T15:04:05.000Z"
	logWindow              = 20 * time.Minute
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DebugLogs)): {
			"/v1.DebugService/GetLogLevel",
			"/v1.DebugService/StreamAuthzTraces",
		},
		user.With(permissions.Modify(resources.DebugLogs)): {
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
func (s *serviceImpl) GetLogLevel(ctx context.Context, req *v1.GetLogLevelRequest) (*v1.LogLevelResponse, error) {
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
func (s *serviceImpl) SetLogLevel(ctx context.Context, req *v1.LogLevelRequest) (*types.Empty, error) {
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
	w, err := zipWriter.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "unable to create zip file %q", fileName)
	}

	jsonEnc := json.NewEncoder(w)
	jsonEnc.SetIndent("", "  ")

	return jsonEnc.Encode(jsonObj)
}

func zipPrometheusMetrics(zipWriter *zip.Writer, name string) error {
	metricsWriter, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	return prometheusutil.ExportText(metricsWriter)
}

func getMemory(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("heap.tar.gz")
	if err != nil {
		return err
	}
	return pprof.WriteHeapProfile(w)
}

func getCPU(ctx context.Context, zipWriter *zip.Writer, duration time.Duration) error {
	w, err := zipWriter.Create("cpu.tar.gz")
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
	w, err := zipWriter.Create("block.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("block")
	return p.WriteTo(w, 0)
}

func getMutex(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("mutex.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("mutex")
	return p.WriteTo(w, 0)
}

func getGoroutines(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("goroutine.txt")
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
	if err := getLogFile(zipWriter, "migration.log", logging.PersistentLoggingPath); err != nil {
		return err
	}
	return nil
}

func getLogFile(zipWriter *zip.Writer, targetPath string, sourcePath string) error {
	w, err := zipWriter.Create(targetPath)
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

func getVersion(zipWriter *zip.Writer) error {
	versions := version.GetAllVersionsDevelopment()
	if buildinfo.ReleaseBuild {
		versions = version.GetAllVersionsUnified()
	}

	return addJSONToZip(zipWriter, "versions.json", versions)
}

func writeTelemetryData(zipWriter *zip.Writer, telemetryInfo *data.TelemetryData) error {
	if telemetryInfo == nil {
		return errors.New("no telemetry data provided")
	}

	return addJSONToZip(zipWriter, "telemetry-data.json", telemetryInfo)
}

func (s *serviceImpl) getLogImbue(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("logimbue-data.json")
	if err != nil {
		return err
	}
	logs, err := s.store.GetLogs()
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
			sac.ResourceScopeKeys(resources.Group)))

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
			sac.ResourceScopeKeys(resources.Role)))

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
			sac.ResourceScopeKeys(resources.Notifier)))

	return s.notifierDataStore.GetScrubbedNotifiers(accessNotifierCtx)
}

func (s *serviceImpl) getConfig(_ context.Context) (interface{}, error) {
	accessConfigCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Config)))

	return s.configDataStore.GetConfig(accessConfigCtx)
}

// DebugHandler is an HTTP handler that outputs debugging information
func (s *serviceImpl) CustomRoutes() []routes.CustomRoute {
	customRoutes := []routes.CustomRoute{
		{
			Route:         "/debug/dump",
			Authorizer:    user.With(permissions.View(resources.DebugLogs)),
			ServerHandler: http.HandlerFunc(s.getDebugDump),
		},
		{
			Route:         "/api/extensions/diagnostics",
			Authorizer:    user.With(permissions.View(resources.DebugLogs)),
			ServerHandler: http.HandlerFunc(s.getDiagnosticDump),
		},
		{
			Route:         "/debug/versions.json",
			Authorizer:    user.With(permissions.View(resources.DebugLogs)),
			ServerHandler: http.HandlerFunc(s.getVersionsJSON),
		},
	}

	return customRoutes
}

type debugDumpOptions struct {
	logs logsMode
	// telemetryMode specifies how to use sensor/central telemetry to gather diagnostics.
	// 0 - don't collect any telemetry data
	// 1 - collect telemetry data for central only
	// 2 - collect telemetry data from sensors and central
	telemetryMode     int
	withCPUProfile    bool
	withLogImbue      bool
	withAccessControl bool
	withNotifiers     bool
	clusters          []string
	since             time.Time
}

func (s *serviceImpl) writeZippedDebugDump(ctx context.Context, w http.ResponseWriter, filename string, opts debugDumpOptions) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	zipWriter := zip.NewWriter(w)

	if err := getVersion(zipWriter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
		if err := getCPU(ctx, zipWriter, cpuProfileDuration); err != nil {
			log.Error(err)
		}

		if err := zipPrometheusMetrics(zipWriter, "metrics-2"); err != nil {
			log.Error(err)
		}
	}

	if opts.logs == fullK8sIntrospectionData {
		if err := s.getK8sDiagnostics(ctx, zipWriter, opts); err != nil {
			log.Error(err)
			opts.logs = localLogs // fallback to local logs
		}
		if err := s.pullSensorMetrics(ctx, zipWriter, opts); err != nil {
			log.Error(err)
		}
	}

	if s.telemetryGatherer != nil && opts.telemetryMode > 0 {
		telemetryData := s.telemetryGatherer.Gather(ctx, opts.telemetryMode >= 2)
		if err := writeTelemetryData(zipWriter, telemetryData); err != nil {
			log.Error(err)
		}
	}

	if opts.withAccessControl {
		fetchAndAddJSONToZip(ctx, zipWriter, "auth-providers.json", s.getAuthProviders)
		fetchAndAddJSONToZip(ctx, zipWriter, "auth-provider-groups.json", s.getGroups)
		fetchAndAddJSONToZip(ctx, zipWriter, "access-control-roles.json", s.getRoles)
	}

	if opts.withNotifiers {
		fetchAndAddJSONToZip(ctx, zipWriter, "notifiers.json", s.getNotifiers)
	}

	fetchAndAddJSONToZip(ctx, zipWriter, "system-configuration.json", s.getConfig)

	// Get logs last to also catch logs made during creation of diag bundle.
	if opts.logs == localLogs {
		if err := getLogs(zipWriter); err != nil {
			log.Error(err)
		}
	}

	if opts.withLogImbue {
		if err := s.getLogImbue(zipWriter); err != nil {
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

	versions := version.GetAllVersionsDevelopment()
	if buildinfo.ReleaseBuild {
		versions = version.GetAllVersionsUnified()
	}
	versionsJSON, err := json.Marshal(&versions)
	if err != nil {
		httputil.WriteErrorf(w, http.StatusInternalServerError, "could not marshal version info to JSON: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(versionsJSON)))
	_, _ = w.Write(versionsJSON)
}

func (s *serviceImpl) getDebugDump(w http.ResponseWriter, r *http.Request) {
	opts := debugDumpOptions{
		logs:              localLogs,
		withCPUProfile:    true,
		withLogImbue:      true,
		withAccessControl: true,
		withNotifiers:     true,
		telemetryMode:     0,
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
		opts.telemetryMode, err = strconv.Atoi(telemetryModeStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid telemetry mode value: %q\n", telemetryModeStr)
			return
		}
	}
	filename := time.Now().Format("stackrox_debug_2006_01_02_15_04_05.zip")

	s.writeZippedDebugDump(r.Context(), w, filename, opts)
}

func (s *serviceImpl) getDiagnosticDump(w http.ResponseWriter, r *http.Request) {
	filename := time.Now().Format("stackrox_diagnostic_2006_01_02_15_04_05.zip")

	opts := debugDumpOptions{
		logs:              fullK8sIntrospectionData,
		telemetryMode:     2,
		withCPUProfile:    false,
		withLogImbue:      true,
		withAccessControl: true,
		withNotifiers:     true,
	}

	err := getOptionalQueryParams(&opts, r.URL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

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
