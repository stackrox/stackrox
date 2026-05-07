package main

import (
	"context"
	"flag"
	"fmt"
	golog "log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/quay/claircore/toolkit/log"
	"github.com/quay/zlog/v2"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/continuousprofiling"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	grpcmetrics "github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/internal/version"
	"github.com/stackrox/rox/scanner/matcher"
	"github.com/stackrox/rox/scanner/services"
	"golang.org/x/sys/unix"
	gogrpc "google.golang.org/grpc"
)

// Backends holds the backend engines the scanner may use depending on the
// configuration and mode in which it is running.
type Backends struct {
	// Indexer is the indexing backend.
	Indexer indexer.Indexer
	// Matcher is the vulnerability matching backend.
	Matcher matcher.Matcher
	// RemoteIndexer is the indexing backend located in a remote scanner instance.
	RemoteIndexer indexer.RemoteIndexer
}

func init() {
	// Set the http.DefaultTransport's Proxy function to one which reads from the proxy configuration file.
	// Note: http.DefaultClient uses http.DefaultTransport.
	if !proxy.UseWithDefaultTransport() {
		golog.Println("Failed to use proxy transport with default HTTP transport. Some proxy features may not work.")
	}

	memlimit.SetMemoryLimit()
}

func main() {
	configPath := flag.String("conf", "", "Path to scanner's configuration file.")
	printVersion := flag.Bool("version", false,
		"Print the build version tag and the vulnerability schema version, then exit.")
	flag.Parse()
	if *printVersion {
		fmt.Println(version.Version, version.VulnerabilityVersion)
		os.Exit(0)
	}
	cfg, err := config.Read(*configPath)
	if err != nil {
		golog.Fatalf("failed to load configuration %q: %v", *configPath, err)
	}

	// Create cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logging and setup context.
	err = initializeLogging(cfg.LogLevel)
	if err != nil {
		golog.Fatalf("failed to initialize logging: %v", err)
	}

	if err := continuousprofiling.SetupClient(continuousprofiling.DefaultConfig()); err != nil {
		slog.ErrorContext(ctx, "unable to start continuous profiling", "reason", err)
	}

	slog.InfoContext(ctx, "starting scanner", "version", version.Version, "build_flavor", buildinfo.BuildFlavor)

	// If certs was specified, configure the identity environment.
	if p := cfg.MTLS.CertsDir; p != "" {
		slog.InfoContext(ctx, "identity certificates filename prefix changed", "certs_prefix", p)
		utils.CrashOnError(os.Setenv(mtls.CAFileEnvName, filepath.Join(p, mtls.CACertFileName)))
		utils.CrashOnError(os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(p, mtls.CAKeyFileName)))
		utils.CrashOnError(os.Setenv(mtls.CertFilePathEnvName, filepath.Join(p, mtls.ServiceCertFileName)))
		utils.CrashOnError(os.Setenv(mtls.KeyFileEnvName, filepath.Join(p, mtls.ServiceKeyFileName)))
	}

	//  If proxy path is set, periodically check for updates.
	if cfg.Proxy.ConfigDir != "" {
		slog.InfoContext(ctx, "proxy configured", "dir", cfg.Proxy.ConfigDir, "file", cfg.Proxy.ConfigFile)
		proxy.WatchProxyConfig(ctx, cfg.Proxy.ConfigDir, cfg.Proxy.ConfigFile, true)
	}

	features.LogFeatureFlags()
	// Initialize metrics and metrics server.
	metricsSrv := metrics.NewServer(metrics.ScannerSubsystem, metrics.NewTLSConfigurerFromEnv())
	metricsSrv.RunForever()
	defer metricsSrv.Stop(ctx)
	metrics.GatherThrottleMetricsForever(metrics.ScannerSubsystem.String())

	// Create backends.
	backends, err := createBackends(ctx, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create backends", "reason", err)
		os.Exit(1)
	}
	defer backends.Close(ctx)
	slog.InfoContext(ctx, "backends created")

	// Initialize gRPC API service.
	grpcSrv, err := createGRPCService(backends, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize gRPC", "reason", err)
		os.Exit(1)
	}
	grpcSrv.Start()
	defer grpcSrv.Stop()

	// Wait for signals.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, unix.SIGINT, unix.SIGTERM)
	sig := <-sigC
	slog.InfoContext(ctx, "signal received", "signal", sig.String())
}

func initializeLogging(logLevel slog.Level) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	h := zlog.NewHandler(os.Stdout, &zlog.Options{
		Level:      logLevel,
		ContextKey: log.AttrsKey,
		LevelKey:   log.LevelKey,
	})
	logger := slog.New(h).With("host", hostname)
	slog.SetDefault(logger)
	return nil
}

