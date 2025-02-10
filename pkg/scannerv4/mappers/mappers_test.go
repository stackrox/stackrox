package mappers

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	nvdschema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/toolkit/types/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stackrox/rox/pkg/scannerv4/updater/manual"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	emptyCPE = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
)

func Test_ToProtoV4IndexReport(t *testing.T) {
	tests := []struct {
		name    string
		arg     *claircore.IndexReport
		want    *v4.IndexReport
		wantErr string
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when default values then contents is defined",
			arg:  &claircore.IndexReport{},
			want: &v4.IndexReport{Contents: &v4.Contents{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToProtoV4IndexReport(tt.arg)
			if tt.wantErr != "" {
				assert.Nil(t, tt.want)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				protoassert.Equal(t, tt.want, got)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ToProtoV4VulnerabilityReport(t *testing.T) {
	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	require.NoError(t, err)

	tests := map[string]struct {
		arg     *claircore.VulnerabilityReport
		want    *v4.VulnerabilityReport
		wantErr string
	}{
		"when nil then nil": {},
		"when default values then attributes are defined": {
			arg:  &claircore.VulnerabilityReport{},
			want: &v4.VulnerabilityReport{Contents: &v4.Contents{}},
		},
		"when invalid time in vulnerability map then nil issued": {
			arg: &claircore.VulnerabilityReport{
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"sample CVE": {
						ID: "sample CVE",
						// Timestamp lower than epoch is invalid.
						Issued: time.Time{}.Add(-time.Hour),
					},
				},
			},
			want: &v4.VulnerabilityReport{
				Contents: &v4.Contents{},
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"sample CVE": {
						Id: "sample CVE",
					},
				}},
		},
		"when sample fields are set then conversion is successful": {
			arg: &claircore.VulnerabilityReport{
				Hash: claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"sample vuln ID": {
						ID:                 "sample vuln ID",
						Name:               "sample vuln name",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           claircore.Critical.String(),
						NormalizedSeverity: claircore.Critical,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id"},
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"sample pkg id": {"sample vuln ID"},
				},
			},
			want: &v4.VulnerabilityReport{
				// Converter doesn't set HashId to empty.
				HashId: "",
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"sample vuln ID": {
						Id:                 "sample vuln ID",
						Name:               "sample vuln name",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "Critical",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id",
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"sample pkg id": {
						Values: []string{"sample vuln ID"},
					},
				},
				Contents: &v4.Contents{},
			},
			wantErr: "",
		},
		"when there are duplicate vulnerabilities then they are filtered": {
			arg: &claircore.VulnerabilityReport{
				Hash: claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"0": {
						ID:                 "0",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: claircore.Low,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id"},
						FixedInVersion:     "sample vuln fixed in",
						Updater:            "rhel8",
					},
					"1": {
						ID:                 "1",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: claircore.Low,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id 2"},
						FixedInVersion:     "sample vuln fixed in",
						Updater:            "rhel8",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"sample pkg id": {"0", "1"},
				},
			},
			want: &v4.VulnerabilityReport{
				// Converter doesn't set HashId to empty.
				HashId: "",
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						Id:                 "0",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id",
						FixedInVersion:     "sample vuln fixed in",
					},
					"1": {
						Id:                 "1",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id 2",
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"sample pkg id": {
						Values: []string{"0"},
					},
				},
				Contents: &v4.Contents{},
			},
			wantErr: "",
		},
		"when there are similar vulnerabilities with different severities and updaters then they are not filtered": {
			arg: &claircore.VulnerabilityReport{
				Hash: claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"0": {
						ID:                 "0",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: claircore.Low,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id"},
						FixedInVersion:     "sample vuln fixed in",
						Updater:            "rhel8",
					},
					"1": {
						ID:                 "1",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: claircore.Medium,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id 2"},
						FixedInVersion:     "sample vuln fixed in",
						Updater:            "rhel8-2",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"sample pkg id": {"0", "1"},
				},
			},
			want: &v4.VulnerabilityReport{
				// Converter doesn't set HashId to empty.
				HashId: "",
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						Id:                 "0",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id",
						FixedInVersion:     "sample vuln fixed in",
					},
					"1": {
						Id:                 "1",
						Name:               "CVE-2019-12900",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "CVSS:3.0/AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id 2",
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"sample pkg id": {
						Values: []string{"1", "0"}, // "1" has a higher severity
					},
				},
				Contents: &v4.Contents{},
			},
			wantErr: "",
		},
	}
	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ToProtoV4VulnerabilityReport(ctx, tt.arg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			protoassert.Equal(t, tt.want, got)
		})
	}
}

