package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/pkg/client"
	"golang.org/x/sys/unix"
)

var authEnvName = "ROX_SCANNERCTL_BASIC_AUTH"

func main() {
	address := flag.String("address", ":8443", "Address of the scanner service.")
	serverName := flag.String("server-name", "scanner-v4.stackrox",
		"Server name of the scanner service, primarily used for TLS verification.")
	skipTLSVerify := flag.Bool("insecure-skip-tls-verify", false, "Skip TLS certificate validation.")
	certsPath := flag.String("certs", "", "Path to directory containing scanner certificates.")
	basicAuth := flag.String("auth", "", fmt.Sprintf("Use the specified basic auth credentials "+
		"(warning: debug only and unsafe, use env var %s).", authEnvName))
	imageDigest := flag.String("digest", "", "Use the specified image digest in the image "+
		"manifest ID. The default is to retrieve the image digest from the registry and "+
		"use that.")
	flag.Parse()

	// If certs was specified, configure the identity environment.
	// TODO: Add a flag to disable mTLS altogether
	if *certsPath != "" {
		utils.CrashOnError(os.Setenv(mtls.CAFileEnvName, filepath.Join(*certsPath, mtls.CACertFileName)))
		utils.CrashOnError(os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(*certsPath, mtls.CAKeyFileName)))
		utils.CrashOnError(os.Setenv(mtls.CertFilePathEnvName, filepath.Join(*certsPath, mtls.ServiceCertFileName)))
		utils.CrashOnError(os.Setenv(mtls.KeyFileEnvName, filepath.Join(*certsPath, mtls.ServiceKeyFileName)))
	}

	// Extract basic auth username and password.
	auth := authn.Anonymous
	if *basicAuth == "" {
		*basicAuth = os.Getenv(authEnvName)
	}
	if *basicAuth != "" {
		u, p, ok := strings.Cut(*basicAuth, ":")
		if !ok {
			log.Fatal("Invalid basic auth: expecting the username and the " +
				"password with a colon (aladdin:opensesame)")
		}
		auth = &authn.Basic{
			Username: u,
			Password: p,
		}
	}

	if len(flag.Args()) < 1 {
		log.Fatalf("Missing <image-url>")
	}

	// Get the image digest, from the URL or command option.
	imageURL := flag.Args()[0]
	ref, err := indexer.GetDigestFromURL(imageURL, auth)
	if err != nil {
		log.Fatalf("failed to retrieve image hash id: %v", err)
	}
	if *imageDigest == "" {
		*imageDigest = ref.DigestStr()
		log.Printf("image digest: %s", *imageDigest)
	}
	if *imageDigest != ref.DigestStr() {
		log.Printf("WARNING: the actual image digest %q is different from %q",
			ref.DigestStr(), *imageDigest)
	}

	// Set options for the gRPC connection.
	opts := []client.Option{
		client.WithServerName(*serverName),
		client.WithAddress(*address),
	}
	if *skipTLSVerify {
		opts = append(opts, client.WithoutTLSVerify)
	}

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

	// Connect to scanner and scan.
	c, err := client.NewGRPCScanner(ctx, opts...)
	if err != nil {
		log.Fatalf("connecting: %v", err)
	}
	vr, err := c.IndexAndScanImage(ctx, ref, auth)
	if err != nil {
		log.Fatalf("scanning: %v", err)
	}
	vrJSON, err := json.MarshalIndent(vr, "", "  ")
	if err != nil {
		log.Fatalf("decoding report: %v", err)
	}

	fmt.Println(string(vrJSON))
}
