package fake

import (
	"fmt"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

const (
	// vmMockDigest may be kept in sync with its copy in pkg/virtualmachine/enricher/enricher_impl.go
	vmMockDigest = "sha256:900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900dc0ffee900d"
)

// reportGenerator generates fake VM index reports using a constant pre-built template
type reportGenerator struct {
	packages     map[string]*v4.Package
	repositories map[string]*v4.Repository
}

func newReportGenerator(numPackages, numRepos int) *reportGenerator {
	repositories := make(map[string]*v4.Repository, numRepos)
	for i := range numRepos {
		repoID := fmt.Sprintf("repo-template-%d", i)
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
		pkgID := fmt.Sprintf("pkg-template-%d", i)
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
	return &reportGenerator{
		repositories: repositories,
		packages:     packages,
	}
}

// generateFakeIndexReport creates a fake VM index report using the provided generator.
func generateFakeIndexReport(gen *reportGenerator, vsockCID uint32) *v1.IndexReport {
	return &v1.IndexReport{
		VsockCid: fmt.Sprintf("%d", vsockCID),
		IndexV4: &v4.IndexReport{
			HashId:  vmMockDigest,
			State:   "IndexFinished",
			Success: true,
			Contents: &v4.Contents{
				Packages:     gen.packages,
				Repositories: gen.repositories,
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