func Test_ToProtoV4VulnerabilityReport_FilterNodeJS(t *testing.T) {
	t.Setenv(features.ScannerV4PartialNodeJSSupport.EnvVar(), "true")

	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	require.NoError(t, err)

	tests := map[string]struct {
		arg     *claircore.VulnerabilityReport
		want    *v4.VulnerabilityReport
		wantErr string
	}{
		"filter Node.js packages without vulns": {
			arg: &claircore.VulnerabilityReport{
				Hash: claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						ID:                 "1",
						Name:               "sample vuln name",
						Description:        "sample vuln description",
						Issued:             now,
						Links:              "sample vuln links",
						Severity:           claircore.Critical.String(),
						NormalizedSeverity: claircore.Critical,
						Package:            &claircore.Package{ID: "sample vuln package id"},
						Dist:               &claircore.Distribution{ID: "sample vuln distribution id"},
						Repo:               &claircore.Repository{ID: "sample vuln repository id"},
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				Packages: map[string]*claircore.Package{
					"0": {
						ID:      "0",
						Name:    "nodejs0",
						Version: "0",
					},
					"1": {
						ID:      "1",
						Name:    "nodejs1",
						Version: "1",
					},
					"2": {
						ID:      "2",
						Name:    "nodejs2",
						Version: "2",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"0": {
						{
							PackageDB: "nodejs:/app/nodejs0",
						},
					},
					"1": {
						{
							PackageDB: "nodejs:/app/nodejs1",
						},
					},
					"2": {
						{
							PackageDB: "nodejs:/app/nodejs2",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1"},
					"2": {},
				},
			},
			want: &v4.VulnerabilityReport{
				// Converter doesn't set HashId to empty.
				HashId: "",
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"1": {
						Id:                 "1",
						Name:               "sample vuln name",
						Description:        "sample vuln description",
						Issued:             protoNow,
						Link:               "sample vuln links",
						Severity:           "Critical",
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
						PackageId:          "sample vuln package id",
						DistributionId:     "sample vuln distribution id",
						RepositoryId:       "sample vuln repository id",
						FixedInVersion:     "sample vuln fixed in",
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"1": {
						Values: []string{"1"},
					},
				},
				Contents: &v4.Contents{
					Packages: []*v4.Package{
						{
							Id:      "1",
							Name:    "nodejs1",
							Version: "1",
							NormalizedVersion: &v4.NormalizedVersion{
								V: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							},
							Cpe: emptyCPE,
						},
					},
					Environments: map[string]*v4.Environment_List{
						"1": {
							Environments: []*v4.Environment{
								{
									PackageDb: "nodejs:/app/nodejs1",
								},
							},
						},
					},
				},
			},
			wantErr: "",
		},
	}
	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ToProtoV4VulnerabilityReport(ctx, tt.arg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			protoassert.Equal(t, tt.want, got)
		})
	}
}

func TestToProtoV4VulnerabilityReport_FilterRHCCLayers(t *testing.T) {
	testutils.MustUpdateFeature(t, features.ScannerV4RedHatLayers, true)

	layerA := claircore.MustParseDigest("sha256:" + strings.Repeat("a", 64))
	layerB := claircore.MustParseDigest("sha256:" + strings.Repeat("b", 64))

	tests := map[string]struct {
		arg     *claircore.VulnerabilityReport
		want    *v4.VulnerabilityReport
		wantErr string
	}{
		"filter non-RPM packages in Red Hat layers": {
			arg: &claircore.VulnerabilityReport{
				Hash: claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"0": {
						ID:      "0",
						Name:    "0",
						Updater: "rhel-vex",
					},
					"1": {
						ID:      "1",
						Name:    "1",
						Updater: "rhel-vex",
					},
					"2": {
						ID:      "2",
						Name:    "2",
						Updater: "something else",
					},
					"3": {
						ID:      "3",
						Name:    "3",
						Updater: "something different",
					},
				},
				Packages: map[string]*claircore.Package{
					"0": {
						ID:      "0",
						Name:    "my go binary",
						Version: "0",
					},
					"1": {
						ID:      "1",
						Name:    "my java jar",
						Version: "1",
					},
					"2": {
						ID:      "2",
						Name:    "my python egg",
						Version: "2",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"0": {
						ID:   "0",
						Name: "Red Hat Container Catalog",
						URI:  `https://catalog.redhat.com/software/containers/explore`,
					},
					"1": {
						ID:   "1",
						Name: "something else",
						URI:  "somethingelse.com",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"0": {
						{
							RepositoryIDs: []string{"0", "1"},
							IntroducedIn:  layerA,
						},
					},
					"1": {
						{
							RepositoryIDs: []string{"1"},
							IntroducedIn:  layerB,
						},
					},
					"2": {
						{
							RepositoryIDs: []string{"0"},
							IntroducedIn:  layerA,
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"0": {"2", "0", "3", "1"},
					"1": {"1", "2"},
					"2": {"2", "3"},
				},
			},
			want: &v4.VulnerabilityReport{
				// Converter doesn't set HashId to empty.
				HashId: "",
				Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						Id:   "0",
						Name: "0",
					},
					"1": {
						Id:   "1",
						Name: "1",
					},
					"2": {
						Id:   "2",
						Name: "2",
					},
					"3": {
						Id:   "3",
						Name: "3",
					},
				},
				PackageVulnerabilities: map[string]*v4.StringList{
					"0": {
						Values: []string{"0", "1"},
					},
					"1": {
						Values: []string{"1", "2"},
					},
				},
				Contents: &v4.Contents{
					Packages: []*v4.Package{
						{
							Id:      "0",
							Name:    "my go binary",
							Version: "0",
							NormalizedVersion: &v4.NormalizedVersion{
								V: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							},
							Cpe: emptyCPE,
						},
						{
							Id:      "1",
							Name:    "my java jar",
							Version: "1",
							NormalizedVersion: &v4.NormalizedVersion{
								V: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							},
							Cpe: emptyCPE,
						},
						{
							Id:      "2",
							Name:    "my python egg",
							Version: "2",
							NormalizedVersion: &v4.NormalizedVersion{
								V: []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
							},
							Cpe: emptyCPE,
						},
					},
					Repositories: []*v4.Repository{
						{
							Id:   "0",
							Name: "Red Hat Container Catalog",
							Uri:  `https://catalog.redhat.com/software/containers/explore`,
							Cpe:  emptyCPE,
						},
						{
							Id:   "1",
							Name: "something else",
							Uri:  "somethingelse.com",
							Cpe:  emptyCPE,
						},
					},
					Environments: map[string]*v4.Environment_List{
						"0": {
							Environments: []*v4.Environment{
								{
									RepositoryIds: []string{"0", "1"},
									IntroducedIn:  layerA.String(),
								},
							},
						},
						"1": {
							Environments: []*v4.Environment{
								{
									RepositoryIds: []string{"1"},
									IntroducedIn:  layerB.String(),
								},
							},
						},
						"2": {
							Environments: []*v4.Environment{
								{
									RepositoryIds: []string{"0"},
									IntroducedIn:  layerA.String(),
								},
							},
						},
					},
				},
			},
			wantErr: "",
		},
	}
	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ToProtoV4VulnerabilityReport(ctx, tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)

			// The assert library cannot compare elements in slices like the ones below
			// while ignoring order. So, sort each slice.
			for _, pkgVulns := range got.GetPackageVulnerabilities() {
				slices.Sort(pkgVulns.GetValues())
			}
			slices.SortFunc(got.GetContents().GetPackages(), func(a, b *v4.Package) int {
				return strings.Compare(a.GetId(), b.GetId())
			})
			slices.SortFunc(got.GetContents().GetRepositories(), func(a, b *v4.Repository) int {
				return strings.Compare(a.GetId(), b.GetId())
			})

			protoassert.Equal(t, tt.want, got)
		})
	}
}

