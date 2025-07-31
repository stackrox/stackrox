package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/vsock-listener/service"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	c := &cobra.Command{
		Use:   "vsock-listener",
		Short: "StackRox VSOCK listener for VM data collection",
	}

	c.AddCommand(
		&cobra.Command{
			Use:   "version",
			Short: "Show version information",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println(buildinfo.ReleaseBuild)
			},
		},
	)

	c.AddCommand(
		&cobra.Command{
			Use:   "run",
			Short: "Run the VSOCK listener service",
			Run: func(cmd *cobra.Command, args []string) {
				runService()
			},
		},
	)

	// Default to run command if no subcommand specified
	if len(os.Args) < 2 {
		runService()
		return
	}

	if err := c.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runService() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal, stopping service...")
		cancel()
	}()

	// Create and start the service
	svc, err := service.NewVSockListener(ctx)
	if err != nil {
		log.Fatalf("Failed to create VSOCK listener service: %v", err)
	}

	if err := svc.Start(); err != nil {
		log.Fatalf("Failed to start VSOCK listener service: %v", err)
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Info("Shutting down VSOCK listener service...")

	if err := svc.Stop(); err != nil {
		log.Errorf("Error stopping service: %v", err)
	}
}
