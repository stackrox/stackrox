package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/internal/version"
	"github.com/stackrox/rox/scanner/pkg/client"
	"golang.org/x/sys/unix"
)

type rootCommand struct {
	cobra.Command
	ScannerClient client.Scanner
}

func rootCmd(ctx context.Context) *cobra.Command {
	cmd := rootCommand{
		Command: cobra.Command{
			Use:          "scannerctl",
			Version:      version.Version,
			Short:        "Controls the StackRox Scanner.",
			SilenceUsage: true,
		},
	}
	cmd.SetContext(ctx)

	flags := cmd.PersistentFlags()

	address := flags.String("address", ":8443", "Address of the scanner service (indexer and matcher).")
	indexerAddr := flags.String("indexer-address", ":8443", "Address of the indexer service.")
	matcherAddr := flags.String("matcher-address", ":8443", "Address of the matcher service.")
	serverName := flags.String("server-name", "scanner-v4.stackrox",
		"Server name of the scanner service, primarily used for TLS verification.")
	skipTLSVerify := flags.Bool("insecure-skip-tls-verify", false, "Skip TLS certificate validation.")
	certsPath := flags.String("certs", "", "Path to directory containing scanner certificates.")

	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if *certsPath != "" {
			// Certs flag configures the identity environment.
			utils.CrashOnError(os.Setenv(mtls.CAFileEnvName, filepath.Join(*certsPath, mtls.CACertFileName)))
			utils.CrashOnError(os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(*certsPath, mtls.CAKeyFileName)))
			utils.CrashOnError(os.Setenv(mtls.CertFilePathEnvName, filepath.Join(*certsPath, mtls.ServiceCertFileName)))
			utils.CrashOnError(os.Setenv(mtls.KeyFileEnvName, filepath.Join(*certsPath, mtls.ServiceKeyFileName)))
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
		// Connect to scanner.
		var err error
		cmd.ScannerClient, err = client.NewGRPCScanner(ctx, opts...)
		if err != nil {
			return fmt.Errorf("connecting: %w", err)
		}
		return nil
	}
	cmd.AddCommand(scanCmd(ctx, &cmd))
	return &cmd.Command
}

func scanCmd(ctx context.Context, parent *rootCommand) *cobra.Command {
	cmd := cobra.Command{
		Use:   "scan http(s)://<image-reference>",
		Short: "Perform vulnerability scans.",
		Args:  cobra.ExactArgs(1),
	}
	flags := cmd.PersistentFlags()
	authEnvName := "ROX_SCANNERCTL_BASIC_AUTH"
	basicAuth := flags.String("auth", "", fmt.Sprintf("Use the specified basic "+
		"auth credentials (warning: debug only and unsafe, use env var %s).",
		authEnvName))
	imageDigest := flags.String("digest", "", "Use the specified image digest in "+
		"the image manifest ID. The default is to retrieve the image digest from "+
		"the registry and use that.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Extract basic auth username and password.
		auth := authn.Anonymous
		if *basicAuth == "" {
			*basicAuth = os.Getenv(authEnvName)
		}
		if *basicAuth != "" {
			u, p, ok := strings.Cut(*basicAuth, ":")
			if !ok {
				return errors.New("invalid basic auth: expecting the username and the " +
					"password with a colon (aladdin:opensesame)")
			}
			auth = &authn.Basic{
				Username: u,
				Password: p,
			}
		}
		// Get the image digest, from the URL or command option.
		imageURL := args[0]
		ref, err := indexer.GetDigestFromURL(imageURL, auth)
		if err != nil {
			return fmt.Errorf("failed to retrieve image hash id: %w", err)
		}
		if *imageDigest == "" {
			*imageDigest = ref.DigestStr()
			log.Printf("image digest: %s", *imageDigest)
		}
		if *imageDigest != ref.DigestStr() {
			log.Printf("WARNING: the actual image digest %q is different from %q",
				ref.DigestStr(), *imageDigest)
		}

		vr, err := parent.ScannerClient.IndexAndScanImage(ctx, ref, auth)
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		vrJSON, err := json.MarshalIndent(vr, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}

		fmt.Println(string(vrJSON))

		return nil
	}

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

	if err := rootCmd(ctx).Execute(); err != nil {
		log.Fatal(err)
	}
}