func Test_ToClairCoreIndexReport(t *testing.T) {
	tests := map[string]struct {
		arg     *v4.Contents
		want    *claircore.IndexReport
		wantErr string
	}{
		"when content is nil then error": {
			wantErr: "empty content",
		},
		"when content is default then report is default": {
			arg:  &v4.Contents{},
			want: &claircore.IndexReport{},
		},
		"when content package has source with source then error": {
			arg: &v4.Contents{
				Packages: []*v4.Package{
					{
						Id:  "sample package",
						Cpe: "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
						Source: &v4.Package{
							Id:     "source",
							Cpe:    "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
							Source: &v4.Package{Id: "deep source"},
						},
					},
				},
			},
			wantErr: "source specifies source",
		},
		"when content package has invalid CPE then error": {
			arg: &v4.Contents{
				Packages: []*v4.Package{
					{
						Id:  "sample package",
						Cpe: "something that is not a cpe",
					},
				},
			},
			wantErr: `internal error: package "sample package": "something that is not a cpe"`,
		},
		"when distribution contains invalid cpe then error": {
			arg: &v4.Contents{
				Distributions: []*v4.Distribution{
					{
						Cpe: "something that is not a cpe",
					},
				},
			},
			wantErr: `internal error: distribution "": "something that is not a cpe"`,
		},
		"when repository contains invalid cpe then error": {
			arg: &v4.Contents{
				Repositories: []*v4.Repository{
					{
						Cpe: "something that is not a cpe",
					},
				},
			},
			wantErr: `internal error: repository "": "something that is not a cpe"`,
		},

		"when all fields are valid then return success": {
			arg: &v4.Contents{
				Packages: []*v4.Package{
					{
						Id:      "sample pkg id",
						Name:    "sample pkg name",
						Version: "sample pkg version",
						NormalizedVersion: &v4.NormalizedVersion{
							Kind: "test",
							V:    []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
						},
						Kind: "sample pkg kind",
						Source: &v4.Package{
							Id:   "sample source id",
							Name: "sample source name",
							Cpe:  "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
						},
						PackageDb:      "sample pkg db",
						RepositoryHint: "sample pkg repo hint",
						Module:         "sample pkg module",
						Arch:           "sample pkg arch",
						Cpe:            "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
					},
				},
				Distributions: []*v4.Distribution{
					{
						Id:              "sample dist id",
						Did:             "sample dist did",
						Name:            "sample dist name",
						Version:         "sample dist version",
						VersionCodeName: "sample dist version codename",
						VersionId:       "sample dist version id",
						Arch:            "sample dist arch",
						Cpe:             "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
						PrettyName:      "sample dist pretty",
					},
				},
				Repositories: []*v4.Repository{
					{
						Id:   "sample id",
						Name: "sample name",
						Key:  "sample key",
						Uri:  "sample URI",
						Cpe:  "cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*",
					},
				},
				Environments: map[string]*v4.Environment_List{
					"sample env": {
						Environments: []*v4.Environment{
							{
								PackageDb:      "sample env pkg db",
								IntroducedIn:   "sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e",
								DistributionId: "sample env distribution id",
								RepositoryIds:  []string{"sample env repository id"},
							},
						},
					},
				},
			},
			want: &claircore.IndexReport{
				Hash:  claircore.Digest{},
				State: "",
				Packages: map[string]*claircore.Package{
					"sample pkg id": {
						ID:      "sample pkg id",
						Name:    "sample pkg name",
						Version: "sample pkg version",
						Kind:    "sample pkg kind",
						Source: &claircore.Package{
							ID:   "sample source id",
							Name: "sample source name",
							CPE:  cpe.MustUnbind("cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*"),
						},
						PackageDB:      "sample pkg db",
						RepositoryHint: "sample pkg repo hint",
						NormalizedVersion: claircore.Version{
							Kind: "test",
							V:    [...]int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
						},
						Module: "sample pkg module",
						Arch:   "sample pkg arch",
						CPE:    cpe.MustUnbind("cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*"),
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"sample dist id": {
						ID:              "sample dist id",
						DID:             "sample dist did",
						Name:            "sample dist name",
						Version:         "sample dist version",
						VersionCodeName: "sample dist version codename",
						VersionID:       "sample dist version id",
						Arch:            "sample dist arch",
						CPE:             cpe.MustUnbind("cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*"),
						PrettyName:      "sample dist pretty",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"sample id": {
						ID:   "sample id",
						Name: "sample name",
						Key:  "sample key",
						URI:  "sample URI",
						CPE:  cpe.MustUnbind("cpe:2.3:a:redhat:scanner:4:*:el9:*:*:*:*:*"),
					},
				},
				Environments: map[string][]*claircore.Environment{
					"sample env": {{
						PackageDB:      "sample env pkg db",
						IntroducedIn:   claircore.MustParseDigest("sha256:9124cd5256c6d674f6b11a4d01fea8148259be1f66ca2cf9dfbaafc83c31874e"),
						DistributionID: "sample env distribution id",
						RepositoryIDs:  []string{"sample env repository id"},
					}},
				},
				Success: false,
				Err:     "",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ToClairCoreIndexReport(tt.arg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_toProtoV4Package(t *testing.T) {
	tests := []struct {
		name    string
		arg     *claircore.Package
		want    *v4.Package
		wantErr string
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when sample values then no error",
			arg: &claircore.Package{
				ID:             "sample id",
				Name:           "sample name",
				Version:        "sample version",
				Kind:           "sample kind",
				Source:         nil,
				PackageDB:      "sample package db",
				Filepath:       "sample file path",
				RepositoryHint: "sample hint",
				NormalizedVersion: claircore.Version{
					Kind: "test",
					V:    [...]int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
				},
				Module: "sample module",
				Arch:   "sample arch",
				CPE:    cpe.WFN{},
			},
			want: &v4.Package{
				Id:      "sample id",
				Name:    "sample name",
				Version: "sample version",
				NormalizedVersion: &v4.NormalizedVersion{
					Kind: "test",
					V:    []int32{1, 2, 3, 4, 5, 6, 7, 8, 9, 0},
				},
				Kind:           "sample kind",
				Source:         nil,
				PackageDb:      "sample package db",
				RepositoryHint: "sample hint",
				Module:         "sample module",
				Arch:           "sample arch",
				Cpe:            emptyCPE,
			},
		},
		{
			name: "when source with source then error",
			arg: &claircore.Package{
				Name: "Sample name",
				Source: &claircore.Package{
					Name: "sample source",
					Source: &claircore.Package{
						Name: "should be removed",
					},
				},
			},
			wantErr: "source specifies source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toProtoV4Package(tt.arg)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, tt.want)
			} else {
				protoassert.Equal(t, tt.want, got)
				assert.NoError(t, err)
			}
		})
	}
	// Test source with another source.
	t.Run("when source has source then error", func(t *testing.T) {
		arg := &claircore.Package{
			Name: "Package",
			Source: &claircore.Package{
				Name: "source",
				Source: &claircore.Package{
					Name: "another source",
				},
			},
		}
		got, err := toProtoV4Package(arg)
		assert.Nil(t, got)
		assert.ErrorContains(t, err, "source specifies source")
	})
}

func Test_toProtoV4Distribution(t *testing.T) {
	tests := []struct {
		name    string
		arg     *claircore.Distribution
		want    *v4.Distribution
		wantErr bool
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when default then no errors",
			arg:  &claircore.Distribution{},
			want: &v4.Distribution{Cpe: emptyCPE},
		},
		{
			name: "when default then no errors",
			arg: &claircore.Distribution{
				ID:              "sample id",
				DID:             "sample did",
				Name:            "sample name",
				Version:         "sample version",
				VersionCodeName: "sample version codename",
				VersionID:       "sample version id",
				Arch:            "sample arch",
				CPE:             cpe.MustUnbind("cpe:/a:redhat:openshift:4.12::el8"),
				PrettyName:      "sample pretty name",
			},
			want: &v4.Distribution{
				Id:              "sample id",
				Did:             "sample did",
				Name:            "sample name",
				Version:         "sample version",
				VersionCodeName: "sample version codename",
				VersionId:       "sample version id",
				Arch:            "sample arch",
				Cpe:             "cpe:2.3:a:redhat:openshift:4.12:*:el8:*:*:*:*:*",
				PrettyName:      "sample pretty name",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toProtoV4Distribution(tt.arg)
			protoassert.Equal(t, tt.want, got)
		})
	}
}

func Test_toProtoV4Repository(t *testing.T) {
	tests := []struct {
		name string
		arg  *claircore.Repository
		want *v4.Repository
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when sample then no error",
			arg: &claircore.Repository{
				ID:   "sample id",
				Name: "sample name",
				Key:  "sample key",
				URI:  "sample URI",
				CPE:  cpe.MustUnbind("cpe:/a:redhat:openshift:4.12::el8"),
			},
			want: &v4.Repository{
				Id:   "sample id",
				Name: "sample name",
				Key:  "sample key",
				Uri:  "sample URI",
				Cpe:  "cpe:2.3:a:redhat:openshift:4.12:*:el8:*:*:*:*:*",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toProtoV4Repository(tt.arg)
			protoassert.Equal(t, tt.want, got)
		})
	}
}

func Test_toProtoV4Environment(t *testing.T) {
	tests := []struct {
		name string
		arg  *claircore.Environment
		want *v4.Environment
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when default then no errors",
			arg:  &claircore.Environment{},
			want: &v4.Environment{},
		},
		{
			name: "when sample values then no errors",
			arg: &claircore.Environment{
				PackageDB:      "sample package db",
				IntroducedIn:   claircore.Digest{},
				DistributionID: "sample distribution",
				RepositoryIDs:  nil,
			},
			want: &v4.Environment{
				PackageDb:      "sample package db",
				IntroducedIn:   "",
				DistributionId: "sample distribution",
				RepositoryIds:  nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toProtoV4Environment(tt.arg)
			protoassert.Equal(t, tt.want, got)
			if tt.want != nil && tt.want.RepositoryIds != nil {
				assert.NotEqual(t, &tt.want.RepositoryIds, &got.RepositoryIds)
			}
		})
	}
}

func Test_toProtoV4Contents(t *testing.T) {
	type args struct {
		pkgs  map[string]*claircore.Package
		dists map[string]*claircore.Distribution
		repos map[string]*claircore.Repository
		envs  map[string][]*claircore.Environment
	}
	tests := map[string]struct {
		args    args
		want    *v4.Contents
		wantErr string
	}{
		"when one empty environment": {
			args: args{
				pkgs:  map[string]*claircore.Package{"sample pkg": {}},
				dists: map[string]*claircore.Distribution{"sample dist": {}},
				repos: map[string]*claircore.Repository{"sample repo": {}},
				envs: map[string][]*claircore.Environment{
					"sample env": {{}},
				},
			},
			want: &v4.Contents{
				Packages: []*v4.Package{{
					Cpe: emptyCPE,
					NormalizedVersion: &v4.NormalizedVersion{
						Kind: "",
						V:    make([]int32, 10),
					},
				}},
				Distributions: []*v4.Distribution{{
					Cpe: emptyCPE,
				}},
				Repositories: []*v4.Repository{{
					Cpe: emptyCPE,
				}},
				Environments: map[string]*v4.Environment_List{
					"sample env": {
						Environments: []*v4.Environment{{}},
					},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := toProtoV4Contents(tt.args.pkgs, tt.args.dists, tt.args.repos, tt.args.envs, nil)
			if tt.wantErr != "" {
				assert.Nil(t, got)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				protoassert.Equal(t, tt.want, got)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_toProtoV4VulnerabilitiesMapWithEPSS(t *testing.T) {
	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	require.NoError(t, err)

	tests := map[string]struct {
		ccVulnerabilities map[string]*claircore.Vulnerability
		nvdVulns          map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem
		epssItems         map[string]map[string]*epss.EPSSItem
		enableRedHatCVEs  bool
		want              map[string]*v4.VulnerabilityReport_Vulnerability
	}{
		"should use EPSS and NVD CVSS scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "Name contains CVE-1234-567",
					Issued:  now,
					Updater: "unknown updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-9999-999": {
						ID: "CVE-9999-999",
					},
					"CVE-1234-567": {
						ID: "CVE-1234-567",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
									},
								},
							},
						},
					},
				},
			},
			epssItems: map[string]map[string]*epss.EPSSItem{
				"foo": {
					"CVE-1234-567": &epss.EPSSItem{
						ModelVersion: "v2023.03.01",
						CVE:          "CVE-1234-567",
						Date:         "2025-01-15T00:00:00+0000",
						EPSS:         0.00215,
						Percentile:   0.59338,
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					EpssMetrics: &v4.VulnerabilityReport_Vulnerability_EPSS{
						ModelVersion: "v2023.03.01",
						Date:         "2025-01-15T00:00:00+0000",
						Probability:  0.00215,
						Percentile:   0.59338,
					},
					Id:     "foo",
					Issued: protoNow,
					Name:   "CVE-1234-567",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
						},
					},
				},
			},
		},

		// TODO(ROX-27729): get the highest EPSS score for an RHSA across all CVEs associated with that RHSA
		"should use EPSS score of highest CVE on multi-CVE RHSAs": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "Name contains CVE-1234-567",
					Issued:  now,
					Updater: "rhel-vex",
					Links:   "https://access.redhat.com/errata/RHSA-2021:1234",
				},
				"bar": {
					ID:      "bar",
					Name:    "Name contains CVE-7654-321",
					Issued:  now,
					Updater: "rhel-vex",
					Links:   "https://access.redhat.com/errata/RHSA-2021:1234",
				},
			},
			nvdVulns: nil,
			epssItems: map[string]map[string]*epss.EPSSItem{
				"foo": {
					"CVE-1234-567": {
						ModelVersion: "v2023.03.01",
						CVE:          "CVE-1234-567",
						Date:         "2025-01-15T00:00:00+0000",
						EPSS:         0.00215,
						Percentile:   0.59338,
					},
					"CVE-7654-321": {
						ModelVersion: "v2023.03.01",
						CVE:          "CVE-7654-321",
						Date:         "2025-01-15T00:00:00+0000",
						EPSS:         0.04215,
						Percentile:   0.69338,
					}},
				"bar": {
					"CVE-1234-567": {
						ModelVersion: "v2023.03.01",
						CVE:          "CVE-1234-567",
						Date:         "2025-01-15T00:00:00+0000",
						EPSS:         0.00215,
						Percentile:   0.59338,
					},
					"CVE-7654-321": {
						ModelVersion: "v2023.03.01",
						CVE:          "CVE-7654-321",
						Date:         "2025-01-15T00:00:00+0000",
						EPSS:         0.04215,
						Percentile:   0.69338,
					}},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					EpssMetrics: &v4.VulnerabilityReport_Vulnerability_EPSS{
						ModelVersion: "v2023.03.01",
						Date:         "2025-01-15T00:00:00+0000",
						Probability:  0.04215,
						Percentile:   0.69338,
					},
					Id:          "foo",
					Issued:      protoNow,
					Name:        "RHSA-2021:1234",
					Cvss:        nil,
					CvssMetrics: nil,
					Link:        "https://access.redhat.com/errata/RHSA-2021:1234",
				},
				"bar": {
					EpssMetrics: &v4.VulnerabilityReport_Vulnerability_EPSS{
						ModelVersion: "v2023.03.01",
						Date:         "2025-01-15T00:00:00+0000",
						Probability:  0.04215,
						Percentile:   0.69338,
					},
					Id:          "bar",
					Issued:      protoNow,
					Name:        "RHSA-2021:1234",
					Cvss:        nil,
					CvssMetrics: nil,
					Link:        "https://access.redhat.com/errata/RHSA-2021:1234",
				},
			},
		},
		"EPSS Missing": { // it could be the feature is turned off or EPSS data is missing for some reason
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"bar": {
					ID:      "bar",
					Name:    "Name contains CVE-5678-1234",
					Issued:  now,
					Updater: "unknown updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"bar": {
					"CVE-5678-1234": {
						ID: "CVE-5678-1234",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:N",
									},
								},
							},
						},
					},
				},
			},
			epssItems: nil,
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"bar": {
					Id:     "bar",
					Name:   "CVE-5678-1234",
					Issued: protoNow,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:N",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-5678-1234",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								Vector: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:N",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-5678-1234",
						},
					},
					// No EpssMetrics because epssItems has no entry for CVE-5678-1234
				},
			},
		},
	}

	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			enableRedHatCVEs := "false"
			t.Setenv(features.ScannerV4RedHatCVEs.EnvVar(), enableRedHatCVEs)
			enableEPSS := "true"
			t.Setenv(features.EPSSScore.EnvVar(), enableEPSS)
			got, err := toProtoV4VulnerabilitiesMap(ctx, tt.ccVulnerabilities, tt.nvdVulns, tt.epssItems, nil)
			assert.NoError(t, err)
			protoassert.MapEqual(t, tt.want, got)
		})
	}
}

