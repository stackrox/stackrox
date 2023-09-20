package main

import (
	"context"
	"crypto/sha512"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	certsPath := flag.String("certs", "", "Path to directory containing scanner certificates.")
	basicAuth := flag.String("auth", "", "Use basic auth to authenticate with registries.")
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
	var username, password string
	if *basicAuth != "" {
		auth := strings.SplitN(*basicAuth, ":", 2)
		if len(auth) < 2 {
			log.Fatalf("Invalid auth: %q", auth)
		}
		username, password = auth[0], auth[1]
	}
	if len(flag.Args()) < 1 {
		log.Fatalf("Missing <image-url>")
	}

	imageURL := flag.Args()[0]
	ctx := context.Background()
	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		log.Fatalf("tls config: %v", err)
	}

	conn, err := grpc.Dial(":8443", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := v4.NewIndexerClient(conn)

	resp, err := c.CreateIndexReport(ctx, &v4.CreateIndexReportRequest{
		HashId: fmt.Sprintf("/v4/containerimage/%x", sha512.Sum512([]byte(imageURL))),
		ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{ContainerImage: &v4.ContainerImageLocator{
			Url:      imageURL,
			Username: username,
			Password: password,
		}},
	})
	log.Printf("Reply: %v (%v)", resp, err)
	defer utils.IgnoreError(conn.Close)
}
