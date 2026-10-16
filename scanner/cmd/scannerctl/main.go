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

// certsCleanup removes temporary certificate files created by --certs-secret.
var certsCleanup func()

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

	cmd.PersistentFlags().String("indexer-address", "", "Address of the indexer service")
	cmd.PersistentFlags().String("matcher-address", "", "Address of the matcher service")

	indexerServerName := flags.String(
		"indexer-server-name",
		"",
		"Server name of the indexer service, primarily used for TLS verification.")
	matcherServerName := flags.String(
		"matcher-server-name",
		"",
		"Server name of the matcher service, primarily used for TLS verification.")
	skipTLSVerify := flags.Bool(
		"insecure-skip-tls-verify",
		false,
		"Skip TLS certificate validation.")
	certsPath := flags.String(
		"certs",
		"",
		"Path to directory containing scanner certificates.")
	certsSecret := flags.String(
		"certs-secret",
		"",
		fmt.Sprintf("Load certificates from a Kubernetes secret ([namespace/]name). "+
			"Namespace defaults to %q. Uses the current kubeconfig context. "+
			"Default: %s/%s.", defaultCertsNamespace, defaultCertsNamespace, defaultCertsSecret))
	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		certsDir := *certsPath

		if *certsPath != "" && cmd.Flags().Changed("certs-secret") {
			return fmt.Errorf("--certs and --certs-secret are mutually exclusive")
		}

		if certsDir == "" {
			ref := defaultCertsSecret
			if cmd.Flags().Changed("certs-secret") {
				ref = *certsSecret
			}
			dir, cleanup, err := loadCertsFromSecret(cmd.Context(), ref)
			if err != nil {
				if cmd.Flags().Changed("certs-secret") {
					return fmt.Errorf("loading certs from secret: %w", err)
				}
				log.Printf("could not load default secret %s (use --certs or --certs-secret to configure): %v", ref, err)
			} else {
				certsDir = dir
				certsCleanup = cleanup
			}
		}

		if certsDir != "" {
			utils.CrashOnError(
				os.Setenv(mtls.CAFileEnvName,
					filepath.Join(certsDir, mtls.CACertFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.CAKeyFileEnvName,
					filepath.Join(certsDir, mtls.CAKeyFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.CertFilePathEnvName,
					filepath.Join(certsDir, mtls.ServiceCertFileName)))
			utils.CrashOnError(
				os.Setenv(mtls.KeyFileEnvName,
					filepath.Join(certsDir, mtls.ServiceKeyFileName)))
		}

		indexerAddr, err := cmd.Flags().GetString("indexer-address")
		if err != nil {
			return fmt.Errorf("getting indexer-address: %w", err)
		}

		matcherAddr, err := cmd.Flags().GetString("matcher-address")
		if err != nil {
			return fmt.Errorf("getting matcher-address: %w", err)
		}

		// Set options for the gRPC connection.
		opts := []client.Option{
			client.WithIndexerAddress(indexerAddr),
			client.WithIndexerServerName(*indexerServerName),
			client.WithIndexerSubject(mtls.ScannerV4IndexerSubject),
			client.WithMatcherAddress(matcherAddr),
			client.WithMatcherServerName(*matcherServerName),
			client.WithMatcherSubject(mtls.ScannerV4MatcherSubject),
		}
		if *skipTLSVerify {
			opts = append(opts, client.SkipTLSVerification)
		}
		if indexerAddr == matcherAddr && *indexerServerName == *matcherServerName {
			opts = append(opts, client.WithSubject(mtls.ScannerV4Subject))
		}
		// Create the client factory.
		factory = opts
		return nil
	}
	cmd.AddCommand(scanCmd(ctx))
	cmd.AddCommand(scaleCmd(ctx))
	cmd.AddCommand(sbomCmd(ctx))
	cmd.AddCommand(scanVM(ctx))
	return &cmd
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if certsCleanup != nil {
			certsCleanup()
		}
	}()
	go func() {
		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, unix.SIGINT, unix.SIGTERM)
		sig := <-sigC
		log.Printf("%s caught, shutting down...", sig)
		cancel()
		go func() {
			<-sigC
			if certsCleanup != nil {
				certsCleanup()
			}
			os.Exit(1)
		}()
	}()
	if err := rootCmd(ctx).Execute(); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