func Test_toProtoV4VulnerabilitiesMap(t *testing.T) {
	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	require.NoError(t, err)
	published2021 := "2021-12-10T10:15:09.143"
	proto2021 := protoconv.ConvertTimeString(published2021)
	tests := map[string]struct {
		ccVulnerabilities map[string]*claircore.Vulnerability
		nvdVulns          map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem
		advisories        map[string]csaf.Advisory
		enableRedHatCVEs  bool
		want              map[string]*v4.VulnerabilityReport_Vulnerability
	}{
		"when nil then nil": {},
		"when vulnerabilities then convert": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:             now,
					Severity:           claircore.Critical.String(),
					NormalizedSeverity: claircore.Critical,
				},
				"bar": {
					Issued:             now,
					Severity:           claircore.High.String(),
					NormalizedSeverity: claircore.High,
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:             protoNow,
					Severity:           "Critical",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
				},
				"bar": {
					Issued:             protoNow,
					Severity:           "High",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				},
			},
		},
		"when vuln with plain fixedIn then convert": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:         now,
					FixedInVersion: "1.2.3",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:         protoNow,
					FixedInVersion: "1.2.3",
				},
			},
		},
		"when vuln urlencoded fixedIn then use fixed value in fixedIn": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:         now,
					FixedInVersion: "fixed=4.5.6",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:         protoNow,
					FixedInVersion: "4.5.6",
				},
			},
		},
		"when severity and unknown distribution then populate the proto": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:   now,
					Severity: "sample severity",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
				},
			},
		},
		"when severity with CVSSv3 and RHEL then find CVSS score": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Name:     "CVE-1234-567",
					Issued:   now,
					Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					Updater:  "rhel-vex",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Name:     "CVE-1234-567",
					Issued:   protoNow,
					Severity: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.8,
							Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 9.8,
								Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/security/cve/CVE-1234-567",
						},
					},
				},
			},
		},
		"when severity with CVSSv2 and RHEL then find CVSS score": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Name:     "CVE-2013-12342",
					Issued:   now,
					Severity: "AV:N/AC:L/Au:N/C:P/I:P/A:P",
					Updater:  "rhel-vex",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Name:     "CVE-2013-12342",
					Issued:   protoNow,
					Severity: "AV:N/AC:L/Au:N/C:P/I:P/A:P",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
							BaseScore: 7.5,
							Vector:    "AV:N/AC:L/Au:N/C:P/I:P/A:P",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/security/cve/CVE-2013-12342",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
								BaseScore: 7.5,
								Vector:    "AV:N/AC:L/Au:N/C:P/I:P/A:P",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/security/cve/CVE-2013-12342",
						},
					},
				},
			},
		},
		"when severity with CVSSv2 is invalid skip CVSS": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:   now,
					Severity: "invalid cvss2 vector",
					Updater:  "rhel-vex",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "invalid cvss2 vector",
				},
			},
		},
		"when severity with CVSSv3 is invalid skip CVSS": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:   now,
					Severity: "invalid cvss3 vector",
					Updater:  "rhel-vex",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "invalid cvss3 vector",
				},
			},
		},
		"when OSV and severity with CVSSv3 then return": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Name:     "CVE-2024-1234",
					Issued:   now,
					Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Updater:  "osv/sample-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Name:     "CVE-2024-1234",
					Issued:   protoNow,
					Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 10.0,
							Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
						Url:    "https://osv.dev/vulnerability/CVE-2024-1234",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 10.0,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
							Url:    "https://osv.dev/vulnerability/CVE-2024-1234",
						},
					},
				},
			},
		},
		"when OSV and severity is not CVSS skip CVSS": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:             now,
					NormalizedSeverity: claircore.Low,
					Severity:           "LOW",
					Updater:            "osv/sample-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:             protoNow,
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
					Severity:           "LOW",
				},
			},
		},
		"when unknown updater then return NVD scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "Name contains CVE-1234-567",
					Issued:  now,
					Updater: "unknown updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-9999-999": {
						ID: "CVE-9999-999",
					},
					"CVE-1234-567": {
						ID: "CVE-1234-567",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
									},
								},
							},
						},
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:     "foo",
					Issued: protoNow,
					Name:   "CVE-1234-567",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
						},
					},
				},
			},
		},
		"when OSV missing severity then return NVD scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "Name contains CVE-1234-567",
					Issued:  now,
					Updater: "osv/sample-updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-9999-999": {
						ID: "CVE-9999-999",
					},
					"CVE-1234-567": {
						ID: "CVE-1234-567",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
									},
								},
							},
						},
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:     "foo",
					Issued: protoNow,
					Name:   "CVE-1234-567",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
						},
					},
				},
			},
		},
		"when using NVD and vuln name is not CVE then return first NVD scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "CVE-1234-567",
					Issued:  now,
					Updater: "unknown updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-1234-567": {
						ID: "CVE-1234-567",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
									},
								},
							},
						},
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:     "foo",
					Name:   "CVE-1234-567",
					Issued: protoNow,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-1234-567",
						},
					},
				},
			},
		},
		"when issued time is empty, use NVD published time": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "CVE-2021-44228",
					Updater: "unknown updater",
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-2021-44228": {
						ID:        "CVE-2021-44228",
						Published: published2021,
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:     "foo",
					Name:   "CVE-2021-44228",
					Issued: proto2021,
				},
			},
		},
		"when manual vulnerability with NVD link, do not get NVD data again": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:       "foo",
					Name:     "CVE-2021-44228",
					Links:    "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
					Updater:  manual.UpdaterName,
					Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Issued:   now,
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-2021-44228": {
						ID: "CVE-2021-44228",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
									},
								},
							},
						},
					},
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:       "foo",
					Name:     "CVE-2021-44228",
					Link:     "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
					Issued:   protoNow,
					Severity: "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.3,
							Vector:    "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
						Url:    "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 9.3,
								Vector:    "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
						},
					},
				},
			},
		},
		"when Red Hat CVEs disabled return RHSA": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:                 "foo",
					Name:               "CVE-2021-44228",
					Links:              "https://access.redhat.com/security/cve/CVE-2021-44228 https://access.redhat.com/errata/RHSA-2021:5132",
					Updater:            "rhel-vex",
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					NormalizedSeverity: claircore.Critical,
					Issued:             now.Add(-1 * time.Second),
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-2021-44228": {
						ID: "CVE-2021-44228",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
										BaseScore:    10.0,
									},
								},
							},
						},
					},
				},
			},
			advisories: map[string]csaf.Advisory{
				"foo": {
					Name:        "RHSA-2021:5132",
					Description: "RHSA description",
					Severity:    "Moderate",
					CVSSv3: csaf.CVSS{
						Score:  9.1,
						Vector: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N",
					},
					CVSSv2: csaf.CVSS{
						Score:  9.4,
						Vector: "AV:N/AC:L/Au:N/C:C/I:C/A:N",
					},
					ReleaseDate: now,
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:                 "foo",
					Name:               "RHSA-2021:5132",
					Description:        "RHSA description",
					Link:               "https://access.redhat.com/security/cve/CVE-2021-44228 https://access.redhat.com/errata/RHSA-2021:5132",
					Issued:             protoNow,
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.1,
							Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N",
						},
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
							BaseScore: 9.4,
							Vector:    "AV:N/AC:L/Au:N/C:C/I:C/A:N",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/errata/RHSA-2021:5132",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 9.1,
								Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N",
							},
							V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
								BaseScore: 9.4,
								Vector:    "AV:N/AC:L/Au:N/C:C/I:C/A:N",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/errata/RHSA-2021:5132",
						},
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 10.0,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
						},
					},
				},
			},
		},
		"when Red Hat CVEs enabled return CVE": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:                 "foo",
					Name:               "CVE-2021-44228",
					Links:              "https://access.redhat.com/security/cve/CVE-2021-44228 https://access.redhat.com/errata/RHSA-2021:5132",
					Updater:            "rhel-vex",
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					NormalizedSeverity: claircore.Critical,
					Issued:             now,
				},
			},
			nvdVulns: map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem{
				"foo": {
					"CVE-2021-44228": {
						ID: "CVE-2021-44228",
						Metrics: &nvdschema.CVEAPIJSON20CVEItemMetrics{
							CvssMetricV31: []*nvdschema.CVEAPIJSON20CVSSV31{
								{
									CvssData: &nvdschema.CVSSV31{
										Version:      "3.1",
										VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
										BaseScore:    10.0,
									},
								},
							},
						},
					},
				},
			},
			enableRedHatCVEs: true,
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:                 "foo",
					Name:               "CVE-2021-44228",
					Advisory:           "RHSA-2021:5132",
					Link:               "https://access.redhat.com/security/cve/CVE-2021-44228 https://access.redhat.com/errata/RHSA-2021:5132",
					Issued:             protoNow,
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.8,
							Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/security/cve/CVE-2021-44228",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 9.8,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/security/cve/CVE-2021-44228",
						},
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 10.0,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
							Url:    "https://nvd.nist.gov/vuln/detail/CVE-2021-44228",
						},
					},
				},
			},
		},
		// Note: Scanner V4 should ultimately return a single RHSA, but this is handled via manipulation to
		// PackageVulnerabilities through dedupeAdvisories.
		"when multiple Red Hat CVEs relate to same RHSA return each RHSA": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:                 "foo",
					Name:               "CVE-2024-24789",
					Links:              "https://access.redhat.com/security/cve/CVE-2024-24789 https://access.redhat.com/errata/RHSA-2024:10775",
					Updater:            "rhel-vex",
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
					NormalizedSeverity: claircore.Medium,
					Issued:             now.Add(-2 * time.Hour),
				},
				"bar": {
					ID:                 "bar",
					Name:               "CVE-2024-24790",
					Links:              "https://access.redhat.com/security/cve/CVE-2024-24790 https://access.redhat.com/errata/RHSA-2024:10775",
					Updater:            "rhel-vex",
					Severity:           "CVSS:3.1/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N",
					NormalizedSeverity: claircore.Medium,
					Issued:             now.Add(-1 * time.Hour),
				},
			},
			advisories: map[string]csaf.Advisory{
				"foo": {
					Name:        "RHSA-2024:10775",
					Description: "RHSA description",
					Severity:    "Moderate",
					CVSSv3: csaf.CVSS{
						Score:  7.5,
						Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
					},
					ReleaseDate: now,
				},
				"bar": {
					Name:        "RHSA-2024:10775",
					Description: "RHSA description",
					Severity:    "Moderate",
					CVSSv3: csaf.CVSS{
						Score:  7.5,
						Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
					},
					ReleaseDate: now,
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Id:                 "foo",
					Name:               "RHSA-2024:10775",
					Description:        "RHSA description",
					Link:               "https://access.redhat.com/security/cve/CVE-2024-24789 https://access.redhat.com/errata/RHSA-2024:10775",
					Issued:             protoNow,
					Severity:           "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 7.5,
							Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/errata/RHSA-2024:10775",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 7.5,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/errata/RHSA-2024:10775",
						},
					},
				},
				"bar": {
					Id:                 "bar",
					Name:               "RHSA-2024:10775",
					Description:        "RHSA description",
					Link:               "https://access.redhat.com/security/cve/CVE-2024-24790 https://access.redhat.com/errata/RHSA-2024:10775",
					Issued:             protoNow,
					Severity:           "CVSS:3.1/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 7.5,
							Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
						},
						Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
						Url:    "https://access.redhat.com/errata/RHSA-2024:10775",
					},
					CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
						{
							V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
								BaseScore: 7.5,
								Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N",
							},
							Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
							Url:    "https://access.redhat.com/errata/RHSA-2024:10775",
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			enableRedHatCVEs := "false"
			if tt.enableRedHatCVEs {
				enableRedHatCVEs = "true"
			}
			t.Setenv(features.ScannerV4RedHatCVEs.EnvVar(), enableRedHatCVEs)
			// EPSS scores are intentionally not covered here
			got, err := toProtoV4VulnerabilitiesMap(ctx, tt.ccVulnerabilities, tt.nvdVulns, nil, tt.advisories)
			assert.NoError(t, err)
			protoassert.MapEqual(t, tt.want, got)
		})
	}
}