// createGRPCService creates a ready-to-start gRPC API instance and register its services.
func createGRPCService(backends *Backends, cfg *config.Config) (grpc.API, error) {
	// Create identity extractors.
	identityExtractor, err := service.NewExtractor()
	if err != nil {
		return nil, fmt.Errorf("identity extractor: %w", err)
	}

	// Custom routes: debugging.
	var customRoutes []routes.CustomRoute
	if !env.ContinuousProfiling.BooleanSetting() {
		customRoutes = make([]routes.CustomRoute, 0, len(routes.DebugRoutes)+
			len(backends.HealthRoutes()))
		for path, handler := range routes.DebugRoutes {
			customRoutes = append(customRoutes, routes.CustomRoute{
				Route:         path,
				Authorizer:    allow.Anonymous(),
				ServerHandler: handler,
				Compression:   true,
			})
		}
	} else {
		customRoutes = make([]routes.CustomRoute, 0, len(backends.HealthRoutes()))
	}

	// Custom routes: health checking.
	customRoutes = append(customRoutes, backends.HealthRoutes()...)

	// Create gRPC API service.
	grpcSrv := grpc.NewAPI(grpc.Config{
		CustomRoutes:       customRoutes,
		IdentityExtractors: []authn.IdentityExtractor{identityExtractor},
		GRPCMetrics:        grpcmetrics.NewGRPCMetrics(),
		HTTPMetrics:        grpcmetrics.NewHTTPMetrics(),
		UnaryInterceptors:  []gogrpc.UnaryServerInterceptor{version.UnaryServerInterceptor()},
		Endpoints: []*grpc.EndpointConfig{
			{
				ListenEndpoint: cfg.GRPCListenAddr,
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      false,
			},
			{
				ListenEndpoint: cfg.HTTPListenAddr,
				TLS:            verifier.NonCA{},
				ServeGRPC:      false,
				ServeHTTP:      true,
			},
		},
		// Setting this value causes the server to tell clients to GOAWAY after the specified duration (+/- some jitter).
		// This is to ensure clients account for server-side horizontal scaling.
		MaxConnectionAge: 2 * time.Minute,
	})

	// Register API services.
	grpcSrv.Register(backends.APIServices()...)

	return grpcSrv, nil
}

// createBackends creates the scanner backends.
func createBackends(ctx context.Context, cfg *config.Config) (*Backends, error) {
	var b Backends
	var err error
	if cfg.Indexer.Enable {
		slog.InfoContext(ctx, "indexer is enabled")
		b.Indexer, err = indexer.NewIndexer(ctx, cfg.Indexer)
		if err != nil {
			return nil, fmt.Errorf("indexer: %w", err)
		}
	} else {
		slog.InfoContext(ctx, "indexer is disabled")
	}
	if cfg.Matcher.Enable {
		slog.InfoContext(ctx, "matcher is enabled")
		if cfg.Matcher.RemoteIndexerEnabled {
			// Create a remote indexer only if the matcher was configured to use one.
			slog.InfoContext(ctx, "remote indexer is enabled")
			b.RemoteIndexer, err = indexer.NewRemoteIndexer(ctx, cfg.Matcher.IndexerAddr)
			if err != nil {
				return nil, fmt.Errorf("matcher: remote indexer: %w", err)
			}
		}
		b.Matcher, err = matcher.NewMatcher(ctx, cfg.Matcher, b.RemoteIndexer)
		if err != nil {
			return nil, fmt.Errorf("matcher: %w", err)
		}
	} else {
		slog.InfoContext(ctx, "matcher is disabled")
	}
	return &b, nil
}

// APIServices returns the list of the gRPC API services based on the configured backends.
func (b *Backends) APIServices() []grpc.APIService {
	var srvs []grpc.APIService
	if b.Indexer != nil {
		srvs = append(srvs, services.NewIndexerService(b.Indexer))
	}
	if b.Matcher != nil {
		// Set the index report getter to the remote indexer if available, otherwise the
		// local indexer. A nil getter is ok, see implementation.
		var getter indexer.ReportGetter
		getter = b.RemoteIndexer
		if getter == nil {
			getter = b.Indexer
		}
		srvs = append(srvs, services.NewMatcherService(b.Matcher, getter))
	}
	return srvs
}

// HealthCheck returns true if all configured backends are healthy and ready.
func (b *Backends) HealthCheck(ctx context.Context) bool {
	var checkList []func(context.Context) error
	if b.Indexer != nil {
		checkList = append(checkList, b.Indexer.Ready)
	}
	if b.Matcher != nil {
		checkList = append(checkList, b.Matcher.Ready)
	}
	for _, check := range checkList {
		if err := check(ctx); err != nil {
			slog.ErrorContext(ctx, "scanner is not ready", "reason", err)
			return false
		}
	}
	return true
}

// HealthRoutes returns HTTP routes for health checking the configured backends.
func (b *Backends) HealthRoutes() (r []routes.CustomRoute) {
	for n, h := range map[string]http.HandlerFunc{
		"/health/readiness": func(w http.ResponseWriter, r *http.Request) {
			st := http.StatusOK
			if !b.HealthCheck(r.Context()) {
				st = http.StatusServiceUnavailable
			}
			w.WriteHeader(st)
		},
	} {
		r = append(r, routes.CustomRoute{
			Route:         n,
			Authorizer:    allow.Anonymous(),
			ServerHandler: h,
		})
	}
	return r
}

// Close closes all the backends used by scanner.
func (b *Backends) Close(ctx context.Context) {
	type Closeable interface {
		Close(context.Context) error
	}
	for _, backend := range []struct {
		name     string
		instance Closeable
	}{
		{"matcher", b.Matcher},
		{"indexer", b.Indexer},
		{"remote indexer", b.RemoteIndexer},
	} {
		if backend.instance != nil {
			err := backend.instance.Close(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "error closing backend", "backend", backend.name, "reason", err)
			}
		}
	}
}
