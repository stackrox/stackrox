//go:build sql_integration

package imagecve

import (
	"context"
	"fmt"
	"strings"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	types2 "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/require"
)

const (
	benchPageSize = 100

	// Rows walk the CVE universe with this step. The stride only has to be
	// coprime with the universe size so consecutive images land on fresh CVEs.
	benchCVEStride = 7

	// 70% of images are referenced by deployments; the remaining 30% are inactive.
	benchActiveImagePercent = 70

	// The Image:<name> filter's image gets at most this many distinct CVEs
	// (two rows each: critical/fixable and important/non-fixable).
	benchTargetMaxCVEs = 1_000
)

// benchDataset is fully specified by these four numbers; everything else
// (active/inactive split, target image, rows per image) is derived.
type benchDataset struct {
	numImages      int
	numDeployments int
	totalCVERows   int
	// numCVEs is the size of the CVE universe. It does not have to divide the
	// row count; every filter will match a different subset, as in real data.
	numCVEs int
	// disableParallelWorkers is only needed for the huge dataset: a parallel hash
	// aggregate over 10M rows overflows the test container's 64MB /dev/shm.
	disableParallelWorkers bool
}

var (
	// 500 images / 100 deployments / 50,000 CVE rows / 400-CVE universe.
	smallDataset = benchDataset{numImages: 500, numDeployments: 100, totalCVERows: 50_000, numCVEs: 400}
	// 5,000 images / 1,000 deployments / 1,000,000 CVE rows / 5,000-CVE universe.
	largeDataset = benchDataset{numImages: 5_000, numDeployments: 1_000, totalCVERows: 1_000_000, numCVEs: 5_000}
	// 15,000 images / 5,000 deployments / 10,000,000 CVE rows / 10,000-CVE universe.
	hugeDataset = benchDataset{
		numImages: 15_000, numDeployments: 5_000, totalCVERows: 10_000_000, numCVEs: 10_000,
		disableParallelWorkers: true,
	}
)

func (d benchDataset) activeImages() int         { return d.numImages * benchActiveImagePercent / 100 }
func (d benchDataset) platformActiveImages() int { return d.activeImages() / 2 }

// targetImageIndex is the first active non-platform image; the Image:<name>
// filter points at it.
func (d benchDataset) targetImageIndex() int { return d.platformActiveImages() }

type benchFilter struct {
	name  string
	query *v1.Query
}

type benchVariant struct {
	name string
	view *imageCVECoreViewImpl
}

type benchEnv struct {
	ctx     context.Context
	withTag benchVariant
	noTag   benchVariant
}

func (e *benchEnv) allVariants() []benchVariant {
	return []benchVariant{e.withTag, e.noTag}
}

// BenchmarkImageCVEViewFilterSQL is the small-scale baseline using the
// severity-count multisort, comparing the schema with and without the search tag.
func BenchmarkImageCVEViewFilterSQL(b *testing.B) {
	env := setupBenchmarkEnv(b, smallDataset)
	runBenchmarks(b, env, benchmarkFilters(smallDataset), benchmarkSortPagination(), []int32{0}, views.ReadOptions{}, env.allVariants())
}

// BenchmarkImageCVEViewLargeScale runs the four query strategies against the
// 1M-row dataset.
func BenchmarkImageCVEViewLargeScale(b *testing.B) {
	benchmarkAffectedImagesSets(b, largeDataset, true)
}

// BenchmarkImageCVEViewHugeScale runs the same strategies against the 10M-row
// dataset, with and without the search tag.
func BenchmarkImageCVEViewHugeScale(b *testing.B) {
	benchmarkAffectedImagesSets(b, hugeDataset, true)
}