func Test_convertToNormalizedSeverity(t *testing.T) {
	ctx := context.Background()
	// Check all severities can be mapped.
	for i := 0; i <= int(claircore.Critical); i++ {
		ccS := claircore.Severity(i)
		switch ccS {
		case claircore.Unknown:
			assert.Equal(t, v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED, toProtoV4VulnerabilitySeverity(ctx, ccS))
		case claircore.Negligible, claircore.Low, claircore.Medium, claircore.High, claircore.Critical:
			assert.NotEqual(t, v4.VulnerabilityReport_Vulnerability_SEVERITY_UNSPECIFIED, toProtoV4VulnerabilitySeverity(ctx, ccS))
		default:
			t.Errorf("Unexpected Severity value %d found", i)
		}
	}
	// Test nothing was added without us knowing.
	assert.Equal(t, int(claircore.Critical), 5)
}

func Test_vulnerabilityName(t *testing.T) {
	testcases := map[string]struct {
		name             string
		links            string
		expected         string
		updater          string
		enableRedHatCVEs bool
	}{
		"Alpine": {
			name:     "CVE-2018-16840",
			expected: "CVE-2018-16840",
		},
		"Amazon Linux": {
			name:     "ALAS-2022-1654",
			expected: "ALAS-2022-1654",
		},
		"Debian": {
			name:     "DSA-4591-1 cyrus-sasl2",
			expected: "DSA-4591-1",
		},
		"RHEL/RHSA": {
			name:     "RHSA-2023:0173: libxml2 security update (Moderate)",
			expected: "RHSA-2023:0173",
		},
		"RHEL/RHBA": {
			name:     "RHBA-2019:1992: cloud-init bug fix and enhancement update (Moderate)",
			expected: "RHBA-2019:1992",
		},
		"RHEL/RHEA": {
			name:     "RHEA-2019:3845: microcode_ctl bug fix and enhancement update (Important)",
			expected: "RHEA-2019:3845",
		},
		"Ubuntu": {
			name:     "CVE-2022-45061 on Ubuntu 22.04 LTS (jammy) - medium.",
			expected: "CVE-2022-45061",
		},
		"GHSA": {
			name:     "GHSA-5wvp-7f3h-6wmm PyArrow: Arbitrary code execution when loading a malicious data file",
			expected: "GHSA-5wvp-7f3h-6wmm",
		},
		"Unknown": {
			name:     "cool CVE right here",
			expected: "cool CVE right here",
		},
		"CVE over GHSA": {
			name:     "GHSA-5wvp-7f3h-6wmm PyArrow: Arbitrary code execution when loading a malicious data file",
			links:    "https://nvd.nist.gov/vuln/detail/CVE-2023-47248",
			expected: "CVE-2023-47248",
		},
		"when rhel updater and Red Hat CVEs disabled then RHSA over CVE": {
			name:     "CVE-2023-25762",
			links:    "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
			expected: "RHSA-2023:1866",
			updater:  "rhel-vex",
		},
		"when rhel updater and Red Hat CVE enabled then CVE over RHSA": {
			name:             "CVE-2023-25762",
			links:            "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
			expected:         "CVE-2023-25762",
			updater:          "rhel-vex",
			enableRedHatCVEs: true,
		},
		"when not rhel updater then CVE over RHEL": {
			links:    "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
			expected: "CVE-2023-25761",
			updater:  "not-rhel",
		},
		"ALAS over CVE": {
			links:    "https://alas.aws.amazon.com/AL2023/ALAS-2023-356.html https://alas.aws.amazon.com/cve/html/CVE-2023-39189.html",
			expected: "ALAS-2023-356",
			updater:  "aws-foobar-",
		},
	}
	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			v := &claircore.Vulnerability{
				Name:    testcase.name,
				Links:   testcase.links,
				Updater: testcase.updater,
			}
			enableRedHatCVEs := "false"
			if testcase.enableRedHatCVEs {
				enableRedHatCVEs = "true"
			}
			t.Setenv(features.ScannerV4RedHatCVEs.EnvVar(), enableRedHatCVEs)
			assert.Equal(t, testcase.expected, vulnerabilityName(v))
		})
	}
	t.Run("when updater is osv/go then prefer GO over RHSA", func(t *testing.T) {
		v := &claircore.Vulnerability{
			Updater:     "osv/go",
			Name:        "GO-2021-0072",
			Description: "Uncontrolled resource allocation in github.com/docker/distribution",
			Links:       "https://github.com/distribution/distribution/pull/2340 https://github.com/distribution/distribution/commit/91c507a39abfce14b5c8541cf284330e22208c0f https://access.redhat.com/errata/RHSA-2017:2603 http://lists.opensuse.org/opensuse-security-announce/2020-09/msg00047.html",
		}
		assert.Equal(t, "GO-2021-0072", vulnerabilityName(v))
	})
}

