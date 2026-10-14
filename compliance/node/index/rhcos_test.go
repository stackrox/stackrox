package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

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

func TestRHCOSImageName(t *testing.T) {
	const labels = `{"name":"openshift/ose-rhel-coreos-10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`
	for name, tc := range map[string]struct {
		path, contents, want string
		wantError            bool
	}{
		"RHCOS location":    {path: "usr/share/buildinfo/labels.json", contents: labels, want: "openshift/ose-rhel-coreos-10"},
		"original location": {path: "root/buildinfo/labels.json", contents: labels, want: "openshift/ose-rhel-coreos-10"},
		"quoted labels":     {path: "usr/share/buildinfo/labels.json", contents: `{"name":"\"openshift/ose-rhel-coreos-9\"","architecture":"\"amd64\"","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`, want: "openshift/ose-rhel-coreos-9"},
		"absent":            {},
		"incomplete":        {path: "usr/share/buildinfo/labels.json", contents: `{"name":"openshift/ose-rhel-coreos-10"}`},
		"malformed":         {path: "usr/share/buildinfo/labels.json", contents: `{`, wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if tc.path != "" {
				writeNodeFile(t, root, tc.path, tc.contents)
			}
			got, err := rhcosImageName(t.Context(), root)
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRHCOSReleaseImageName(t *testing.T) {
	for name, tc := range map[string]struct {
		versionID, version, extra, major string
	}{
		"RHEL 8 older RHCOS": {"4.12", "412.86.202603240000-0", "RHEL_VERSION=8.6\nPLATFORM_ID=platform:el8\n", "8"},
		"RHEL 9 platform":    {"4.16", "416.94.202603240000-0", "PLATFORM_ID=platform:el9\n", "9"},
		"RHEL 9 layered":     {"9.6", "9.6.20260324-0", "", "9"},
		"RHEL 10 layered":    {"10.2", "10.2.20260324-0", "", "10"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeNodeFile(t, root, "etc/os-release", fmt.Sprintf("VARIANT_ID=coreos\nVERSION=%s\nOPENSHIFT_VERSION=4.21\nVERSION_ID=%s\n%s", tc.version, tc.versionID, tc.extra))
			rel, err := osRelease(t.Context(), root)
			require.NoError(t, err)
			assert.Equal(t, "openshift/ose-rhel-coreos-"+tc.major, rel.imageName)
			assert.Equal(t, tc.version, rel.version)
			assert.Contains(t, rel.repoCPE.String(), "el"+tc.major)
		})
	}
}

func (s *nodeIndexerSuite) TestIndexerRHCOSLabels() {
	server := s.createTestServer(false)
	// A read or parse error falls back to the os-release-derived name rather than
	// failing the index, so RPM coverage is preserved.
	for name, tc := range map[string]struct {
		labels, want string
	}{
		"metadata name":       {labels: `{"name":"openshift/ose-rhel-coreos-10","architecture":"amd64","org.opencontainers.image.created":"2026-07-15T17:00:53Z"}`, want: "openshift/ose-rhel-coreos-10"},
		"absent metadata":     {want: "openshift/ose-rhel-coreos-9"},
		"incomplete metadata": {labels: `{}`, want: "openshift/ose-rhel-coreos-9"},
		"malformed metadata":  {labels: `{`, want: "openshift/ose-rhel-coreos-9"},
	} {
		s.Run(name, func() {
			root := s.T().TempDir()
			osRelease, err := os.ReadFile("testdata-rhcos/etc/os-release")
			s.Require().NoError(err)
			writeNodeFile(s.T(), root, "etc/os-release", string(osRelease))
			if tc.labels != "" {
				writeNodeFile(s.T(), root, "usr/share/buildinfo/labels.json", tc.labels)
			}
			cfg := DefaultNodeIndexerConfig()
			cfg.HostPath = "testdata"
			cfg.OSReleasePath = root
			cfg.Repo2CPEMappingURL = server.URL
			cfg.Client = server.Client()
			report, err := NewNodeIndexer(cfg).IndexNode(context.Background())
			s.Require().NoError(err)
			pkgs := report.GetContents().GetPackages()
			s.Len(pkgs, 110)
			for _, id := range []string{rhcos.PackageID, rhcos.SourceID} {
				s.Equal(tc.want, pkgs[id].GetName())
				s.Equal("9.6.20260324-0", pkgs[id].GetVersion())
			}
			s.Equal("rhcos", pkgs[rhcos.LegacyPackageID].GetName())
			s.Equal("rhcos", pkgs[rhcos.LegacySourceID].GetName())
		})
	}
}
