package scannerv4

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// TypeString is the name of the ScannerV4 scanner.
	// TODO: does this need to stay exported?
	TypeString = "scannerv4"
)

var (
	_ types.Scanner = (*scannerv4)(nil)

	log = logging.LoggerForModule()

	defaultIndexerEndpoint    = fmt.Sprintf("scanner-v4-indexer.%s.svc:8443", env.Namespace.Setting())
	defaultMatcherEndpoint    = fmt.Sprintf("scanner-v4-matcher.%s.svc:8443", env.Namespace.Setting())
	defaultMaxConcurrentScans = int64(30)

	// TODO: make configurable
	scanTimeout = 2 * time.Minute
)

func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return TypeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
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
	log.Info("ScannerV4 - newScanner: %+v", integration)
	// TODO: Can the new scanner client / connection be created lazily? in the event that scannerv4 is not ready
	// 'right now', don't want to fail the creation of the integration so that it isn't skipped going forward.

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
		// TODO: Should we error here? if scanner not yet ready to receive traffic, we'd still want the integration created
		// does this error out if cannot establish connectivity? check
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
	log.Info("ScannerV4 - GetScan")
	// If we haven't retrieved any metadata, then we won't be able to scan with ScannerV4
	// DELME: In scenarios where only a tag (no digest) is available, current ScannerV4 will break
	// per comments
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

	// TODO: Create util method so that central/sensor do not duplicate
	// Beg: copy from sensor's grpc_client
	n := fmt.Sprintf("%s/%s@%s", image.GetName().GetRegistry(), image.GetName().GetRemote(), utils.GetSHA(image))
	ref, err := name.NewDigest(n, opts...)
	if err != nil {
		// TODO: ROX-19576: Is the assumption that images always have SHA correct?
		return nil, fmt.Errorf("creating digest reference: %w", err)
	}
	// End

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	vr, err := s.scannerClient.IndexAndScanImage(ctx, ref, &auth)
	if err != nil {
		return nil, fmt.Errorf("index and scan image report (reference: %q): %w", ref.Name(), err)
	}

	// TODO: convert vr to *storage.ImageScan (take a look at clairv4)
	_ = vr

	// TODO: Implement
	return nil, errors.New("ScannerV4 NOT Implemented")
}

func (s *scannerv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	log.Info("ScannerV4 - GetVulnDefinitionsInfo")
	// TODO(ROX-21040): Implementation dependant on the API existing.
	return nil, errors.New("ScannerV4 - GetVulnDefinitionsInfo NOT Implemented")
}

func (s *scannerv4) Match(image *storage.ImageName) bool {
	r := s.activeRegistries.Match(image)
	// TODO: remove this log entry and return s.activeRegistries.Match(image) once done building
	log.Infof("ScannerV4 - Match for %q == %t", image.GetFullName(), r)
	return r
}

func (s *scannerv4) Name() string {
	log.Info("ScannerV4 - Name")

	return s.name
}

func (s *scannerv4) Test() error {
	log.Info("ScannerV4 - Test")
	// TODO: Implement
	// TODO: gRPC API to test ScannerV4 indexer/matcher health does not yet exist.
	log.Warnf("ScannerV4 - Returning FAKE 'success' to Test")
	return nil
}

func (s *scannerv4) Type() string {
	log.Info("ScannerV4 - Type")
	return TypeString
}