func Test_advisory(t *testing.T) {
	testutils.MustUpdateFeature(t, features.ScannerV4RedHatCVEs, true)
	testcases := map[string]struct {
		vuln     *claircore.Vulnerability
		expected string
	}{
		"non-VEX": {
			vuln: &claircore.Vulnerability{
				Links:   "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
				Updater: "not-vex",
			},
			expected: "",
		},
		"no RHSA": {
			vuln: &claircore.Vulnerability{
				Links:   "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/security/cve/CVE-2023-25762",
				Updater: "rhel-vex",
			},
			expected: "",
		},
		"RHSA": {
			vuln: &claircore.Vulnerability{
				Links:   "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
				Updater: "rhel-vex",
			},
			expected: "RHSA-2023:1866",
		},
	}
	for name, testcase := range testcases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testcase.expected, advisory(testcase.vuln))
		})
	}
}

func Test_versionID(t *testing.T) {
	tests := map[string]struct {
		d         *claircore.Distribution
		versionID string
	}{
		"when version ID is empty and distribution is Alpine then use version": {
			d:         &claircore.Distribution{Version: "sample alpine version ID", DID: "alpine"},
			versionID: "sample alpine version ID",
		},
		"when version is not empty and distribution is Alpine then use version ID": {
			d:         &claircore.Distribution{Version: "sample alpine version", DID: "alpine"},
			versionID: "sample alpine version",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			versionID := VersionID(tt.d)
			assert.Equal(t, tt.versionID, versionID)
		})
	}
}

