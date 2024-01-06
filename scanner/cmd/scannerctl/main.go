package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/internal/version"
	"golang.org/x/sys/unix"
)

// factory is the global scanner client factory.
var factory scannerFactory

// scannerFactory holds the data to create scanner clients.
type scannerFactory []client.Option

// Create creates a new scanner client.
func (o scannerFactory) Create(ctx context.Context) (client.Scanner, error) {
	c, err := client.NewGRPCScanner(ctx, o...)
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	return c, nil
}

// rootCmd creates the base command when called without any subcommands.
func rootCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:          "scannerctl",
		Version:      version.Version,
		Short:        "Controls the StackRox Scanner.",
		SilenceUsage: true,
	}
	cmd.SetContext(ctx)
	flags := cmd.PersistentFlags()
	address := flags.String(
		"address",
		":8443",
		"Address of the scanner service (indexer and matcher).")
	indexerAddr := flags.String(
		"indexer-address",
		":8443",
		"Address of the indexer service.")
	matcherAddr := flags.String(
		"matcher-address",
		":8443",
		"Address of the matcher service.")
	serverName := flags.String(
		"server-name",
		"scanner-v4.stackrox",
		"Server name of the scanner service, primarily used for TLS verification.")
	skipTLSVerify := flags.Bool(
		"insecure-skip-tls-verify",
		false,
		"Skip TLS certificate validation.")
	certsPath := flags.String(
		"certs",
		"",
		"Path to directory containing scanner certificates.")
	cmd.PersistentPreRun = func(_ *cobra.Command, _ []string) {
		if *certsPath != "" {
			// Certs flag configures the identity environment.
			utils.CrashOnError(
				os.Setenv(mtls.CAFileEnvName,
					filepath.Join(*certsPath, mtls.CACertFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.CAKeyFileEnvName,
					filepath.Join(*certsPath, mtls.CAKeyFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.CertFilePathEnvName,
					filepath.Join(*certsPath, mtls.ServiceCertFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.KeyFileEnvName,
					filepath.Join(*certsPath, mtls.ServiceKeyFileName)))
		}
		// Set options for the gRPC connection.
		opts := []client.Option{
			client.WithServerName(*serverName),
			client.WithAddress(*address),
		}
		if *skipTLSVerify {
			opts = append(opts, client.SkipTLSVerification)
		}
		if *indexerAddr != "" {
			opts = append(opts, client.WithIndexerAddress(*indexerAddr))
		}
		if *matcherAddr != "" {
			opts = append(opts, client.WithMatcherAddress(*matcherAddr))
		}
		// Create the client factory.
		factory = opts
	}
	cmd.AddCommand(scanCmd(ctx))
	cmd.AddCommand(scaleCmd(ctx))
	return &cmd
}

func main() {
	// Create a context that is cancellable on the usual command line signals. Double
	// signal forcefully exits.
	ctx, cancel := context.WithCancel(context.Background())
	defer ctx.Done()
	go func() {
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, unix.SIGINT, unix.SIGTERM)
		sig := <-sigC
		log.Printf("%s caught, shutting down...", sig)
		// Cancel the main context.
		cancel()
		go func() {
			// A second signal will forcefully quit.
			<-sigC
			os.Exit(1)
		}()
	}()
	// Execute command.
	if err := rootCmd(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}
