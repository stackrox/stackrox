package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	certsPath := flag.String("certs", "", "Path to directory containing scanner certificates.")
	flag.Parse()

	// If certs was specified, configure the identity environment.
	// TODO: Add a flag to disable mTLS altogether
	if *certsPath != "" {
		utils.CrashOnError(os.Setenv(mtls.CAFileEnvName, filepath.Join(*certsPath, mtls.CACertFileName)))
		utils.CrashOnError(os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(*certsPath, mtls.CAKeyFileName)))
		utils.CrashOnError(os.Setenv(mtls.CertFilePathEnvName, filepath.Join(*certsPath, mtls.ServiceCertFileName)))
		utils.CrashOnError(os.Setenv(mtls.KeyFileEnvName, filepath.Join(*certsPath, mtls.ServiceKeyFileName)))
	}

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
		HashId:          "",
		ResourceLocator: nil,
	})
	log.Printf("Reply: %v (%v)", resp, err)
	defer conn.Close()
}