// benchmarkAffectedImagesSets compares the default query (severity-count multisort)
// against the multisort removed and against adding Severity/Fixable filters. All
// runs use the search tag and the affected-images sort, and measure the first four
// page loads of a result set that is many pages long.
func benchmarkAffectedImagesSets(b *testing.B, ds benchDataset, includeNoTag bool) {
	env := setupBenchmarkEnv(b, ds)
	variants := []benchVariant{env.withTag}
	if includeNoTag {
		variants = env.allVariants()
	}
	sets := []struct {
		name       string
		filters    []benchFilter
		pagination *v1.QueryPagination
		options    views.ReadOptions
	}{
		{
			name:       "no-extra-filters-multisort",
			filters:    benchmarkFilters(ds),
			pagination: benchmarkSortPagination(),
		},
		{
			name:       "no-extra-filters-affected-images",
			filters:    benchmarkFilters(ds),
			pagination: affectedImagesPagination(),
		},
		{
			name: "critical-fixable-affected-images",
			filters: addSeverityAndFixable(benchmarkFilters(ds),
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, true),
			pagination: affectedImagesPagination(),
			// The severity breakdown is not rendered when the user has already
			// narrowed the view to one severity/fixability combination.
			options: views.ReadOptions{SkipGetImagesBySeverity: true},
		},
		{
			name: "important-not-fixable-affected-images",
			filters: addSeverityAndFixable(benchmarkFilters(ds),
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY, false),
			pagination: affectedImagesPagination(),
			options:    views.ReadOptions{SkipGetImagesBySeverity: true},
		},
	}
	firstFourPageOffsets := []int32{0, benchPageSize, 2 * benchPageSize, 3 * benchPageSize}
	for _, set := range sets {
		b.Run(set.name, func(b *testing.B) {
			runBenchmarks(b, env, set.filters, set.pagination, firstFourPageOffsets, set.options, variants)
		})
	}
}

func setupBenchmarkEnv(b *testing.B, ds benchDataset) *benchEnv {
	b.Setenv(features.FlattenImageData.EnvVar(), "true")
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)
	imageStore := imageV2DS.GetTestPostgresDataStore(b, testDB.DB)

	images := benchmarkImages(ds)
	for _, image := range images {
		require.NoError(b, imageStore.UpsertImage(ctx, image))
	}
	createBenchmarkDeployments(b, ctx, deploymentStore, images, ds)

	var cveRowCount int
	require.NoError(b, testDB.DB.QueryRow(ctx, "select count(*) from image_cves_v2").Scan(&cveRowCount))
	b.Logf("dataset: %d images (%d active, %d inactive), %d deployments, %d image_cves_v2 rows",
		ds.numImages, ds.activeImages(), ds.numImages-ds.activeImages(), ds.numDeployments, cveRowCount)
	require.Equal(b, ds.totalCVERows, cveRowCount)

	// Views get their own pool with parallel workers disabled (see benchmarkDBConfig).
	benchDB, err := postgres.New(ctx, benchmarkDBConfig(testDB, ds))
	require.NoError(b, err)
	b.Cleanup(benchDB.Close)

	return &benchEnv{
		ctx: ctx,
		withTag: benchVariant{
			name: "with-search-tag",
			view: &imageCVECoreViewImpl{db: benchDB, schema: pgSchema.ImageCvesV2Schema},
		},
		noTag: benchVariant{
			name: "without-search-tag",
			// Simulate the schema without search:"Image ID" on image_id_v2 by
			// dropping that field, so "Image ID" resolves via the images_v2 join.
			view: &imageCVECoreViewImpl{db: benchDB, schema: schemaWithoutImageIDField(b, pgSchema.ImageCvesV2Schema)},
		},
	}
}

func createBenchmarkDeployments(b *testing.B, ctx context.Context, store deploymentDS.DataStore, images []*storage.ImageV2, ds benchDataset) {
	platformActive := ds.platformActiveImages()
	active := ds.activeImages()
	platformDeployments := ds.numDeployments / 2

	for d := 0; d < ds.numDeployments; d++ {
		platform := d < platformDeployments
		start, end := deploymentImageRange(d, platformDeployments, ds.numDeployments, platformActive, active)

		namespace := fmt.Sprintf("bench-ns-%d", d)
		if platform {
			namespace = "kube-" + namespace
		}
		deployment := fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, namespace, images[start])
		deployment.PlatformComponent = platform
		for j := start + 1; j < end; j++ {
			deployment.Containers = append(deployment.Containers, &storage.Container{
				Name:  fmt.Sprintf("bench-container-%d", j),
				Image: types2.ToContainerImageV2(images[j]),
			})
		}
		require.NoError(b, store.UpsertDeployment(ctx, deployment))
	}
}

// deploymentImageRange returns the [start, end) image indexes referenced by
// deployment d. The two halves of the active images belong to the platform and
// non-platform deployments respectively, and are split evenly across them.
func deploymentImageRange(d, platformDeployments, totalDeployments, platformActive, active int) (int, int) {
	base, count, deployments := 0, platformActive, platformDeployments
	if d >= platformDeployments {
		base, count, deployments = platformActive, active-platformActive, totalDeployments-platformDeployments
		d -= platformDeployments
	}
	return base + d*count/deployments, base + (d+1)*count/deployments
}

