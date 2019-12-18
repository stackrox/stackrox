package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	grpcPkg "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	cpuProfileDuration = 30 * time.Second
)

var (
	log = logging.LoggerForModule()

	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.DebugLogs)): {
			"/v1.DebugService/GetLogLevel",
			"/v1.DebugService/SetLogLevel",
		},
	})
)

// Service provides the interface to the gRPC service for debugging
type Service interface {
	grpcPkg.APIServiceWithCustomRoutes
	v1.DebugServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a Service that implements v1.DebugServiceServer
func New() Service {
	return &serviceImpl{}
}

type serviceImpl struct{}

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

	// If the request is global, then return all modules who have a log level that does not match the global level
	if len(req.GetModules()) == 0 {
		level := logging.GetGlobalLogLevel()
		resp.Level = logging.LabelForLevelOrInvalid(level)
		logging.ForEachLogger(func(l *logging.Logger) {
			moduleLevel := l.LogLevel()
			if moduleLevel != level {
				resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{Module: l.GetModule(), Level: l.GetLogLevel()})
			}
		})
		return resp, nil
	}

	loggers, unknownModules := logging.GetLoggersByModule(req.GetModules())
	if len(unknownModules) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}
	for _, l := range loggers {
		resp.ModuleLevels = append(resp.ModuleLevels, &v1.ModuleLevel{Module: l.GetModule(), Level: l.GetLogLevel()})
	}
	return resp, nil
}

// SetLogLevel implements v1.DebugServiceServer, and it sets the log level for StackRox services.
func (s *serviceImpl) SetLogLevel(ctx context.Context, req *v1.LogLevelRequest) (*types.Empty, error) {
	levelStr := req.GetLevel()
	levelInt, ok := logging.LevelForLabel(levelStr)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown log level %s", levelStr)
	}

	// If this is a global request, then set the global level and return
	if len(req.GetModules()) == 0 {
		logging.SetGlobalLogLevel(levelInt)
		return &types.Empty{}, nil
	}

	loggers, unknownModules := logging.GetLoggersByModule(req.GetModules())
	if len(unknownModules) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown module(s): %s", strings.Join(unknownModules, ", "))
	}

	for _, logger := range loggers {
		logger.SetLogLevel(levelInt)
	}
	return &types.Empty{}, nil
}

func zipPrometheusMetrics(zipWriter *zip.Writer, name string) error {
	metricsWriter, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	return getPrometheusMetrics(metricsWriter)
}

func getPrometheusMetrics(w io.Writer) error {
	g := prometheus.DefaultGatherer
	mfs, err := g.Gather()
	if err != nil {
		// Failed to gather metrics.  Write the error to the file and return.  If we fail to write the error to the
		// file return both errors.
		_, writeErr := fmt.Fprintf(w, "# ERROR: %s\n", err.Error())
		return errorhelpers.NewErrorListWithErrors("gathering prometheus metrics", []error{err, writeErr}).ToError()
	}
	for _, mf := range mfs {
		if _, err := expfmt.MetricFamilyToText(w, mf); err != nil {
			// Failed to write a metric family.  Write the error to the file and continue
			if _, writeErr := w.Write([]byte(fmt.Sprintf("# ERROR: %s\n", err.Error()))); writeErr != nil {
				// Failed to write the error to the file.  Return both errors.
				errList := errorhelpers.NewErrorListWithErrors(fmt.Sprintf("writing metric family %s", mf.GetName()), []error{err, writeErr})
				return errList.ToError()
			}

		}
	}
	return nil
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
	w, err := zipWriter.Create("central.log")
	if err != nil {
		return err
	}

	logFile, err := os.Open(logging.LoggingPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, logFile)
	return err
}

func getVersion(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("versions.json")
	if err != nil {
		return err
	}
	versions := version.GetAllVersions()
	data, err := json.Marshal(versions)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

// DebugHandler is an HTTP handler that outputs debugging information
func (s *serviceImpl) CustomRoutes() []routes.CustomRoute {
	customRoutes := []routes.CustomRoute{
		{
			Route:         "/debug/dump",
			Authorizer:    user.With(permissions.View(resources.DebugLogs)),
			ServerHandler: http.HandlerFunc(s.getDebugDump),
		},
	}

	if features.Telemetry.Enabled() {
		customRoutes = append(customRoutes,
			routes.CustomRoute{
				Route:         "/api/extensions/diagnostics",
				Authorizer:    user.With(permissions.View(resources.DebugLogs)),
				ServerHandler: http.HandlerFunc(s.getDiagnosticDump),
			},
		)
	}

	return customRoutes
}

func writeZippedDebugDump(ctx context.Context, w http.ResponseWriter, filename string, withLogs, withCPUProfile bool) {
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

	if withCPUProfile {
		if err := getCPU(ctx, zipWriter, cpuProfileDuration); err != nil {
			log.Error(err)
		}

		if err := zipPrometheusMetrics(zipWriter, "metrics-2"); err != nil {
			log.Error(err)
		}
	}

	if withLogs {
		if err := getLogs(zipWriter); err != nil {
			log.Error(err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		log.Error(err)
	}
}

func (s *serviceImpl) getDebugDump(w http.ResponseWriter, r *http.Request) {
	withLogs := true
	for _, p := range r.URL.Query()["logs"] {
		v, err := strconv.ParseBool(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "invalid log value: %q\n", p)
			return
		}
		withLogs = v
	}

	filename := time.Now().Format("stackrox_debug_2006_01_02_15_04_05.zip")

	writeZippedDebugDump(r.Context(), w, filename, withLogs, true)
}

func (s *serviceImpl) getDiagnosticDump(w http.ResponseWriter, r *http.Request) {
	filename := time.Now().Format("stackrox_diagnostic_2006_01_02_15_04_05.zip")

	writeZippedDebugDump(r.Context(), w, filename, true, false)
}
