package vm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	layerMediaType = "application/vnd.claircore.filesystem"
	rhcosPackageDB = "sqlite:usr/share/rpm"
)

var (
	// layerDigest is a dummy digest solely meant as a workaround to use Claircore.
	layerDigest   = fmt.Sprintf("sha256:%s", strings.Repeat("a", 64))
	ccLayerDigest = claircore.MustParseDigest(layerDigest)

	clientOnce       sync.Once
	defaultClient    *http.Client
	defaultClientErr error
)

func init() {
	// Set up zerolog for claircore
	l := zerolog.New(os.Stderr).Level(zerolog.InfoLevel)
	zlog.Set(&l)
}

// logDebugf logs debug messages (minimal logger to avoid pkg/logging dependency)
func logDebugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

// ignoreError ignores an error from a function (replaces pkgutils.IgnoreError)
func ignoreError(f func() error) {
	if f != nil {
		_ = f()
	}
}

// getProxyFunc returns a proxy function from environment variables
// This is a minimal version to avoid pulling in pkg/httputil/proxy (which pulls k8s)
func getProxyFunc() func(*http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}

// loadClientCertificate loads client certificate from standard locations
// This is a minimal version to avoid pkg/mtls dependency (which pulls k8s)
func loadClientCertificate() (tls.Certificate, error) {
	// Standard paths for mTLS certificates in StackRox
	certFile := "/run/secrets/stackrox.io/certs/cert.pem"
	keyFile := "/run/secrets/stackrox.io/certs/key.pem"

	// Try loading cert/key pair
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "loading client certificate")
	}
	return cert, nil
}

func getDefaultClient() (*http.Client, error) {
	clientOnce.Do(func() {
		clientCert, err := loadClientCertificate()
		if err != nil {
			defaultClientErr = errors.Wrap(err, "obtaining client certificate")
			return
		}
		defaultClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{clientCert},
				},
				Proxy: getProxyFunc(),
			},
			Timeout: 30 * time.Second,
		}
	})
	return defaultClient, defaultClientErr
}

// VMIndexerConfig represents VM indexer configuration parameters.
// This is a simplified version for VM scanning without registry dependencies.
type VMIndexerConfig struct {
	// HostPath is the mount point of the read-only host filesystem.
	HostPath string
	// Client is the HTTP client used to fetch repo-to-CPE mapping.
	Client *http.Client
	// Repo2CPEMappingURL can be used to fetch the repo mapping file.
	Repo2CPEMappingURL string
	// Timeout controls the timeout for remote API calls.
	Timeout time.Duration
	// PackageDBFilter filters packages by packageDB.
	// Empty string means no filtering.
	PackageDBFilter string
}

// IndexVM indexes a VM at the configured host path and returns an index report.
// This function is VM-specific and avoids heavy dependencies like pkg/registries.
func IndexVM(ctx context.Context, cfg VMIndexerConfig) (*v4.IndexReport, error) {
	if _, err := os.Stat(cfg.HostPath); err != nil {
		return nil, errors.Wrapf(err, "host path %q does not exist", cfg.HostPath)
	}

	layer, err := createLayer(ctx, layerDigest, cfg.HostPath)
	if err != nil {
		return nil, err
	}
	defer ignoreError(layer.Close)

	repos, err := scanRepositories(ctx, cfg, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan repositories")
	}

	pkgs, err := scanPackages(ctx, cfg.PackageDBFilter, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan packages")
	}

	ccReport, err := coalesceReport(ctx, ccLayerDigest, repos, pkgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	logDebugf("Finished coalescing report: %d repositories, %d packages",
		len(ccReport.Repositories), len(ccReport.Packages))

	ccReport.Success = true
	ccReport.State = controller.IndexFinished.String()

	// Use our minimal conversion instead of pkg/scannerv4/mappers
	report := toProtoV4IndexReport(ccReport)
	return report, nil
}

func createLayer(ctx context.Context, digest string, hostPath string) (*claircore.Layer, error) {
	if hostPath == "" {
		return nil, errors.New("host path is empty")
	}

	absoluteHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return nil, errors.Wrapf(err, "resolving absolute host path %q", hostPath)
	}

	hostURI := (&url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absoluteHostPath),
	}).String()

	desc := &claircore.LayerDescription{
		Digest:    digest,
		URI:       hostURI,
		MediaType: layerMediaType,
	}

	l := &claircore.Layer{}
	err = l.Init(ctx, desc, nil)
	return l, errors.Wrap(err, "failed to init layer")
}

func scanRepositories(ctx context.Context, cfg VMIndexerConfig, l *claircore.Layer) ([]*claircore.Repository, error) {
	client := cfg.Client
	if client == nil {
		var err error
		client, err = getDefaultClient()
		if err != nil {
			return nil, errors.Wrap(err, "creating repository scanner http client")
		}
	}

	scanner := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		// Do not reach out to the Red Hat Container Catalog API.
		DisableAPI:         true,
		Repo2CPEMappingURL: cfg.Repo2CPEMappingURL,
		Timeout:            cfg.Timeout,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to encode configuration")
	}
	if err := scanner.Configure(ctx, json.NewDecoder(&buf).Decode, client); err != nil {
		return nil, errors.Wrap(err, "failed to configure repository scanner")
	}

	repos, err := scanner.Scan(ctx, l)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan repositories")
	}
	for i, r := range repos {
		r.ID = strconv.Itoa(i)
	}
	return repos, nil
}

func scanPackages(ctx context.Context, packageDBFilter string, layer *claircore.Layer) ([]*claircore.Package, error) {
	scanner := rhel.PackageScanner{}
	pkgs, err := scanner.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RHEL scanner")
	}

	// Filter out packages if filter is specified
	if packageDBFilter == "" {
		return pkgs, nil
	}

	filtered := pkgs[:0]
	for _, pkg := range pkgs {
		if pkg.PackageDB == packageDBFilter {
			filtered = append(filtered, pkg)
		}
	}
	return filtered, nil
}

func coalesceReport(ctx context.Context, digest claircore.Digest, repos []*claircore.Repository, pkgs []*claircore.Package) (*claircore.IndexReport, error) {
	layerArtifacts := []*ccindexer.LayerArtifacts{
		{
			Hash:  digest,
			Repos: repos,
			Pkgs:  pkgs,
		},
	}

	coalescer := rhel.Coalescer{}
	return coalescer.Coalesce(ctx, layerArtifacts)
}
