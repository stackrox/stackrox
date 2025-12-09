package fake

import (
	"fmt"
	"math/rand"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	precomputedReportVariants = 5
	// reportIntervalJitterPercent defines the percentage of random jitter to apply to report intervals.
	// With 0.05 (5%), a 60s interval will vary between 57s-63s, making timing more realistic.
	reportIntervalJitterPercent = 0.05
	// vmMockDigest may be kept in sync with its copy in pkg/virtualmachine/enricher/enricher_impl.go
	vmMockDigest = "sha256:900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900d"
)

type reportTemplate struct {
	packages     map[string]*v4.Package
	repositories map[string]*v4.Repository
}

// reportGenerator generates fake VM index reports using pre-built templates
type reportGenerator struct {
	templates            []reportTemplate
	currentTemplateMutex sync.Mutex
	currentTemplateIdx   uint32
}

func newReportGenerator(numPackages, numRepos int) *reportGenerator {
	variantCount := precomputedReportVariants
	if variantCount <= 0 {
		variantCount = 1
	}

	templates := make([]reportTemplate, variantCount)
	for variant := range variantCount {
		repositories := make(map[string]*v4.Repository, numRepos)
		for i := range numRepos {
			repoID := fmt.Sprintf("repo-template-%d-%d", variant, i)
			repositories[repoID] = &v4.Repository{
				Id:   repoID,
				Name: fmt.Sprintf("repository-%d", i),
				Uri:  fmt.Sprintf("https://repo%d.example.com", i),
				Key:  fmt.Sprintf("key-%d", i),
				Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*", // valid CPE to also load scanner V4 matcher.
			}
		}

		packages := make(map[string]*v4.Package, numPackages)
		for i := range numPackages {
			pkgID := fmt.Sprintf("pkg-template-%d-%d", variant, i)

			packages[pkgID] = &v4.Package{
				Id:             pkgID,
				Name:           fmt.Sprintf("package%d", i),
				Version:        fmt.Sprintf("1.%d.%d", i/10, i%10),
				Kind:           "binary",
				Arch:           "amd64",
				RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
				Cpe:            "cpe:2.3:a:*:*:*:*:*:*:*:*:*:*",
				PackageDb:      "sqlite:usr/share/rpm",
				Source: &v4.Package{
					Id:      pkgID,
					Name:    fmt.Sprintf("package%d", i),
					Version: fmt.Sprintf("1.%d.%d", i/10, i%10),
					Kind:    "source",
					Cpe:     "cpe:2.3:a:*:*:*:*:*:*:*:*:*:*",
				},
				NormalizedVersion: &v4.NormalizedVersion{
					Kind: "rpm",
					V:    []int32{1, 0, 0},
				},
				Module: fmt.Sprintf("module%d", i),
			}
		}

		templates[variant] = reportTemplate{
			packages:     packages,
			repositories: repositories,
		}
	}

	return &reportGenerator{
		templates: templates,
	}
}

func (g *reportGenerator) nextTemplate() reportTemplate {
	if len(g.templates) == 0 {
		return reportTemplate{
			packages:     make(map[string]*v4.Package),
			repositories: make(map[string]*v4.Repository),
		}
	}
	g.currentTemplateMutex.Lock()
	defer g.currentTemplateMutex.Unlock()
	idx := g.currentTemplateIdx % uint32(len(g.templates))
	g.currentTemplateIdx++
	return g.templates[idx]
}

// generateFakeIndexReport creates a fake VM index report using the provided generator.
func generateFakeIndexReport(gen *reportGenerator, vsockCID uint32) *v1.IndexReport {
	template := gen.nextTemplate()

	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  vmMockDigest,
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     template.packages,
				Repositories: template.repositories,
				Environments: map[string]*v4.Environment_List{
					"1": {
						Environments: []*v4.Environment{
							{
								PackageDb:     "sqlite:usr/share/rpm",
								IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
								RepositoryIds: []string{"cpe:/o:redhat:enterprise_linux:9::fastdatapath", "cpe:/a:redhat:openshift:4.16::el9"},
							},
						},
					},
				},
			},
		},
	}
}

// jitteredInterval returns a duration with random jitter applied.
// The result is in the range [interval * (1 - jitterPercent), interval * (1 + jitterPercent)].
// For example, with interval=60s and jitterPercent=0.05, returns a value between 57s and 63s.
func jitteredInterval(interval time.Duration, jitterPercent float64) time.Duration {
	// Calculate jitter range: interval * jitterPercent
	jitterRange := float64(interval) * jitterPercent
	// Random value in [-jitterRange, +jitterRange]
	jitter := (rand.Float64()*2 - 1) * jitterRange
	// Return interval with jitter applied
	return time.Duration(float64(interval) + jitter)
}