func Test_sortByNVDCVSS(t *testing.T) {
	type args struct {
		ids             []string
		vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no NVD CVSS scores",
			args: args{
				ids: []string{"0", "1"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{},
					},
					"1": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{},
					},
				},
			},
			want: []string{"0", "1"},
		},
		{
			name: "one NVD CVSS score",
			args: args{
				ids: []string{"0", "1"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{},
					},
					"1": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{
								Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
								V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
									BaseScore: 1.0,
								},
							},
						},
					},
				},
			},
			want: []string{"1", "0"},
		},
		{
			name: "multiple NVD CVSS scores",
			args: args{
				ids: []string{"0", "1", "2"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"0": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{
								Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_OSV,
								V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
									BaseScore: 10.0,
								},
							},
						},
					},
					"1": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{
								Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_RED_HAT,
								V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
									BaseScore: 10.0,
								},
							},
							{
								Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
								V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
									BaseScore: 1.0,
								},
							},
						},
					},
					"2": {
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{
								Source: v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD,
								V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
									BaseScore: 3.0,
								},
							},
						},
					},
				},
			},
			want: []string{"2", "1", "0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortByNVDCVSS(tt.args.ids, tt.args.vulnerabilities)
			assert.Equalf(t, tt.want, tt.args.ids, "sortByNVDCVSS(%v, %v)", tt.args.ids, tt.args.vulnerabilities)
		})
	}
}

