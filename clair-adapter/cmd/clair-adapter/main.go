package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/stackrox/rox/clair-adapter/clairclient"
	"github.com/stackrox/rox/clair-adapter/config"
	"github.com/stackrox/rox/clair-adapter/enricher"
	csafpkg "github.com/stackrox/rox/clair-adapter/enricher/csaf"
	"github.com/stackrox/rox/clair-adapter/healthz"
	idxpkg "github.com/stackrox/rox/clair-adapter/indexer"
	matcherpkg "github.com/stackrox/rox/clair-adapter/matcher"
	"github.com/stackrox/rox/clair-adapter/services"
	"github.com/stackrox/rox/clair-adapter/updater"
	"github.com/stackrox/rox/clair-adapter/vulnimporter"
	pkggrpc "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnservice "github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()
	if err := run(*configPath); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))

	slog.Info("starting clair-adapter",
		"clair_url", cfg.ClairURL,
		"clair_db", cfg.ClairDBConnString != "",
		"grpc_addr", cfg.GRPCListenAddr,
		"http_addr", cfg.HTTPListenAddr,
		"vuln_url", cfg.VulnerabilitiesURL,
		"certs_dir", cfg.CertsDir,
	)

	// Configure mTLS certificate paths if CertsDir is set.
	if cfg.CertsDir != "" {
		os.Setenv(mtls.CAFileEnvName, filepath.Join(cfg.CertsDir, mtls.CACertFileName))
		os.Setenv(mtls.CertFilePathEnvName, filepath.Join(cfg.CertsDir, mtls.ServiceCertFileName))
		os.Setenv(mtls.KeyFileEnvName, filepath.Join(cfg.CertsDir, mtls.ServiceKeyFileName))
		slog.Info("mTLS configured", "certs_dir", cfg.CertsDir)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	clairClient, err := clairclient.NewClient(cfg.ClairURL)
	if err != nil {
		return fmt.Errorf("creating clair client: %w", err)
	}

	// Set up vulnerability bundle importer and enrichment fetcher if Clair DB is configured.
	var imp *vulnimporter.Importer
	var enrichFetcher *vulnimporter.EnrichmentFetcher
	if cfg.ClairDBConnString != "" {
		slog.Info("connecting to Clair database for direct vulnerability import")
		store, pool, err := vulnimporter.NewMatcherStoreAndPool(ctx, cfg.ClairDBConnString)
		if err != nil {
			return fmt.Errorf("creating Clair matcher store: %w", err)
		}
		imp = vulnimporter.NewImporter(store, nil)
		enrichFetcher = vulnimporter.NewEnrichmentFetcher(pool)
	}

	// Configure HTTP client for fetching vulnerability bundles.
	// Use mTLS if certs are available (for Central's definitions endpoint).
	var fetcherOpts []updater.FetcherOption
	if cfg.CertsDir != "" {
		mtlsClient, err := updater.NewMTLSHTTPClient()
		if err != nil {
			slog.WarnContext(ctx, "failed to create mTLS HTTP client, using default", "error", err)
		} else {
			fetcherOpts = append(fetcherOpts, updater.WithHTTPClient(mtlsClient))
		}
	}
	if imp != nil {
		fetcherOpts = append(fetcherOpts, updater.WithImporter(imp))
	}

	// Updater: fetch vulnerability data, serve to Clair, and import into Clair's DB.
	updaterServer := updater.NewServer()
	fetcher := updater.NewFetcher(updaterServer, []string{cfg.VulnerabilitiesURL}, fetcherOpts...)
	go func() {
		slog.Info("starting vulnerability data fetcher")
		if err := fetcher.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("vulnerability fetcher failed", "error", err)
		}
	}()

	// Keep the updater HTTP server as a diagnostic endpoint.
	updaterHTTPServer := &http.Server{
		Addr:    cfg.UpdaterListenAddr,
		Handler: updaterServer,
	}
	go func() {
		slog.Info("updater HTTP server listening", "addr", cfg.UpdaterListenAddr)
		if err := updaterHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("updater HTTP server failed", "error", err)
		}
	}()

	// Enrichment pipeline.
	csafEnricher := csafpkg.NewEnricher()
	enricherPipeline := enricher.NewPipeline(enricher.WithCSAFEnricher(csafEnricher))

	// Indexer.
	var idx idxpkg.Indexer
	if cfg.Indexer.Enable {
		idx = idxpkg.NewLocalIndexer(clairClient, nil)
	}

	// Matcher.
	var mtch matcherpkg.Matcher
	if cfg.Matcher.Enable {
		var matcherOpts []matcherpkg.LocalMatcherOption
		if enrichFetcher != nil {
			matcherOpts = append(matcherOpts, matcherpkg.WithEnrichmentFetcher(enrichFetcher))
		}
		mtch = matcherpkg.NewLocalMatcher(clairClient, enricherPipeline, nil, matcherOpts...)
	}

	// Health handler.
	readinessFunc := func() bool {
		hctx, hcancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer hcancel()
		_, err := clairClient.GetIndexState(hctx)
		return err == nil
	}
	healthHandler := healthz.NewHandler(readinessFunc)

	// Identity extractor for mTLS client certificates.
	identityExtractor, err := authnservice.NewExtractor()
	if err != nil {
		return fmt.Errorf("creating identity extractor: %w", err)
	}

	// gRPC API with mTLS.
	grpcAPI := pkggrpc.NewAPI(pkggrpc.Config{
		CustomRoutes:       healthHandler.CustomRoutes(),
		IdentityExtractors: []authn.IdentityExtractor{identityExtractor},
		Endpoints: []*pkggrpc.EndpointConfig{
			{ListenEndpoint: cfg.GRPCListenAddr, TLS: verifier.NonCA{}, ServeGRPC: true, ServeHTTP: false},
			{ListenEndpoint: cfg.HTTPListenAddr, TLS: verifier.NonCA{}, ServeGRPC: false, ServeHTTP: true},
		},
	})

	var apiServices []pkggrpc.APIService
	if idx != nil {
		apiServices = append(apiServices, services.NewIndexerService(idx))
	}
	if mtch != nil {
		apiServices = append(apiServices, services.NewMatcherService(mtch))
	}
	grpcAPI.Register(apiServices...)

	startSig := grpcAPI.Start()
	select {
	case <-startSig.Done():
		if err := startSig.Err(); err != nil {
			return fmt.Errorf("failed to start gRPC API: %w", err)
		}
		slog.Info("gRPC API started", "grpc_addr", cfg.GRPCListenAddr, "http_addr", cfg.HTTPListenAddr)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for gRPC API to start")
	}

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcAPI.Stop()
	updaterHTTPServer.Shutdown(shutdownCtx)

	slog.Info("shutdown complete")
	return nil
}
