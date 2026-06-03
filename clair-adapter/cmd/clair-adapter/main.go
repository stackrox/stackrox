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

	// Set up logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))

	slog.Info("starting clair-adapter",
		"clair_url", cfg.ClairURL,
		"grpc_addr", cfg.GRPCListenAddr,
		"http_addr", cfg.HTTPListenAddr,
		"indexer_enabled", cfg.Indexer.Enable,
		"matcher_enabled", cfg.Matcher.Enable,
	)

	// Create Clair client
	clairClient, err := clairclient.NewClient(cfg.ClairURL)
	if err != nil {
		return fmt.Errorf("creating clair client: %w", err)
	}

	// Create CSAF enricher + pipeline
	csafEnricher := csafpkg.NewEnricher()
	enricherPipeline := enricher.NewPipeline(enricher.WithCSAFEnricher(csafEnricher))

	// Create indexer (nil metadataStore for now — DB init is future plan)
	var idx idxpkg.Indexer
	if cfg.Indexer.Enable {
		idx = idxpkg.NewLocalIndexer(clairClient, nil)
	}

	// Create matcher (nil metadataStore)
	var mtch matcherpkg.Matcher
	if cfg.Matcher.Enable {
		mtch = matcherpkg.NewLocalMatcher(clairClient, enricherPipeline, nil)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	if idx != nil {
		indexService := services.NewIndexerService(idx)
		v4.RegisterIndexerServer(grpcServer, indexService)
	}
	if mtch != nil {
		matchService := services.NewMatcherService(mtch)
		v4.RegisterMatcherServer(grpcServer, matchService)
	}
	reflection.Register(grpcServer)

	// Create health HTTP server
	isReady := func() bool {
		// Ping Clair to verify connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := clairClient.GetIndexState(ctx)
		return err == nil
	}
	healthHandler := healthz.NewHandler(isReady)
	httpServer := &http.Server{
		Addr:    cfg.HTTPListenAddr,
		Handler: healthHandler,
	}

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", cfg.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("creating grpc listener: %w", err)
	}
	go func() {
		slog.Info("grpc server listening", "addr", cfg.GRPCListenAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			slog.Error("grpc server failed", "error", err)
		}
	}()

	// Start HTTP health server
	go func() {
		slog.Info("http health server listening", "addr", cfg.HTTPListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
		}
	}()

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down gracefully")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("http server shutdown failed", "error", err)
	}

	slog.Info("shutdown complete")
	return nil
}