func Test_getMaxBaseScore(t *testing.T) {
	type args struct {
		cvssMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS
	}
	tests := []struct {
		name string
		args args
		want float32
	}{
		{
			name: "single V3 score",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 9.8},
					},
				},
			},
			want: 9.8,
		},
		{
			name: "single V2 score",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 7.5},
					},
				},
			},
			want: 7.5,
		},
		{
			name: "multiple scores/highest V3",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 6.0},
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 9.0},
					},
				},
			},
			want: 9.0,
		},
		{
			name: "multiple scores/highest V2",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 8.5},
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 7.0},
					},
				},
			},
			want: 7.0,
		},
		{
			name: "nil CVSS metrics",
			args: args{
				cvssMetrics: nil,
			},
			want: 0.0,
		},
		{
			name: "empty CVSS metrics",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{},
			},
			want: 0.0,
		},
		{
			name: "nil V3 and V2 in CVSS metrics",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V3: nil,
						V2: nil,
					},
				},
			},
			want: 0.0,
		},
		{
			name: "multiple first is selected",
			args: args{
				cvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
					{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 4.0},
					},
					{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 8.0},
					},
				},
			},
			want: 4.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, baseScore(tt.args.cvssMetrics), "baseScore(%v)", tt.args.cvssMetrics)
		})
	}
}

func Test_sortBySeverity(t *testing.T) {
	type args struct {
		ids             []string
		vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "sort by severity descending",
			args: args{
				ids: []string{"111", "222", "333"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW},
					"222": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL},
					"333": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT},
				},
			},
			want: []string{"222", "333", "111"},
		},
		{
			name: "sort by severity with ties resolved by CVSS score",
			args: args{
				ids: []string{"111", "222", "333"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 8.0}},
						},
					},
					"222": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{BaseScore: 9.5}},
						},
					},
					"333": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
					},
				},
			},
			want: []string{"222", "111", "333"},
		},
		{
			name: "nil vulnerabilities/nil first",
			args: args{
				ids: []string{"111", "222", "333"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": nil,
					"222": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE},
					"333": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW},
				},
			},
			want: []string{"222", "333", "111"},
		},
		{
			name: "nil vulnerabilities/nil after",
			args: args{
				ids: []string{"111", "222", "333"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE},
					"222": {NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW},
					"333": nil,
				},
			},
			want: []string{"111", "222", "333"},
		},
		{
			name: "nil vulnerabilities/all keep order",
			args: args{
				ids: []string{"222", "111"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": nil,
					"222": nil,
				},
			},
			want: []string{"222", "111"},
		},
		{
			name: "Empty input",
			args: args{
				ids:             []string{},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{},
			},
			want: []string{},
		},
		{
			name: "All vulnerabilities with same severity and CVSS",
			args: args{
				ids: []string{"111", "222", "333"},
				vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
					"111": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 4.0}},
						},
					},
					"222": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 4.0}},
						},
					},
					"3": {
						NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW,
						CvssMetrics: []*v4.VulnerabilityReport_Vulnerability_CVSS{
							{V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{BaseScore: 4.0}},
						},
					},
				},
			},
			want: []string{"111", "222", "333"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortBySeverity(tt.args.ids, tt.args.vulnerabilities)
			assert.Equalf(t, tt.want, tt.args.ids, "sortBySeverity(%v, %v)", tt.args.ids, tt.args.vulnerabilities)
		})
	}
}

func Test_dedupeAdvisories(t *testing.T) {
	now := timestamppb.Now()

	testcases := []struct {
		name     string
		vulnIDs  []string
		vulns    map[string]*v4.VulnerabilityReport_Vulnerability
		expected []string
	}{
		{
			name:    "basic",
			vulnIDs: []string{"0", "1", "2", "3", "4", "5"},
			vulns: map[string]*v4.VulnerabilityReport_Vulnerability{
				"0": {
					Id:                 "0",
					Name:               "RHSA-2021:0735",
					Description:        "Red Hat Security Advisory: nodejs:10 security update",
					Issued:             now,
					Link:               "https://access.redhat.com/errata/RHSA-2021:0735",
					Severity:           "Important",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				},
				"1": {
					Id:                 "1",
					Name:               "RHSA-2021:0735",
					Description:        "Red Hat Security Advisory: nodejs:10 security update",
					Issued:             now,
					Link:               "https://access.redhat.com/errata/RHSA-2021:0735",
					Severity:           "Important",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT,
				},
				"2": {
					Id:                 "2",
					Name:               "RHSA-2021:0548",
					Description:        "Red Hat Security Advisory: nodejs:10 security update",
					Issued:             now,
					Link:               "https://access.redhat.com/errata/RHSA-2021:0548",
					Severity:           "Moderate",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				},
				"3": {
					Id:          "3",
					Name:        "CVE-2025-12342",
					Description: "very vulnerable",
				},
				"4": {
					Id:                 "4",
					Name:               "RHSA-2021:0548",
					Description:        "Red Hat Security Advisory: nodejs:10 security update",
					Issued:             now,
					Link:               "https://access.redhat.com/errata/RHSA-2021:0548",
					Severity:           "Moderate",
					NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				},
				"5": {
					Id:          "5",
					Name:        "CVE-2025-12342",
					Description: "very vulnerable",
				},
			},
			expected: []string{"0", "2", "3", "5"},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupeAdvisories(tt.vulnIDs, tt.vulns)
			assert.ElementsMatch(t, tt.expected, got)
		})
	}
}