// benchmarkDBConfig only overrides the default parallel-worker setting for the
// huge dataset, where a parallel hash aggregate overflows the test Postgres
// container's default 64MB /dev/shm. Small and large datasets use the default.
func benchmarkDBConfig(testDB *pgtest.TestPostgres, ds benchDataset) *postgres.Config {
	cfg := testDB.Config().Copy()
	if !ds.disableParallelWorkers {
		return cfg
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	cfg.ConnConfig.RuntimeParams["max_parallel_workers_per_gather"] = "0"
	return cfg
}

func runBenchmarks(b *testing.B, env *benchEnv, filters []benchFilter, pagination *v1.QueryPagination, offsets []int32, options views.ReadOptions, variants []benchVariant) {
	for _, filter := range filters {
		for _, offset := range offsets {
			pageQuery := filter.query.CloneVT()
			pageQuery.Pagination = pagination.CloneVT()
			pageQuery.Pagination.Offset = offset
			page := offset/benchPageSize + 1

			for _, variant := range variants {
				b.Run(fmt.Sprintf("%s/%s/page-%d", variant.name, filter.name, page), func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						res, err := variant.view.Get(env.ctx, pageQuery, options)
						if err != nil {
							b.Fatal(err)
						}
						if len(res) == 0 || len(res) > benchPageSize {
							b.Fatalf("expected 1..%d results, got %d", benchPageSize, len(res))
						}
					}
				})
			}
		}
	}
}

func benchmarkFilters(ds benchDataset) []benchFilter {
	observed := storage.VulnerabilityState_OBSERVED.String()
	deferred := storage.VulnerabilityState_DEFERRED.String()
	targetImage := benchmarkImageName(ds.targetImageIndex())

	return []benchFilter{
		{
			name: "user workload observed",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "false").
				AddExactMatches(search.VulnerabilityState, observed).
				ProtoQuery(),
		},
		{
			name: "user workload deferred",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "false").
				AddExactMatches(search.VulnerabilityState, deferred).
				ProtoQuery(),
		},
		{
			name: "platform observed",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "true").
				AddExactMatches(search.VulnerabilityState, observed).
				ProtoQuery(),
		},
		{
			name: "platform deferred",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "true").
				AddExactMatches(search.VulnerabilityState, deferred).
				ProtoQuery(),
		},
		{
			name: "inactive observed",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, search.NullString).
				AddExactMatches(search.VulnerabilityState, observed).
				ProtoQuery(),
		},
		{
			name: "inactive deferred",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, search.NullString).
				AddExactMatches(search.VulnerabilityState, deferred).
				ProtoQuery(),
		},
		{
			name: "user workload observed for one image",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "false").
				AddExactMatches(search.VulnerabilityState, observed).
				AddExactMatches(search.ImageName, targetImage).
				ProtoQuery(),
		},
		{
			name: "vulnerability state observed",
			query: search.NewQueryBuilder().
				AddExactMatches(search.VulnerabilityState, observed).
				ProtoQuery(),
		},
		{
			name: "platform component any observed",
			query: search.NewQueryBuilder().
				AddStrings(search.PlatformComponent, "true", "false", search.NullString).
				AddExactMatches(search.VulnerabilityState, observed).
				ProtoQuery(),
		},
	}
}

func addSeverityAndFixable(filters []benchFilter, severity storage.VulnerabilitySeverity, fixable bool) []benchFilter {
	out := make([]benchFilter, 0, len(filters))
	for _, filter := range filters {
		extra := search.NewQueryBuilder().
			AddExactMatches(search.Severity, severity.String()).
			AddBools(search.Fixable, fixable).
			ProtoQuery()
		query := search.ConjunctionQuery(filter.query.CloneVT(), extra)
		query.Pagination = filter.query.GetPagination()
		out = append(out, benchFilter{name: filter.name, query: query})
	}
	return out
}

func benchmarkSortPagination() *v1.QueryPagination {
	return search.NewPagination().
		Limit(benchPageSize).
		AddSortOption(search.NewSortOption(search.CriticalSeverityCount).Reversed(true)).
		AddSortOption(search.NewSortOption(search.ImportantSeverityCount).Reversed(true)).
		AddSortOption(search.NewSortOption(search.ModerateSeverityCount).Reversed(true)).
		AddSortOption(search.NewSortOption(search.LowSeverityCount).Reversed(true)).
		AddSortOption(search.NewSortOption(search.UnknownSeverityCount).Reversed(true)).
		Proto()
}

