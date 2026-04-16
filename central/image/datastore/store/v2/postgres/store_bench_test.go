//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

// BenchmarkRealisticPopulateImage benchmarks image retrieval with realistic component/CVE counts.
// 50 images x 100 components x 10 CVEs = 50,000 CVEs across 5,000 components.
func BenchmarkRealisticPopulateImage(b *testing.B) {
	if features.FlattenImageData.Enabled() {
		b.Skip("Skipping benchmark - FlattenImageData is enabled.")
	}
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	store := New(testDB.DB, false, concurrency.NewKeyFence())

	numImages := 50
	numComponents := 100
	numCVEs := 10
	images := make([]*storage.Image, 0, numImages)
	for i := 0; i < numImages; i++ {
		components := make([]*storage.EmbeddedImageScanComponent, 0, numComponents)
		for j := 0; j < numComponents; j++ {
			vulns := make([]*storage.EmbeddedVulnerability, 0, numCVEs)
			for k := 0; k < numCVEs; k++ {
				vulns = append(vulns, &storage.EmbeddedVulnerability{
					Cve:               fmt.Sprintf("CVE-2024-%d%d%d", i, j, k),
					Cvss:              5,
					VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
					Severity:          storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
					SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.0.0"},
				})
			}
			components = append(components, &storage.EmbeddedImageScanComponent{
				Name:    fmt.Sprintf("pkg-%d-%d", i, j),
				Version: fmt.Sprintf("%d.0.0", j),
				Vulns:   vulns,
			})
		}
		img := fixtures.GetImageWithUniqueComponents(0)
		img.Id = fmt.Sprintf("sha256:realistic-%d", i)
		img.Scan = &storage.ImageScan{
			Components: components,
		}
		images = append(images, img)
	}

	for _, image := range images {
		require.NoError(b, store.Upsert(ctx, image))
	}

	b.Run("WalkByQuery", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := store.WalkByQuery(ctx, search.EmptyQuery(), func(image *storage.Image) error {
				count++
				return nil
			})
			require.NoError(b, err)
		}
	})

	b.Run("GetSingleImage", func(b *testing.B) {
		for b.Loop() {
			_, _, err := store.Get(ctx, "sha256:realistic-0")
			require.NoError(b, err)
		}
	})
}

// TODO(ROX-30117): Remove this benchmark when FlattenImageData feature flag is removed.
// BenchmarkWalkComparison benchmarks both Walk functions for comparison
func BenchmarkWalkComparison(b *testing.B) {
	if features.FlattenImageData.Enabled() {
		b.Skip("Skipping benchmark - FlattenImageData is enabled.")
	}
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	store := New(testDB.DB, false, concurrency.NewKeyFence())

	// Setup: Insert test images
	numImages := 100
	images := make([]*storage.Image, 0, numImages)
	for i := 0; i < numImages; i++ {
		img := fixtures.GetImageWithUniqueComponents(5)
		img.Id = fmt.Sprintf("%d", i)
		images = append(images, img)
	}

	for _, image := range images {
		require.NoError(b, store.Upsert(ctx, image))
	}

	b.Run("WalkByQuery", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := store.WalkByQuery(ctx, search.EmptyQuery(), func(image *storage.Image) error {
				count++
				return nil
			})
			require.NoError(b, err)
		}
	})

	b.Run("WalkMetadataByQuery", func(b *testing.B) {
		for b.Loop() {
			count := 0
			err := store.WalkMetadataByQuery(ctx, search.EmptyQuery(), func(image *storage.Image) error {
				count++
				return nil
			})
			require.NoError(b, err)
		}
	})
}
