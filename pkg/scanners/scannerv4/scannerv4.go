package scannerv4

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/scanner/pkg/client"
)

const (
	typeString = "scannerv4"
)

var (
	_ types.Scanner = (*scannerv4)(nil)

	log = logging.LoggerForModule()

	defaultIndexerEndpoint    = fmt.Sprintf("scanner-v4-indexer.%s.svc:8443", env.Namespace.Setting())
	defaultMatcherEndpoint    = fmt.Sprintf("scanner-v4-matcher.%s.svc:8443", env.Namespace.Setting())
	defaultMaxConcurrentScans = int64(30)

	scanTimeout = env.ScanTimeout.DurationSetting()
)

func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration, set)
		return scan, err
	}
}

type scannerv4 struct {
	types.ScanSemaphore

	name             string
	activeRegistries registries.Set
	scannerClient    client.Scanner
}

func newScanner(integration *storage.ImageIntegration, activeRegistries registries.Set) (*scannerv4, error) {
	conf := integration.GetScannerV4()
	if conf == nil {
		return nil, errors.New("ScannerV4 configuration required")
	}

	indexerEndpoint := defaultIndexerEndpoint
	if conf.IndexerEndpoint != "" {
		indexerEndpoint = conf.IndexerEndpoint
	}

	matcherEndpoint := defaultMatcherEndpoint
	if conf.IndexerEndpoint != "" {
		matcherEndpoint = conf.MatcherEndpoint
	}

	numConcurrentScans := defaultMaxConcurrentScans
	if conf.GetNumConcurrentScans() != 0 {
		numConcurrentScans = int64(conf.GetNumConcurrentScans())
	}

	log.Debugf("Creating ScannerV4 with name [%s] indexer address [%s], matcher address [%s], num concurrent scans [%d]", integration.GetName(), indexerEndpoint, matcherEndpoint, numConcurrentScans)
	ctx := context.Background()
	c, err := client.NewGRPCScanner(ctx,
		client.WithIndexerAddress(indexerEndpoint),
		client.WithMatcherAddress(matcherEndpoint),
		// TODO(ROX-19050): Set the Scanner V4 TLS validation when certificates are ready.
		// client.SkipTLSVerification,
	)

	if err != nil {
		return nil, err
	}

	scanner := &scannerv4{
		name:             integration.GetName(),
		activeRegistries: activeRegistries,
		ScanSemaphore:    types.NewSemaphoreWithValue(numConcurrentScans),
		scannerClient:    c,
	}

	return scanner, nil
}

func (s *scannerv4) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image.GetMetadata() == nil {
		return nil, nil
	}

	rc := s.activeRegistries.GetRegistryMetadataByImage(image)
	if rc == nil {
		log.Debugf("No registry matched during scan of %q", image.GetName().GetFullName())
		return nil, nil
	}

	var opts []name.Option
	if rc.Insecure {
		opts = append(opts, name.Insecure)
	}

	auth := authn.Basic{
		Username: rc.Username,
		Password: rc.Password,
	}

	// TODO: Consider creating util method so that central/sensor do not duplicate this logic
	// Beg: copy from sensor's grpc_client.
	n := fmt.Sprintf("%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), utils.GetSHA(image))
	digest, err := name.NewDigest(n, opts...)
	if err != nil {
		// TODO: ROX-19576: Is the assumption that images always have SHA correct?
		return nil, fmt.Errorf("creating digest reference: %w", err)
	}
	// End

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	vr, err := s.scannerClient.IndexAndScanImage(ctx, digest, &auth)
	if err != nil {
		return nil, fmt.Errorf("index and scan image report (reference: %q): %w", digest.Name(), err)
	}

	log.Debugf("Vuln report received for %q (hash %q): %d dists, %d envs, %d pkgs, %d repos, %d pkg vulns, %d vulns",
		image.GetName().GetFullName(),
		vr.GetHashId(),
		len(vr.GetContents().GetDistributions()),
		len(vr.GetContents().GetEnvironments()),
		len(vr.GetContents().GetPackages()),
		len(vr.GetContents().GetRepositories()),
		len(vr.GetPackageVulnerabilities()),
		len(vr.GetVulnerabilities()),
	)

	return imageScan(vr), nil
}

func (s *scannerv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	// TODO(ROX-21040): Implementation dependant on the API existing.
	return nil, errors.New("ScannerV4 - GetVulnDefinitionsInfo NOT Implemented")
}

func (s *scannerv4) Match(image *storage.ImageName) bool {
	return s.activeRegistries.Match(image)
}

func (s *scannerv4) Name() string {
	return s.name
}

func (s *scannerv4) Test() error {
	// TODO(ROX-20624): Dependant on the matcher/indexer test endpoints being avail.
	log.Warn("ScannerV4 - Returning FAKE 'success' to Test")
	return nil
}

func (s *scannerv4) Type() string {
	return typeString
}
