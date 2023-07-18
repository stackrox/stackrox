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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	certsFlag := flag.String("certs", "", "Path to directory containing scanner certificates.")
	flag.Parse()

	// If certs was specified, configure the identity environment.
	// TODO: Add a flag to disable mTLS altogether
	if *certsFlag != "" {
		os.Setenv(mtls.CAFileEnvName, filepath.Join(*certsFlag, mtls.CACertFileName))
		os.Setenv(mtls.CAKeyFileEnvName, filepath.Join(*certsFlag, mtls.CAKeyFileName))
		os.Setenv(mtls.CertFilePathEnvName, filepath.Join(*certsFlag, mtls.ServiceCertFileName))
		os.Setenv(mtls.KeyFileEnvName, filepath.Join(*certsFlag, mtls.ServiceKeyFileName))
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
