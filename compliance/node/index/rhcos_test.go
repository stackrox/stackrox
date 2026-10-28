package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stackrox/rox/pkg/scannerv4/rhcos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeNodeFile(t *testing.T, root, path, contents string) {
	t.Helper()
	path = filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0644))
}

func TestRHCOSImageMetadata(t *testing.T) {
	const labels = `{"name":"openshift/ose-rhel-coreos-10","cpe":"cpe:/a:redhat:openshift:5.0::el10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`
	for name, tc := range map[string]struct {
		path, contents, want string
		wantCPE              string
		wantPresent          bool
		wantError            bool
	}{
		"RHCOS location": {
			path: "usr/share/buildinfo/labels.json", contents: labels,
			want: "openshift/ose-rhel-coreos-10", wantCPE: "cpe:/a:redhat:openshift:5.0::el10", wantPresent: true,
		},
		"original location": {
			path: "root/buildinfo/labels.json", contents: labels,
			want: "openshift/ose-rhel-coreos-10", wantCPE: "cpe:/a:redhat:openshift:5.0::el10", wantPresent: true,
		},
		"quoted labels": {
			path:     "usr/share/buildinfo/labels.json",
			contents: `{"name":"\"openshift/ose-rhel-coreos-9\"","cpe":"\"cpe:/a:redhat:openshift:4.22::el9\"","architecture":"\"amd64\"","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			want:     "openshift/ose-rhel-coreos-9", wantCPE: "cpe:/a:redhat:openshift:4.22::el9", wantPresent: true,
		},
		"absent": {},
		"incomplete": {
			path: "usr/share/buildinfo/labels.json", contents: `{"name":"openshift/ose-rhel-coreos-10"}`,
			wantPresent: true, wantError: true,
		},
		"malformed": {path: "usr/share/buildinfo/labels.json", contents: `{`, wantPresent: true, wantError: true},
		"CPE without package metadata": {
			path: "usr/share/buildinfo/labels.json", contents: `{"cpe":"cpe:/a:redhat:openshift:5.0::el10"}`,
			wantPresent: true, wantError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if tc.path != "" {
				writeNodeFile(t, root, tc.path, tc.contents)
			}
			got, present, err := readRHCOSImageMetadata(t.Context(), root)
			assert.Equal(t, tc.wantPresent, present)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.name)
			if tc.wantCPE == "" {
				assert.Nil(t, got.repoCPE)
			} else {
				require.NotNil(t, got.repoCPE)
				wantCPE, err := cpe.Unbind(tc.wantCPE)
				require.NoError(t, err)
				assert.Equal(t, wantCPE, *got.repoCPE)
			}
		})
	}
}

func TestRHCOSPackageVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		contents       string
		wantVersion    string
		wantNormalized claircore.Version
		wantError      string
	}{
		"IMAGE_VERSION takes precedence": {
			contents:       "IMAGE_VERSION=9.8.20261006-0\nVERSION=\"9.4.20260813-0 (Plow)\"\n",
			wantVersion:    "9.8.20261006-0",
			wantNormalized: claircore.Version{Kind: "rhctag", V: [10]int32{9, 8}},
		},
		"IMAGE_VERSION does not require VERSION": {
			contents: "IMAGE_VERSION=9.8.20261006-0\n", wantVersion: "9.8.20261006-0",
		},
		"valid IMAGE_VERSION ignores invalid display VERSION": {
			contents: "IMAGE_VERSION=9.8.20261006-0\nVERSION=invalid\n", wantVersion: "9.8.20261006-0",
		},
		"invalid IMAGE_VERSION does not silently use VERSION": {
			contents: "IMAGE_VERSION=invalid\nVERSION=9.8.20261006-0\n", wantError: "failed to parse RHCOS version",
		},
		"missing IMAGE_VERSION uses plain VERSION": {
			contents: "VERSION=9.8.20261006-0\n", wantVersion: "9.8.20261006-0",
		},
		"missing IMAGE_VERSION removes known display suffix": {
			contents: "VERSION=\"9.4.20260813-0 (Plow)\"\n", wantVersion: "9.4.20260813-0",
		},
		"empty IMAGE_VERSION uses VERSION": {
			contents: "IMAGE_VERSION=\nVERSION=\"9.8.20261006-0 (Plow)\"\n", wantVersion: "9.8.20261006-0",
		},
		"other parenthesized suffix is preserved": {
			contents: "VERSION=\"9.8.20261006-0 (Other)\"\n", wantVersion: "9.8.20261006-0 (Other)",
		},
		"suffix without separating space is preserved": {
			contents: "VERSION=\"9.8.20261006-0(Plow)\"\n", wantVersion: "9.8.20261006-0(Plow)",
		},
		"extra separating space is preserved": {
			contents: "VERSION=\"9.8.20261006-0  (Plow)\"\n", wantVersion: "9.8.20261006-0  (Plow)",
		},
		"text after the suffix is preserved": {
			contents: "VERSION=\"9.8.20261006-0 (Plow) extra\"\n", wantVersion: "9.8.20261006-0 (Plow) extra",
		},
		"text before the build is preserved in the parse error": {
			contents: "VERSION=\"extra 9.8.20261006-0 (Plow)\"\n", wantError: `failed to parse RHCOS version "extra 9.8.20261006-0 (Plow)"`,
		},
		"short build date is preserved": {
			contents: "VERSION=\"9.8.202610-0 (Plow)\"\n", wantVersion: "9.8.202610-0 (Plow)",
		},
		"release qualifier is preserved": {
			contents: "VERSION=\"9.8.20261006-0.debug (Plow)\"\n", wantVersion: "9.8.20261006-0.debug (Plow)",
		},
		"compact CoreOS version is preserved": {
			contents: "VERSION=\"418.94.202608111829-0 (Plow)\"\n", wantVersion: "418.94.202608111829-0 (Plow)",
		},
		"neither version field is available": {
			wantError: "IMAGE_VERSION and VERSION not found",
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeNodeFile(t, root, "etc/os-release", "VARIANT_ID=coreos\n"+tc.contents)
			rel, err := osRelease(t.Context(), root)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantVersion, rel.version)
			if tc.wantNormalized.Kind != "" {
				assert.Equal(t, tc.wantNormalized, rel.normVersion)
			}
		})
	}
}

func TestRHCOSReleaseImageName(t *testing.T) {
	for name, tc := range map[string]struct {
		versionID, version, major string
	}{
		"RHEL 9 layered":    {"9.6", "9.6.20260324-0", "9"},
		"RHEL 10 layered":   {"10.2", "10.2.20260324-0", "10"},
		"future RHEL major": {"11.0", "11.0.20260324-0", "11"},
		// Pre-layered RHCOS reports the OCP version in VERSION_ID, not the RHEL
		// version (see the upstream layering design's /etc/os-release before/after
		// examples). This is a known, accepted limitation of this fallback for
		// pre-layered RHCOS, not the intended behavior: the derived major is the
		// OCP major, not the RHEL major. labels.json is the correct source for
		// that case when present.
		"pre-layered RHCOS VERSION_ID is the OCP version": {"4.19", "419.94.202412101440-0", "4"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeNodeFile(t, root, "etc/os-release", fmt.Sprintf("VARIANT_ID=coreos\nVERSION=%s\nOPENSHIFT_VERSION=4.21\nVERSION_ID=%s\n", tc.version, tc.versionID))
			rel, err := osRelease(t.Context(), root)
			require.NoError(t, err)
			metadata, err := rhcosIdentityFromOSRelease(rel.osRel)
			require.NoError(t, err)
			assert.Equal(t, "openshift/ose-rhel-coreos-"+tc.major, metadata.name)
			assert.Equal(t, tc.version, rel.version)
			assert.Contains(t, metadata.repoCPE.String(), "el"+tc.major)
		})
	}
}

func (s *nodeIndexerSuite) TestIndexerRHCOSLabels() {
	server := s.createTestServer(false)
	for name, tc := range map[string]struct {
		labels, osRelease, want, wantCPE string
		wantVersion                      string
		omitImage                        bool
		labelsDirectory                  bool
	}{
		"full labels bundle used": {
			labels: `{"name":"openshift/ose-rhel-coreos-10","cpe":"cpe:/a:redhat:openshift:5.0::el10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			want:   "openshift/ose-rhel-coreos-10", wantCPE: "cpe:/a:redhat:openshift:5.0::el10",
		},
		"absent labels.json falls back to os-release pair": {want: "openshift/ose-rhel-coreos-9"},
		"IMAGE_VERSION populates both package pairs": {
			osRelease: "VARIANT_ID=coreos\nIMAGE_VERSION=9.8.20261006-0\nVERSION=\"9.8.20261006-0 (Plow)\"\nVERSION_ID=9.8\nOPENSHIFT_VERSION=4.22\n",
			want:      "openshift/ose-rhel-coreos-9", wantCPE: "cpe:/a:redhat:openshift:4.22::el9", wantVersion: "9.8.20261006-0",
		},
		"labels and IMAGE_VERSION work without VERSION": {
			labels:    `{"name":"openshift/ose-rhel-coreos-9","cpe":"cpe:/a:redhat:openshift:4.22::el9","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			osRelease: "VARIANT_ID=coreos\nIMAGE_VERSION=9.8.20261006-0\n",
			want:      "openshift/ose-rhel-coreos-9", wantCPE: "cpe:/a:redhat:openshift:4.22::el9", wantVersion: "9.8.20261006-0",
		},
		"VERSION fallback removes Plow for both package pairs": {
			osRelease: "VARIANT_ID=coreos\nVERSION=\"9.4.20260813-0 (Plow)\"\nVERSION_ID=9.4\nOPENSHIFT_VERSION=4.22\n",
			want:      "openshift/ose-rhel-coreos-9", wantCPE: "cpe:/a:redhat:openshift:4.22::el9", wantVersion: "9.4.20260813-0",
		},
		"invalid IMAGE_VERSION preserves RPM-only success": {
			osRelease: "VARIANT_ID=coreos\nIMAGE_VERSION=invalid\nVERSION=9.6.20260324-0\nVERSION_ID=9.6\nOPENSHIFT_VERSION=4.21\n",
			omitImage: true,
		},
		"missing both version fields preserves RPM-only success": {
			osRelease: "VARIANT_ID=coreos\nVERSION_ID=9.6\nOPENSHIFT_VERSION=4.21\n",
			omitImage: true,
		},
		"valid labels identity does not require os-release fallback fields": {
			labels:    `{"name":"openshift/ose-rhel-coreos-10","cpe":"cpe:/a:redhat:openshift:5.0::el10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			osRelease: "VARIANT_ID=coreos\nVERSION=9.6.20260324-0\n",
			want:      "openshift/ose-rhel-coreos-10", wantCPE: "cpe:/a:redhat:openshift:5.0::el10",
		},
		"absent labels.json requires complete os-release identity": {
			osRelease: "VARIANT_ID=coreos\nVERSION=9.6.20260324-0\n",
			omitImage: true,
		},
		"labels missing cpe uses os-release pair": {
			labels: `{"name":"openshift/ose-rhel-coreos-10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			want:   "openshift/ose-rhel-coreos-9",
		},
		"labels missing name uses os-release pair": {
			labels: `{"cpe":"cpe:/a:redhat:openshift:5.0::el10"}`,
			want:   "openshift/ose-rhel-coreos-9",
		},
		"malformed labels.json uses os-release pair":  {labels: `{`, want: "openshift/ose-rhel-coreos-9"},
		"unreadable labels.json uses os-release pair": {labelsDirectory: true, want: "openshift/ose-rhel-coreos-9"},
		"invalid labels cpe discards labels name": {
			labels: `{"name":"openshift/ose-rhel-coreos-10","cpe":"invalid","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`,
			want:   "openshift/ose-rhel-coreos-9",
		},
		"invalid labels and incomplete fallback omit image packages": {
			labels: `{`, osRelease: "VARIANT_ID=coreos\nVERSION=9.6.20260324-0\n", omitImage: true,
		},
	} {
		s.Run(name, func() {
			root := s.T().TempDir()
			osRelease := tc.osRelease
			if osRelease == "" {
				osReleaseBytes, err := os.ReadFile("testdata-rhcos/etc/os-release")
				s.Require().NoError(err)
				osRelease = string(osReleaseBytes)
			}
			writeNodeFile(s.T(), root, "etc/os-release", osRelease)
			if tc.labels != "" {
				writeNodeFile(s.T(), root, "usr/share/buildinfo/labels.json", tc.labels)
			}
			if tc.labelsDirectory {
				s.Require().NoError(os.MkdirAll(filepath.Join(root, "usr/share/buildinfo/labels.json"), 0755))
			}
			cfg := DefaultNodeIndexerConfig()
			cfg.HostPath = "testdata"
			cfg.OSReleasePath = root
			cfg.Repo2CPEMappingURL = server.URL
			cfg.Client = server.Client()
			report, err := NewNodeIndexer(cfg).IndexNode(context.Background())
			s.Require().NoError(err)
			s.Require().NotNil(report)
			s.True(report.GetSuccess())
			s.Equal("IndexFinished", report.GetState())
			s.Empty(report.GetErr())
			if tc.omitImage {
				s.Len(report.GetContents().GetPackages(), 106, "successfully scanned RPM packages should remain available")
				s.Len(report.GetContents().GetRepositories(), 2)
				for _, id := range []string{rhcos.PackageID, rhcos.SourceID, rhcos.LegacyPackageID, rhcos.LegacySourceID} {
					s.NotContains(report.GetContents().GetPackages(), id)
				}
				return
			}
			pkgs := report.GetContents().GetPackages()
			s.Len(pkgs, 110)
			for _, id := range []string{rhcos.PackageID, rhcos.SourceID} {
				s.Equal(tc.want, pkgs[id].GetName())
			}
			wantVersion := tc.wantVersion
			if wantVersion == "" {
				wantVersion = "9.6.20260324-0"
			}
			for _, id := range []string{rhcos.PackageID, rhcos.SourceID, rhcos.LegacyPackageID, rhcos.LegacySourceID} {
				s.Equal(wantVersion, pkgs[id].GetVersion())
			}
			s.Equal("rhcos", pkgs[rhcos.LegacyPackageID].GetName())
			s.Equal("rhcos", pkgs[rhcos.LegacySourceID].GetName())
			wantCPE := tc.wantCPE
			if wantCPE == "" {
				wantCPE = "cpe:/a:redhat:openshift:4.21::el9"
			}
			repos := report.GetContents().GetRepositories()
			s.Require().Contains(repos, "rhcos-repo")
			parsedCPE, err := cpe.Unbind(wantCPE)
			s.Require().NoError(err)
			s.Equal(parsedCPE.String(), repos["rhcos-repo"].GetName())
			for _, id := range []string{rhcos.PackageID, rhcos.LegacyPackageID} {
				environments := report.GetContents().GetEnvironments()[id].GetEnvironments()
				s.Require().Len(environments, 1)
				s.Equal([]string{"rhcos-repo"}, environments[0].GetRepositoryIds())
			}
		})
	}
}
