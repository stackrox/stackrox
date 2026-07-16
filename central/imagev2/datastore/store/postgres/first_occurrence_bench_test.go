//go:build sql_integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	convertutils "github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BenchmarkFirstImageOccurrence compares the selective column read
// (getImageCVETimestamps) against the previous full-deserialization approach
// for preserving FirstImageOccurrence timestamps during image rescans.
//
// Three cases exercise realistic overlap patterns:
//   - 250 unique CVEs, no overlap (each component has distinct CVEs)
//   - 100 unique CVEs across 350 rows (~3.5 rows per CVE, partial overlap)
//   - 500 unique CVEs across 750 rows (~1.5 rows per CVE, moderate overlap)
func BenchmarkFirstImageOccurrence(b *testing.B) {
	if !features.FlattenImageData.Enabled() {
		b.Setenv("ROX_FLATTEN_IMAGE_DATA", "true")
	}

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)
	s := New(testDB.DB, false, concurrency.NewKeyFence())
	storeImpl := s.(*storeImpl)

	for _, tc := range []struct {
		name          string
		numComponents int
		vulnsPerComp  int
		cvePoolSize   int
	}{
		// No overlap: 50 components × 5 CVEs, pool of 250 → each CVE appears once
		{"250rows_no_overlap", 50, 5, 250},
		// Partial overlap: 50 components × 7 CVEs, pool of 100 → each CVE appears ~3.5 times
		{"350rows_100cves", 50, 7, 100},
		// Moderate overlap at higher scale: 75 components × 10 CVEs, pool of 500
		{"750rows_500cves", 75, 10, 500},
	} {
		image := buildBenchImage(tc.numComponents, tc.vulnsPerComp, tc.cvePoolSize)
		require.NoError(b, s.Upsert(ctx, image))

		// Count actual unique CVEs to validate the expected GROUP BY reduction.
		tx, txCtx, err := storeImpl.begin(ctx)
		require.NoError(b, err)
		actualUnique, err := getImageCVETimestamps(txCtx, tx, image.GetId(), NewImageIDField)
		require.NoError(b, err)
		require.NoError(b, tx.Rollback(ctx))
		uniqueCount := len(actualUnique)

		var totalRows int
		tx, txCtx, err = storeImpl.begin(ctx)
		require.NoError(b, err)
		require.NoError(b, tx.QueryRow(txCtx,
			"SELECT COUNT(*) FROM "+imageComponentsV2CVEsTable+" WHERE imageidv2 = $1",
			image.GetId()).Scan(&totalRows))
		require.NoError(b, tx.Rollback(ctx))

		b.Logf("%s: %d total rows, %d unique CVEs (%.1fx reduction from GROUP BY)",
			tc.name, totalRows, uniqueCount, float64(totalRows)/float64(uniqueCount))

		b.Run(tc.name+"/SelectiveColumnRead", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tx, txCtx, err := storeImpl.begin(ctx)
				require.NoError(b, err)
				result, err := getImageCVETimestamps(txCtx, tx, image.GetId(), NewImageIDField)
				require.NoError(b, err)
				require.Len(b, result, uniqueCount)
				require.NoError(b, tx.Rollback(ctx))
			}
		})

		b.Run(tc.name+"/FullDeserialization", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				tx, txCtx, err := storeImpl.begin(ctx)
				require.NoError(b, err)
				result, err := getImageCVETimestampsFull(txCtx, tx, image.GetId(), NewImageIDField)
				require.NoError(b, err)
				require.Len(b, result, uniqueCount)
				require.NoError(b, tx.Rollback(ctx))
			}
		})
	}
}

// getImageCVETimestampsFull reproduces the old approach: SELECT serialized,
// deserialize every row into a full ImageCVEV2 proto, convert to
// EmbeddedVulnerability, then extract timestamps. This is the baseline.
func getImageCVETimestampsFull(ctx context.Context, tx *postgres.Tx, imageID string, imageIDField string) (map[string]*timestamppb.Timestamp, error) {
	rows, err := tx.Query(ctx,
		"SELECT serialized FROM "+imageComponentsV2CVEsTable+" WHERE "+imageIDField+" = $1",
		imageID)
	if err != nil {
		return nil, err
	}

	imageCVEs, err := pgutils.ScanRows[storage.ImageCVEV2, *storage.ImageCVEV2](rows)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*timestamppb.Timestamp, len(imageCVEs))
	for _, cve := range imageCVEs {
		vuln := convertutils.ImageCVEV2ToEmbeddedVulnerability(cve)
		ts := vuln.GetFirstImageOccurrence()
		if ts == nil {
			continue
		}
		if existing, ok := result[vuln.GetCve()]; !ok || protoutils.After(existing, ts) {
			result[vuln.GetCve()] = ts
		}
	}
	return result, nil
}

// buildBenchImage creates an image with partial CVE overlap across components.
// Each component draws vulnsPerComponent CVEs from a shared pool of cvePoolSize
// unique CVEs using a sliding window. This produces realistic overlap: adjacent
// components share some CVEs (like packages depending on the same library),
// while distant components have mostly distinct CVEs.
func buildBenchImage(numComponents, vulnsPerComponent, cvePoolSize int) *storage.ImageV2 {
	now := time.Now()
	stride := max(1, cvePoolSize/numComponents)

	components := make([]*storage.EmbeddedImageScanComponent, 0, numComponents)
	for i := 0; i < numComponents; i++ {
		vulns := make([]*storage.EmbeddedVulnerability, 0, vulnsPerComponent)
		for j := 0; j < vulnsPerComponent; j++ {
			cveIdx := (i*stride + j) % cvePoolSize
			ts := now.Add(-time.Duration(cveIdx) * time.Hour)
			vulns = append(vulns, &storage.EmbeddedVulnerability{
				Cve:                   fmt.Sprintf("CVE-2024-%05d", cveIdx),
				Cvss:                  5.0 + float32(j),
				Severity:              storage.VulnerabilitySeverity(1 + int32(j%4)),
				VulnerabilityType:     storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
				VulnerabilityTypes:    []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_IMAGE_VULNERABILITY},
				Summary:               fmt.Sprintf("A vulnerability in component %d affecting libfoo", i),
				Link:                  fmt.Sprintf("https://nvd.nist.gov/vuln/detail/CVE-2024-%05d", cveIdx),
				SetFixedBy:            &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.0.1"},
				FirstImageOccurrence:  protocompat.ConvertTimeToTimestampOrNil(&ts),
				FirstSystemOccurrence: protocompat.ConvertTimeToTimestampOrNil(&ts),
			})
		}
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    fmt.Sprintf("pkg-%d", i),
			Version: fmt.Sprintf("%d.0.0", i),
			Vulns:   vulns,
		})
	}

	id := uuid.NewV4().String()
	return &storage.ImageV2{
		Id:     id,
		Digest: "sha256:bench" + id[:8],
		Name: &storage.ImageName{
			Registry: "registry.example.com",
			Remote:   "bench/image",
			Tag:      "latest",
			FullName: "registry.example.com/bench/image:latest",
		},
		Scan: &storage.ImageScan{
			ScanTime:        protocompat.TimestampNow(),
			OperatingSystem: "rhel:9",
			Components:      components,
		},
	}
}
