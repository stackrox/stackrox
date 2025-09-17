package enricher

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/delegatedregistry"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/integrationhealth"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"golang.org/x/time/rate"
)

// ImageEnricherV2 provides functions for enriching images with integrations.
//
//go:generate mockgen-wrapper
type ImageEnricherV2 interface {
	// EnrichImage will enrich an image with its metadata, scan results, signatures and signature verification results.
	EnrichImage(ctx context.Context, enrichCtx EnrichmentContext, image *storage.ImageV2) (EnrichmentResult, error)
	// EnrichWithVulnerabilities will enrich an image with its components and their associated vulnerabilities only.
	// This will always force re-enrichment and not take existing values into account.
	EnrichWithVulnerabilities(image *storage.ImageV2, components *scannerTypes.ScanComponents, notes []scannerV1.Note) (EnrichmentResult, error)
	// EnrichWithSignatureVerificationData will enrich an image with signature verification results only.
	// This will always force re-verification and not take existing values into account.
	EnrichWithSignatureVerificationData(ctx context.Context, image *storage.ImageV2) (EnrichmentResult, error)
}

// ImageGetterV2 will be used to retrieve a specific image from the datastore.
type ImageGetterV2 func(ctx context.Context, id string) (*storage.ImageV2, bool, error)

// NewV2 returns a new ImageEnricherV2 instance for the given subsystem.
// (The subsystem is just used for Prometheus metrics.)
func NewV2(cvesSuppressor CVESuppressor, is integration.Set, subsystem pkgMetrics.Subsystem, metadataCache cache.ImageMetadata,
	imageGetter ImageGetterV2, healthReporter integrationhealth.Reporter,
	signatureIntegrationGetter SignatureIntegrationGetter, scanDelegator delegatedregistry.Delegator) ImageEnricherV2 {
	enricher := &enricherV2Impl{
		cvesSuppressor: cvesSuppressor,
		integrations:   is,

		// number of consecutive errors per registry or scanner to ascertain health of the integration
		errorsPerRegistry:         make(map[registryTypes.ImageRegistry]int32),
		errorsPerScanner:          make(map[scannerTypes.ImageScannerWithDataSource]int32, len(is.ScannerSet().GetAll())),
		integrationHealthReporter: healthReporter,

		metadataLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:   metadataCache,

		signatureIntegrationGetter: signatureIntegrationGetter,
		signatureVerifier:          signatures.VerifyAgainstSignatureIntegrations,
		signatureFetcher:           signatures.NewSignatureFetcher(),

		imageGetter: imageGetter,

		asyncRateLimiter: rate.NewLimiter(rate.Every(1*time.Second), 5),

		metrics: newMetrics(subsystem),

		scanDelegator: scanDelegator,
	}
	return enricher
}
