package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/quay/zlog"
	grpcmetrics "github.com/stackrox/rox/central/grpc/metrics"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/stackrox/scanner/v4/indexer"
	"github.com/stackrox/stackrox/scanner/v4/matcher"
	"github.com/stackrox/stackrox/scanner/v4/version"
	"golang.org/x/sys/unix"
)

type Backends struct {
	Indexer *indexer.Indexer
	Matcher *matcher.Matcher
}

func main() {
	// TODO: Use a configuration file.
	certsPath := flag.String("certs", "", "Path to directory containing scanner certificates.")
	flag.Parse()

	// If certs was specified, configure the identity environment.
	if *certsPath != "" {
		os.Setenv(mtls.CAFileEnvName, filepath.Join(*certsPath, mtls.CACertFileName))
		os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(*certsPath, mtls.CAKeyFileName))
		os.Setenv(mtls.CertFilePathEnvName, filepath.Join(*certsPath, mtls.ServiceCertFileName))
		os.Setenv(mtls.KeyFileEnvName, filepath.Join(*certsPath, mtls.ServiceKeyFileName))
	}

	// Create cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize logging and setup context.
	err := initializeLogging()
	if err != nil {
		log.Fatalf("failed to initialize logging: %v", err)
	}
	ctx = zlog.ContextWithValues(ctx, "component", "main")
	zlog.Info(ctx).Str("version", version.Version).Msg("starting scanner")

	// Initialize metrics and metrics server.
	metricsSrv := metrics.NewServer(metrics.ScannerSubsystem, metrics.NewTLSConfigurerFromEnv())
	metricsSrv.RunForever()
	defer metricsSrv.Stop(ctx)
	metrics.GatherThrottleMetricsForever(metrics.ScannerSubsystem.String())

	// Create backends.
	backends, err := createBackends(ctx)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("failed to create backends")
		os.Exit(1)
	}
	defer backends.Close(ctx)
	zlog.Info(ctx).Msg("backends are ready")

	// Initialize gRPC API service.
	grpcSrv, err := createGRPCService(backends)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("failed to initialize gRPC")
		os.Exit(1)
	}
	grpcSrv.Start()
	defer grpcSrv.Stop()

	// Wait for signals.
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, unix.SIGTERM)
	sig := <-sigC
	zlog.Info(ctx).Str("signal", sig.String()).Send()
}

// createGRPCService creates a ready-to-start gRPC API instance and register its services.
func createGRPCService(backends *Backends) (grpc.API, error) {
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
		GRPCMetrics:        grpcmetrics.GRPCSingleton(),
		HTTPMetrics:        grpcmetrics.HTTPSingleton(),
		Endpoints: []*grpc.EndpointConfig{
			{
				ListenEndpoint: ":8443",
				TLS:            verifier.NonCA{},
				ServeGRPC:      true,
				ServeHTTP:      false,
			},
			{
				ListenEndpoint: ":9095",
				TLS:            verifier.NonCA{},
				ServeGRPC:      false,
				ServeHTTP:      true,
			},
		},
	})

	// Create and register API services.
	var srvs []grpc.APIService
	if backends.Indexer != nil {
		s, err := indexer.NewIndexerService(backends.Indexer)
		if err != nil {
			return nil, fmt.Errorf("indexer service: %w", err)
		}
		srvs = append(srvs, s)
	}
	if backends.Matcher != nil {
		s, err := matcher.NewMatcherService(backends.Matcher)
		if err != nil {
			return nil, fmt.Errorf("matcher service: %w", err)
		}
		srvs = append(srvs, s)
	}
	grpcSrv.Register(srvs...)

	return grpcSrv, nil
}

// createAPIServices creates all backends.
func createBackends(ctx context.Context) (*Backends, error) {
	// TODO: Use modes to decide which backends to start.
	// Indexer.
	i, err := indexer.NewIndexer(ctx)
	if err != nil {
		return nil, fmt.Errorf("indexer: %w", err)
	}
	// Matcher.
	m, err := matcher.NewMatcher(ctx)
	if err != nil {
		return nil, fmt.Errorf("matcher: %w", err)
	}
	return &Backends{
		Indexer: i,
		Matcher: m,
	}, nil
}

func (b *Backends) Close(ctx context.Context) {
	if b.Indexer != nil {
		err := b.Indexer.Close(ctx)
		if err != nil {
			zlog.Error(ctx).Err(err).Msg("closing indexer")
		}
	}
	if b.Matcher != nil {
		err := b.Matcher.Close(ctx)
		if err != nil {
			zlog.Error(ctx).Err(err).Msg("closing matcher")
		}
	}
}
