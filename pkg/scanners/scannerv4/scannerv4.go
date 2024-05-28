package scannerv4

import (
	"context"
	"errors"
	"fmt"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/types"
	pkgscanner "github.com/stackrox/rox/pkg/scannerv4"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	scannerv1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	_ types.Scanner                  = (*scannerv4)(nil)
	_ types.ImageVulnerabilityGetter = (*scannerv4)(nil)

	log = logging.LoggerForModule()

	// DefaultIndexerEndpoint is the default gRPC endpoint for the indexer.
	DefaultIndexerEndpoint = fmt.Sprintf("scanner-v4-indexer.%s.svc:8443", env.Namespace.Setting())

	// DefaultMatcherEndpoint is the default gRPC endpoint for the matcher.
	DefaultMatcherEndpoint = fmt.Sprintf("scanner-v4-matcher.%s.svc:8443", env.Namespace.Setting())

	defaultMaxConcurrentScans = int64(30)

	scanTimeout     = env.ScanTimeout.DurationSetting()
	metadataTimeout = 1 * time.Minute

	// zeroProtoTimestampFromTime represents the zero value of a proto
	// timestamp when initialized from the zero time.
	zeroProtoTimestampFromTime, _ = protoTypes.TimestampProto(time.Time{})
)

// Creator provides the type scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return types.ScannerV4, func(integration *storage.ImageIntegration) (types.Scanner, error) {
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
		return nil, errors.New("scanner V4 configuration required")
	}

	indexerEndpoint := DefaultIndexerEndpoint
	if conf.IndexerEndpoint != "" {
		indexerEndpoint = conf.IndexerEndpoint
	}

	matcherEndpoint := DefaultMatcherEndpoint
	if conf.MatcherEndpoint != "" {
		matcherEndpoint = conf.MatcherEndpoint
	}

	numConcurrentScans := defaultMaxConcurrentScans
	if conf.GetNumConcurrentScans() > 0 {
		numConcurrentScans = int64(conf.GetNumConcurrentScans())
	}

	log.Debugf("Creating Scanner V4 with name [%s] indexer address [%s], matcher address [%s], num concurrent scans [%d]", integration.GetName(), indexerEndpoint, matcherEndpoint, numConcurrentScans)
	ctx := context.Background()
	c, err := client.NewGRPCScanner(ctx,
		client.WithIndexerAddress(indexerEndpoint),
		client.WithMatcherAddress(matcherEndpoint),
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

	auth := authn.Basic{
		Username: rc.Username,
		Password: rc.Password,
	}

	digest, err := pkgscanner.DigestFromImage(image)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: rc.GetInsecure()}
	vr, err := s.scannerClient.IndexAndScanImage(ctx, digest, &auth, opt)
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

	return imageScan(image.GetMetadata(), vr), nil
}

func (s *scannerv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), metadataTimeout)
	defer cancel()

	metadata, err := s.scannerClient.GetMatcherMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("pulling metadata from matcher: %w", err)
	}

	lastTms := metadata.GetLastVulnerabilityUpdate()
	if lastTms == nil || lastTms.Equal(protoTypes.Timestamp{}) || lastTms.Equal(zeroProtoTimestampFromTime) {
		return nil, errors.New("no timestamp available")
	}

	return &v1.VulnDefinitionsInfo{
		LastUpdatedTimestamp: lastTms,
	}, nil
}

func (s *scannerv4) Match(image *storage.ImageName) bool {
	return s.activeRegistries.Match(image)
}

func (s *scannerv4) Name() string {
	return s.name
}

func (s *scannerv4) Test() error {
	// TODO(ROX-20624): Dependent on the matcher/indexer test endpoints being avail.
	log.Warn("ScannerV4 - Returning FAKE 'success' to Test")
	return nil
}

func (s *scannerv4) Type() string {
	return types.ScannerV4
}

func (s *scannerv4) GetVulnerabilities(image *storage.Image, components *types.ScanComponents, _ []scannerv1.Note) (*storage.ImageScan, error) {
	v4Contents := components.ScannerV4()

	digest, err := pkgscanner.DigestFromImage(image)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	vr, err := s.scannerClient.GetVulnerabilities(ctx, digest, v4Contents)
	if err != nil {
		return nil, fmt.Errorf("get vulnerability report (reference: %q): %w", digest.Name(), err)
	}

	log.Debugf("Vuln report (match) received for %q (hash %q): %d dists, %d envs, %d pkgs, %d repos, %d pkg vulns, %d vulns",
		image.GetName().GetFullName(),
		vr.GetHashId(),
		len(vr.GetContents().GetDistributions()),
		len(vr.GetContents().GetEnvironments()),
		len(vr.GetContents().GetPackages()),
		len(vr.GetContents().GetRepositories()),
		len(vr.GetPackageVulnerabilities()),
		len(vr.GetVulnerabilities()),
	)

	return imageScan(image.GetMetadata(), vr), nil
}
