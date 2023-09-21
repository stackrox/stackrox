package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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

	// Extract the image digest, if not specified.
	if *imageDigest == "" {
		log.Printf("retrieving image digest: %s", imageURL)
		var err error
		auth := authn.FromConfig(authn.AuthConfig{
			Username: username,
			Password: password,
		})
		*imageDigest, err = getImageDigestFromRegistry(imageURL, auth)
		if err != nil {
			log.Fatalf("failed to retrieve image hash id: %v", err)
		}
		log.Printf("image digest: %s", *imageDigest)
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
	defer utils.IgnoreError(conn.Close)

	idxC := v4.NewIndexerClient(conn)
	vulnC := v4.NewMatcherClient(conn)

	hashId := fmt.Sprintf("/v4/containerimage/%s", *imageDigest)
	indexReport, err := idxC.GetIndexReport(ctx, &v4.GetIndexReportRequest{HashId: hashId})
	if err != nil || indexReport.State == "IndexError" {
		indexReport, err = idxC.CreateIndexReport(ctx, &v4.CreateIndexReportRequest{
			HashId: hashId,
			ResourceLocator: &v4.CreateIndexReportRequest_ContainerImage{ContainerImage: &v4.ContainerImageLocator{
				Url:      imageURL,
				Username: username,
				Password: password,
			}},
		})
		if err != nil {
			log.Fatalf("create report failed: %#v (%v)", indexReport, err)
		}
	}
	log.Printf("Index Report: %s", indexReport.GetHashId())
	vulnResp, err := vulnC.GetVulnerabilities(ctx, &v4.GetVulnerabilitiesRequest{
		HashId:   hashId,
		Contents: nil,
	})
	if err != nil {
		log.Fatalf("failed to get vulnerabilities: %s", err)
	}
	vulnJSON, err := json.MarshalIndent(vulnResp, "", "  ")
	if err != nil {
		log.Fatalf("could not marshal vulnerability report: %s", err)
	}
	fmt.Println(string(vulnJSON))
}

func getImageDigestFromRegistry(imageURL string, auth authn.Authenticator) (string, error) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}
	refStr := strings.TrimPrefix(imageURL, u.Scheme+"://")

	// Create a new image reference
	ref, err := name.ParseReference(refStr)
	if err != nil {
		log.Fatalf("parsing reference: %v", err)
	}

	// Retrieve the image with authentication
	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		log.Fatalf("reading image: %v", err)
	}

	// Get the digest of the image
	digest, err := img.Digest()
	if err != nil {
		log.Fatalf("getting digest: %v", err)
	}
	return digest.String(), nil
}
