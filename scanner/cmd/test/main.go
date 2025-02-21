package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/indexer"
)

var (
	insecureRemoteTransport = proxiedRemoteTransport(true)
)

func proxiedRemoteTransport(insecure bool) http.RoundTripper {
	tr := func() *http.Transport {
		tr, ok := remote.DefaultTransport.(*http.Transport)
		if !ok {
			// The proxy function was already modified to proxy.TransportFunc.
			// See scanner/cmd/scanner/main.go.
			return http.DefaultTransport.(*http.Transport).Clone()
		}
		tr = tr.Clone()
		tr.Proxy = proxy.TransportFunc
		return tr
	}()
	if insecure {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
			// This ensures we do not use host certs.
			RootCAs: x509.NewCertPool(),
		}
	}
	return tr
}

// parseContainerImageURL returns an image reference from an image URL.
func parseContainerImageURL(imageURL string) (name.Reference, error) {
	// We expect input was sanitized, so all errors here are considered internal errors.
	if imageURL == "" {
		return nil, errors.New("invalid URL: empty")
	}
	// Parse image reference to ensure it is valid.
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return nil, err
	}
	// Check URL scheme and update ref parsing options.
	parseOpts := []name.Option{name.StrictValidation}
	switch parsedURL.Scheme {
	case "http":
		parseOpts = append(parseOpts, name.Insecure)
	case "https":
	default:
		return nil, fmt.Errorf("invalid URL scheme %q", parsedURL.Scheme)
	}
	// Strip the URL scheme:// and parse host/path as an image reference.
	imageRef := strings.TrimPrefix(imageURL, parsedURL.Scheme+"://")
	ref, err := name.ParseReference(imageRef, parseOpts...)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

func getImageManifestID(ref name.Digest) string {
	return fmt.Sprintf("/v4/containerimage/%s", ref.DigestStr())
}

func main() {
	// TODO: Change this URL to Artifactory.
	imageURL := "https://34.23.154.189/docker/nginx@sha256:37c022aa2e42b98eb787cfe6be34e5457081c5b7693a4d8ea8fa43b2f07e1bbc"

	ref, err := parseContainerImageURL(imageURL)
	if err != nil {
		panic(err)
	}
	fmt.Println(ref)

	desc, err := remote.Get(ref,
		remote.WithContext(context.Background()),
		remote.WithAuth(&authn.Basic{
			Username: "admin",
			Password: "Password1",
		}),
		remote.WithTransport(insecureRemoteTransport),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(desc.MediaType)

	index, err := indexer.NewIndexer(context.Background(), config.IndexerConfig{
		Database: config.Database{
			ConnString: "host=127.0.0.1 port=5432 sslmode=disable",
		},
		Enable:              true,
		GetLayerTimeout:     config.Duration(time.Minute),
		RepositoryToCPEURL:  "https://security.access.redhat.com/data/metrics/repository-to-cpe.json",
		NameToReposURL:      "https://security.access.redhat.com/data/metrics/container-name-repos-map.json",
	})
	if err != nil {
		panic(err)
	}

	dig, err := name.NewDigest(imageURL)
	if err != nil {
		panic(err)
	}

	report, err := index.IndexContainerImage(
		context.Background(),
		getImageManifestID(dig),
		"https://34.23.154.189/docker/nginx@sha256:37c022aa2e42b98eb787cfe6be34e5457081c5b7693a4d8ea8fa43b2f07e1bbc",
		indexer.InsecureSkipTLSVerify(true),
		indexer.WithAuth(&authn.Basic{
			Username: "admin",
			Password: "Password1",
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(report)
}
