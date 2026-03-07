package enricher

import (
	"context"

	imageCVEInfoDS "github.com/stackrox/rox/central/cve/image/info/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

type enricherImpl struct {
	imageCVEInfoDS imageCVEInfoDS.DataStore
}

// New creates a new CVEInfoEnricher.
func New(imageCVEInfoDS imageCVEInfoDS.DataStore) imageEnricher.CVEInfoEnricher {
	return &enricherImpl{
		imageCVEInfoDS: imageCVEInfoDS,
	}
}

// EnrichImageWithCVEInfo enriches a V1 image's CVEs with timing metadata.
func (e *enricherImpl) EnrichImageWithCVEInfo(ctx context.Context, image *storage.Image) error {
	if !features.CVEFixTimestampCriteria.Enabled() {
		return nil
	}

	scan := image.GetScan()
	if scan == nil {
		return nil
	}

	// Populate the ImageCVEInfo table with CVE timing metadata
	if err := e.upsertImageCVEInfos(ctx, scan); err != nil {
		return err
	}

	// Enrich the CVEs with accurate timestamps from lookup table
	return e.enrichCVEsFromImageCVEInfo(ctx, scan)
}

// EnrichImageV2WithCVEInfo enriches a V2 image's CVEs with timing metadata.
func (e *enricherImpl) EnrichImageV2WithCVEInfo(ctx context.Context, image *storage.ImageV2) error {
	if !features.CVEFixTimestampCriteria.Enabled() {
		return nil
	}

	scan := image.GetScan()
	if scan == nil {
		return nil
	}

	// Populate the ImageCVEInfo table with CVE timing metadata
	if err := e.upsertImageCVEInfos(ctx, scan); err != nil {
		return err
	}

	// Enrich the CVEs with accurate timestamps from lookup table
	return e.enrichCVEsFromImageCVEInfo(ctx, scan)
}

// upsertImageCVEInfos populates the ImageCVEInfo lookup table with CVE timing metadata.
func (e *enricherImpl) upsertImageCVEInfos(ctx context.Context, scan *storage.ImageScan) error {
	infos := make([]*storage.ImageCVEInfo, 0)
	now := protocompat.TimestampNow()

	for _, component := range scan.GetComponents() {
		for _, vuln := range component.GetVulns() {
			// Determine fix available timestamp: use scanner-provided value if available,
			// otherwise fabricate from scan time if the CVE is fixable (has a fix version).
			// This handles non-Red Hat data sources that don't provide fix timestamps.
			fixAvailableTimestamp := vuln.GetFixAvailableTimestamp()
			if fixAvailableTimestamp == nil && vuln.GetFixedBy() != "" {
				fixAvailableTimestamp = now
			}

			// Use scanner-provided FirstSystemOccurrence if available, otherwise use current time.
			// The datastore upsert will preserve the earliest timestamp across scans.
			firstSystemOccurrenceTimestamp := vuln.GetFirstSystemOccurrence()
			if firstSystemOccurrenceTimestamp == nil {
				firstSystemOccurrenceTimestamp = now
			}

			info := &storage.ImageCVEInfo{
				Id:                    cve.ImageCVEInfoID(vuln.GetCve(), component.GetName(), vuln.GetDatasource()),
				Cve:                   vuln.GetCve(),
				FixAvailableTimestamp: fixAvailableTimestamp,
				FirstSystemOccurrence: firstSystemOccurrenceTimestamp,
			}
			infos = append(infos, info)
		}
	}

	if len(infos) == 0 {
		return nil
	}

	return e.imageCVEInfoDS.UpsertMany(sac.WithAllAccess(ctx), infos)
}

// enrichCVEsFromImageCVEInfo enriches the image's CVEs with accurate timestamps from the lookup table.
func (e *enricherImpl) enrichCVEsFromImageCVEInfo(ctx context.Context, scan *storage.ImageScan) error {
	cveNames := set.NewStringSet()
	for _, component := range scan.GetComponents() {
		for _, vuln := range component.GetVulns() {
			cveNames.Add(vuln.GetCve())
		}
	}

	if cveNames.Cardinality() == 0 {
		return nil
	}

	// Collect timestamps and aggregate the min timestamp from CVE infos with the same CVE name.
	// Build two maps from the same result set:
	// - CVE names -> Min timestamp -> FirstSystemOccurrence
	// - CVE info IDs (in this scan)-> ImageCVEInfo -> FixAvailableTimestamp
	allInfos, err := e.imageCVEInfoDS.GetByCVENames(sac.WithAllAccess(ctx), cveNames.AsSlice())
	if err != nil {
		return err
	}

	cveMinTimestamps := make(map[string]*protocompat.Timestamp)
	infoMap := make(map[string]*storage.ImageCVEInfo)

	for _, info := range allInfos {
		// Track MIN FirstSystemOccurrence per CVE name
		cveName := info.GetCve()
		timestamp := info.GetFirstSystemOccurrence()
		if timestamp != nil {
			if existing, ok := cveMinTimestamps[cveName]; !ok || protocompat.CompareTimestamps(timestamp, existing) < 0 {
				cveMinTimestamps[cveName] = timestamp
			}
		}
		infoMap[info.GetId()] = info
	}

	// Enrich CVEs
	for _, component := range scan.GetComponents() {
		for _, vuln := range component.GetVulns() {
			if minTimestamp, ok := cveMinTimestamps[vuln.GetCve()]; ok {
				vuln.FirstSystemOccurrence = minTimestamp
			}

			id := cve.ImageCVEInfoID(vuln.GetCve(), component.GetName(), vuln.GetDatasource())
			if info, ok := infoMap[id]; ok {
				if vuln.GetFixAvailableTimestamp() == nil && vuln.GetFixedBy() != "" {
					// Set the fix timestamp if it was not provided by the scanner
					vuln.FixAvailableTimestamp = info.GetFixAvailableTimestamp()
				}
			}
		}
	}

	return nil
}
