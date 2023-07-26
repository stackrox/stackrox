package services

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/pkg/cpe"
	"github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
)

func Test_convertToIndexReport(t *testing.T) {
	sampleDigest := claircore.MustParseDigest("sha256:aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f")
	tests := []struct {
		name string
		arg  *claircore.IndexReport
		want *v4.IndexReport
	}{
		{
			name: "when nil then nil",
		},
		{
			name: "when default values",
			arg:  &claircore.IndexReport{},
			want: &v4.IndexReport{},
		},
		{
			name: "when happy sample values",
			arg: &claircore.IndexReport{
				Hash:  sampleDigest,
				State: "sample_state",
				Packages: map[string]*claircore.Package{
					"sample_package_key": {
						Name: "sample_package",
					}},
				Distributions: map[string]*claircore.Distribution{
					"sample_distribution_key": {
						Name: "sample_distribution",
					}},
				Repositories: map[string]*claircore.Repository{
					"sample_repository_key": {
						Name: "sample_repository",
					}},
				Environments: map[string][]*claircore.Environment{"sample_env_key": {
					{
						PackageDB:    "sample_db",
						IntroducedIn: sampleDigest,
					},
				}},
				Success: true,
				Err:     "",
			},
			want: &v4.IndexReport{
				State: "sample_state",
				Packages: []*v4.Package{
					{
						Name: "sample_package",
						NormalizedVersion: &v4.NormalizedVersion{
							Kind: "",
							V:    make([]int32, 10),
						},
						Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
				},
				Distributions: []*v4.Distribution{
					{
						Name: "sample_distribution",
						Cpe:  "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
				},
				Repositories: []*v4.Repository{
					{
						Name: "sample_repository",
						Cpe:  "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
				},
				Environments: map[string]*v4.Environment_List{"sample_env_key": {Environments: []*v4.Environment{
					{
						PackageDb:    "sample_db",
						IntroducedIn: sampleDigest.String(),
					},
				}}},
				Success: true,
				Err:     "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, convertToIndexReport(tt.arg))
		})
	}
}

func Test_convertToPackage(t *testing.T) {
	tests := []struct {
		name string
		arg  *claircore.Package
		want *v4.Package
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
				Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToPackage(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
	// Test source with another source (prevents recursion).
	t.Run("when source has source no convertion", func(t *testing.T) {
		arg := &claircore.Package{
			Name: "Package",
			Source: &claircore.Package{
				Name: "source",
				Source: &claircore.Package{
					Name: "another source",
				},
			},
		}
		got := convertToPackage(arg)
		assert.Nil(t, got.GetSource().GetSource())
	})
}

func Test_convertToDistribution(t *testing.T) {
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
			want: &v4.Distribution{Cpe: "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"},
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
			got := convertToDistribution(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_convertToRepository(t *testing.T) {
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
			got := convertToRepository(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_convertToEnvironment(t *testing.T) {
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
			got := convertToEnvironment(tt.arg)
			assert.Equal(t, tt.want, got)
			if tt.want != nil && tt.want.RepositoryIds != nil {
				assert.NotEqual(t, &tt.want.RepositoryIds, &got.RepositoryIds)
			}
		})
	}
}