// affectedImagesPagination sorts by distinct affected image count. ImageID
// resolves locally on image_cves_v2 thanks to the search tag, so this sort does
// not force the images_v2 join. (The UI currently sends "Image Sha" here, which
// does force the join.)
func affectedImagesPagination() *v1.QueryPagination {
	return search.NewPagination().
		Limit(benchPageSize).
		AddSortOption(search.NewSortOption(search.ImageID).
			AggregateBy(aggregatefunc.Count, true).
			Reversed(true)).
		Proto()
}

// benchmarkImages builds the synthetic images. Each image has
// (totalCVERows - targetRows) / (numImages - 1) CVE rows, give or take one, so the
// dataset totals exactly ds.totalCVERows.
func benchmarkImages(ds benchDataset) []*storage.ImageV2 {
	severities := []storage.VulnerabilitySeverity{
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
	}

	target := ds.targetImageIndex()
	// The target image gets one Critical+Fixable and one Important+NonFixable row
	// per CVE, so the single-image query returns every one of its CVEs under either
	// severity filter.
	targetCVEs := min(benchTargetMaxCVEs, ds.numCVEs)
	targetRows := 2 * targetCVEs
	regularImages := ds.numImages - 1
	rowsPerImage, extraRowImages := (ds.totalCVERows-targetRows)/regularImages, (ds.totalCVERows-targetRows)%regularImages

	images := make([]*storage.ImageV2, 0, ds.numImages)
	regularIndex := 0
	for i := 0; i < ds.numImages; i++ {
		isTarget := i == target
		rows := rowsPerImage
		if isTarget {
			rows = targetRows
		} else {
			if regularIndex < extraRowImages {
				rows++
			}
			regularIndex++
		}

		vulns := make([]*storage.EmbeddedVulnerability, 0, rows)
		for k := 0; k < rows; k++ {
			var cveIndex int
			var severity storage.VulnerabilitySeverity
			var state storage.VulnerabilityState
			var fixable bool

			if isTarget {
				cveIndex = k % targetCVEs
				state = storage.VulnerabilityState_OBSERVED
				if (k/targetCVEs)%2 == 0 {
					severity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
					fixable = true
				} else {
					severity = storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
					fixable = false
				}
			} else {
				cveIndex = (i*benchCVEStride + k) % ds.numCVEs
				// Advance the phase every two images so a given CVE appears in both
				// states. (With an odd stride, parity of i+k is fixed per CVE.)
				state = storage.VulnerabilityState_OBSERVED
				if (i/2+k)%2 == 1 {
					state = storage.VulnerabilityState_DEFERRED
				}
				severity = severities[(i+k)%len(severities)]
				fixable = (i+k)%3 == 0
			}

			vuln := &storage.EmbeddedVulnerability{
				Cve:      fmt.Sprintf("CVE-2026-%05d", cveIndex),
				Severity: severity,
				State:    state,
			}
			if fixable {
				vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.2.3"}
			}
			vulns = append(vulns, vuln)
		}

		images = append(images, imageUtils.ConvertToV2(&storage.Image{
			Id:      fmt.Sprintf("sha256:bench-%d", i),
			Name:    &storage.ImageName{FullName: benchmarkImageName(i)},
			SetCves: &storage.Image_Cves{Cves: int32(rows)},
			Scan: &storage.ImageScan{
				ScanTime:        protocompat.TimestampNow(),
				OperatingSystem: "bench:1",
				Components: []*storage.EmbeddedImageScanComponent{{
					Name:    "bench-component",
					Version: "1.0",
					Vulns:   vulns,
				}},
			},
		}))
	}
	return images
}

func benchmarkImageName(index int) string {
	return fmt.Sprintf("example.com/bench-%d:latest", index)
}

func schemaWithoutImageIDField(b *testing.B, base *walker.Schema) *walker.Schema {
	cloned := *base
	cloned.Fields = make([]walker.Field, 0, len(base.Fields))
	removed := 0
	for _, field := range base.Fields {
		if strings.EqualFold(field.Search.FieldName, search.ImageID.String()) {
			removed++
			continue
		}
		cloned.Fields = append(cloned.Fields, field)
	}
	require.Equal(b, 1, removed, "expected exactly one %q search field on %s", search.ImageID, base.Table)
	return &cloned
}
