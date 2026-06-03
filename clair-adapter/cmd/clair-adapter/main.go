package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
		"grpc_addr", cfg.GRPCListenAddr,
		"http_addr", cfg.HTTPListenAddr,
		"updater_addr", cfg.UpdaterListenAddr,
		"vuln_url", cfg.VulnerabilitiesURL,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	clairClient, err := clairclient.NewClient(cfg.ClairURL)
	if err != nil {
		return fmt.Errorf("creating clair client: %w", err)
	}

	// Updater: fetch vulnerability data and serve to Clair
	updaterServer := updater.NewServer()
	fetcher := updater.NewFetcher(updaterServer, []string{cfg.VulnerabilitiesURL})
	go func() {
		slog.Info("starting vulnerability data fetcher")
		if err := fetcher.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("vulnerability fetcher failed", "error", err)
		}
	}()

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

	// Enrichment pipeline
	csafEnricher := csafpkg.NewEnricher()
	enricherPipeline := enricher.NewPipeline(enricher.WithCSAFEnricher(csafEnricher))

	// Indexer
	var idx idxpkg.Indexer
	if cfg.Indexer.Enable {
		idx = idxpkg.NewLocalIndexer(clairClient, nil)
	}

	// Matcher
	var mtch matcherpkg.Matcher
	if cfg.Matcher.Enable {
		mtch = matcherpkg.NewLocalMatcher(clairClient, enricherPipeline, nil)
	}

	// gRPC server
	grpcServer := grpc.NewServer()
	if idx != nil {
		v4.RegisterIndexerServer(grpcServer, services.NewIndexerService(idx))
	}
	if mtch != nil {
		v4.RegisterMatcherServer(grpcServer, services.NewMatcherService(mtch))
	}
	reflection.Register(grpcServer)

	grpcLis, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("creating grpc listener: %w", err)
	}
	go func() {
		slog.Info("gRPC server listening", "addr", cfg.GRPCListenAddr)
		if err := grpcServer.Serve(grpcLis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	// Health server
	healthHandler := healthz.NewHandler(func() bool {
		hctx, hcancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer hcancel()
		_, err := clairClient.GetIndexState(hctx)
		return err == nil
	})
	httpServer := &http.Server{
		Addr:    cfg.HTTPListenAddr,
		Handler: healthHandler,
	}
	go func() {
		slog.Info("health HTTP server listening", "addr", cfg.HTTPListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health HTTP server failed", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	httpServer.Shutdown(shutdownCtx)
	updaterHTTPServer.Shutdown(shutdownCtx)

	slog.Info("shutdown complete")
	return nil
}
