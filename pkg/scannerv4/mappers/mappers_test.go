package mappers

import (
	"context"
	"net/url"
	"testing"
	"time"

	nvdschema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/quay/claircore"
	"github.com/quay/claircore/pkg/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

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
		"when invalid time in vulnerability map then error": {
			arg: &claircore.VulnerabilityReport{
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"sample CVE": {
						ID: "sample CVE",
						// Timestamp lower than epoch is invalid.
						Issued: time.Time{}.Add(-time.Hour),
					},
				},
			},
			wantErr: "internal error",
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
						Values: []string{"0", "1"},
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
	t.Setenv(env.ScannerV4PartialNodeJSSupport.EnvVar(), "true")

	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	assert.NoError(t, err)

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
		tt := tt
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

func Test_toProtoV4VulnerabilitiesMap(t *testing.T) {
	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	assert.NoError(t, err)
	tests := map[string]struct {
		ccVulnerabilities map[string]*claircore.Vulnerability
		nvdVulns          map[string]map[string]*nvdschema.CVEAPIJSON20CVEItem
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
		"when severity with CVSSv3 and RHEL then find encoded CVSS scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued: now,
					Severity: url.Values{
						"severity":     []string{"sample severity"},
						"cvss3_vector": []string{"CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
						"cvss3_score":  []string{"9.9"},
					}.Encode(),
					Updater: "RHEL8-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.9,
							Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						},
					},
				},
			},
		},
		"when severity with CVSSv2 and RHEL then find encoded CVSS scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued: now,
					Severity: url.Values{
						"severity":     []string{"sample severity"},
						"cvss2_vector": []string{"AV:N/AC:L/Au:N/C:P/I:P/A:P"},
						"cvss2_score":  []string{"1.1"},
					}.Encode(),
					Updater: "RHEL8-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
							BaseScore: 1.1,
							Vector:    "AV:N/AC:L/Au:N/C:P/I:P/A:P",
						},
					},
				},
			},
		},
		"when severity with CVSSv2 and CVSSv3 and RHEL then find encoded CVSS scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued: now,
					Severity: url.Values{
						"severity":     []string{"sample severity"},
						"cvss2_vector": []string{"AV:N/AC:L/Au:N/C:P/I:P/A:P"},
						"cvss2_score":  []string{"1.1"},
						"cvss3_vector": []string{"CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
						"cvss3_score":  []string{"9.9"},
					}.Encode(),
					Updater: "RHEL8-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							BaseScore: 9.9,
							Vector:    "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
						},
						V2: &v4.VulnerabilityReport_Vulnerability_CVSS_V2{
							BaseScore: 1.1,
							Vector:    "AV:N/AC:L/Au:N/C:P/I:P/A:P",
						},
					},
				},
			},
		},
		"when severity with CVSSv2 is invalid skip CVSS": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued: now,
					Severity: url.Values{
						"severity":     []string{"sample severity"},
						"cvss2_vector": []string{"invalid cvss2 vector"},
					}.Encode(),
					Updater: "RHEL8-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
				},
			},
		},
		"when severity with CVSSv3 is invalid skip CVSS": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued: now,
					Severity: url.Values{
						"severity":     []string{"sample severity"},
						"cvss3_vector": []string{"invalid cvss3 vector"},
					}.Encode(),
					Updater: "RHEL8-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "sample severity",
				},
			},
		},
		"when OSV and severity with CVSSv3 then return": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					Issued:   now,
					Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Updater:  "osv/sample-updater",
				},
			},
			want: map[string]*v4.VulnerabilityReport_Vulnerability{
				"foo": {
					Issued:   protoNow,
					Severity: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
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
					},
				},
			},
		},
		"when using NVD and vuln name is not CVE then return first NVD scores": {
			ccVulnerabilities: map[string]*claircore.Vulnerability{
				"foo": {
					ID:      "foo",
					Name:    "NOT-A-CVE",
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
					Name:   "NOT-A-CVE",
					Issued: protoNow,
					Cvss: &v4.VulnerabilityReport_Vulnerability_CVSS{
						V3: &v4.VulnerabilityReport_Vulnerability_CVSS_V3{
							Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H",
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := toProtoV4VulnerabilitiesMap(ctx, tt.ccVulnerabilities, tt.nvdVulns)
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
		name     string
		links    string
		expected string
		updater  string
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
		"when rhel updater then RHEL over CVE": {
			links:    "https://access.redhat.com/security/cve/CVE-2023-25761 https://access.redhat.com/errata/RHSA-2023:1866 https://access.redhat.com/security/cve/CVE-2023-25762",
			expected: "RHSA-2023:1866",
			updater:  "RHEL9-FOOBAR",
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
