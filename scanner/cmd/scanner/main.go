package main

import (
	"context"
	"flag"
	"fmt"
	golog "log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	grpcmetrics "github.com/stackrox/rox/pkg/grpc/metrics"
	"github.com/stackrox/rox/pkg/grpc/routes"
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

func main() {
	configPath := flag.String("conf", "", "Path to scanner's configuration file.")
	flag.Parse()
	cfg, err := config.Read(*configPath)
	if err != nil {
		golog.Fatalf("failed to load configuration %q: %v", *configPath, err)
	}

	// Create cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logging and setup context.
	err = initializeLogging(zerolog.Level(cfg.LogLevel))
	if err != nil {
		golog.Fatalf("failed to initialize logging: %v", err)
	}
	ctx = zlog.ContextWithValues(ctx, "component", "main")
	zlog.Info(ctx).Str("version", version.Version).Msg("starting scanner")

	// If certs was specified, configure the identity environment.
	if p := cfg.MTLS.CertsDir; p != "" {
		zlog.Info(ctx).Str("certs_prefix", p).Msg("identity certificates filename prefix changed")
		utils.CrashOnError(os.Setenv(mtls.CAFileEnvName, filepath.Join(p, mtls.CACertFileName)))
		utils.CrashOnError(os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(p, mtls.CAKeyFileName)))
		utils.CrashOnError(os.Setenv(mtls.CertFilePathEnvName, filepath.Join(p, mtls.ServiceCertFileName)))
		utils.CrashOnError(os.Setenv(mtls.KeyFileEnvName, filepath.Join(p, mtls.ServiceKeyFileName)))
	}

	// Initialize metrics and metrics server.
	metricsSrv := metrics.NewServer(metrics.ScannerSubsystem, metrics.NewTLSConfigurerFromEnv())
	metricsSrv.RunForever()
	defer metricsSrv.Stop(ctx)
	metrics.GatherThrottleMetricsForever(metrics.ScannerSubsystem.String())

	// Create backends.
	backends, err := createBackends(ctx, cfg)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("failed to create backends")
		os.Exit(1)
	}
	defer backends.Close(ctx)
	zlog.Info(ctx).Msg("backends are ready")

	// Initialize gRPC API service.
	grpcSrv, err := createGRPCService(backends, cfg)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("failed to initialize gRPC")
		os.Exit(1)
	}
	grpcSrv.Start()
	defer grpcSrv.Stop()

	// Wait for signals.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, unix.SIGINT, unix.SIGTERM)
	sig := <-sigC
	zlog.Info(ctx).Str("signal", sig.String()).Send()
}

// initializeLogging Initialize zerolog and Quay's zlog.
func initializeLogging(logLevel zerolog.Level) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Str("host", hostname).
		Logger()
	zlog.Set(&logger)
	// Disable the default zerolog logger.
	log.Logger = zerolog.Nop()
	return nil
}

// createGRPCService creates a ready-to-start gRPC API instance and register its services.
func createGRPCService(backends *Backends, cfg *config.Config) (grpc.API, error) {
	// Create identity extractors.
	identityExtractor, err := service.NewExtractor()
	if err != nil {
		return nil, fmt.Errorf("identity extractor: %w", err)
	}

	// Create gRPC API service and debug routes.
	customRoutes := make([]routes.CustomRoute, 0, len(routes.DebugRoutes))
	for path, handler := range routes.DebugRoutes {
		customRoutes = append(customRoutes, routes.CustomRoute{
			Route:         path,
			Authorizer:    allow.Anonymous(),
			ServerHandler: handler,
			Compression:   true,
		})
	}
	grpcSrv := grpc.NewAPI(grpc.Config{
		CustomRoutes:       customRoutes,
		IdentityExtractors: []authn.IdentityExtractor{identityExtractor},
		GRPCMetrics:        grpcmetrics.NewGRPCMetrics(),
		HTTPMetrics:        grpcmetrics.NewHTTPMetrics(),
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
	})

	// Register API services.
	grpcSrv.Register(getServices(backends)...)

	return grpcSrv, nil
}

// createBackends creates the scanner backends.
func createBackends(ctx context.Context, cfg *config.Config) (*Backends, error) {
	var b Backends
	var err error
	if cfg.Indexer.Enable {
		zlog.Info(ctx).Msg("indexer is enabled")
		b.Indexer, err = indexer.NewIndexer(ctx, cfg.Indexer)
		if err != nil {
			return nil, fmt.Errorf("indexer: %w", err)
		}
	} else {
		zlog.Info(ctx).Msg("indexer is disabled")
	}
	if cfg.Matcher.Enable {
		zlog.Info(ctx).Msg("matcher is enabled")
		if cfg.Matcher.RemoteIndexer {
			// Create a remote indexer only if the matcher was configured to use one.
			zlog.Info(ctx).Msg("remote indexer is enabled")
			b.RemoteIndexer, err = indexer.NewRemoteIndexer(ctx, cfg.Matcher.IndexerAddr)
			if err != nil {
				return nil, fmt.Errorf("matcher: remote indexer: %w", err)
			}
		}
		b.Matcher, err = matcher.NewMatcher(ctx, cfg.Matcher)
		if err != nil {
			return nil, fmt.Errorf("matcher: %w", err)
		}
	} else {
		zlog.Info(ctx).Msg("matcher is disabled")
	}
	return &b, nil
}

// getServices returns the list of the API services based on the backends provided.
func getServices(b *Backends) []grpc.APIService {
	var srvs []grpc.APIService
	if b.Indexer != nil {
		srvs = append(srvs, services.NewIndexerService(b.Indexer))
	}
	if b.Matcher != nil {
		// Set the index report getter to the remote indexer if available, otherwise the
		// local indexer. A nil getter is ok, see implementation.
		var getter indexer.ReportGetter
		getter = b.Indexer
		if b.RemoteIndexer != nil {
			getter = b.RemoteIndexer
		}
		srvs = append(srvs, services.NewMatcherService(b.Matcher, getter))
	}
	return srvs
}

// Close closes all the backends used by the scanner.
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
				zlog.Error(ctx).Err(err).Msgf("closing %s", backend.name)
			}
		}
	}
}
