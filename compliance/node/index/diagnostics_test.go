package index

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/quay/claircore"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestIndexDiagnosticsPreservesReport(t *testing.T) {
	const mapping = `{"data":{"rhel-9-for-x86_64-baseos-eus-rpms__9_DOT_4":{"cpes":["cpe:/o:redhat:rhel_eus:9.4::baseos"]}}}`
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(mapping))
	}))
	defer server.Close()
	cfg := createConfig("testdata", server.Client(), server.URL)
	cfg.PackageDBFilter = rhcosPackageDBs
	want, err := NewNodeIndexer(cfg).IndexNode(t.Context())
	require.NoError(t, err)
	for name, enabled := range map[string]bool{
		"should preserve report when capture succeeds":                true,
		"should preserve report when capture directory is unwritable": false,
	} {
		t.Run(name, func(t *testing.T) {
			cfg.DebugDir = filepath.Join(t.TempDir(), "captures")
			if !enabled {
				require.NoError(t, os.WriteFile(cfg.DebugDir, nil, 0600))
			}
			before := requests
			got, err := NewNodeIndexer(cfg).IndexNode(t.Context())
			require.NoError(t, err)
			assert.Empty(t, cmp.Diff(normalizePackageIDs(want), normalizePackageIDs(got), protocmp.Transform()),
				"diagnostics must not change package data or repository assignments")
			assert.Equal(t, before+1, requests, "capture must use the scanner's request")
			if !enabled {
				return
			}
			dirs, err := filepath.Glob(filepath.Join(cfg.DebugDir, "scan-*"))
			require.NoError(t, err)
			require.Len(t, dirs, 1)
			for _, file := range []string{
				"scan.json", "index-root-files.json", "index-root-manifest-globs.json",
				"repositories.json", "packages-raw.json", "packages-filtered.json",
				"coalesced.json", "rpm-assignments.json", "rhcos-identity.json", "index-report.json",
			} {
				b, err := os.ReadFile(filepath.Join(dirs[0], file))
				require.NoError(t, err)
				assert.True(t, json.Valid(b), file)
			}
			b, err := os.ReadFile(filepath.Join(dirs[0], "mapping-response.json"))
			require.NoError(t, err)
			assert.Equal(t, mapping, string(b))
			b, err = os.ReadFile(filepath.Join(dirs[0], "packages-raw.json"))
			require.NoError(t, err)
			var raw []*claircore.Package
			require.NoError(t, json.Unmarshal(b, &raw))
			require.NotEmpty(t, raw)
			assert.Empty(t, raw[0].ID, "capture precedes ID assignment")
			b, err = os.ReadFile(filepath.Join(dirs[0], "index-root/usr/share/buildinfo/content_manifest.json"))
			require.NoError(t, err)
			original, err := os.ReadFile("testdata/usr/share/buildinfo/content_manifest.json")
			require.NoError(t, err)
			assert.Equal(t, original, b)
		})
	}
}

// normalizePackageIDs removes IDs derived from nondeterministic RPM traversal order.
func normalizePackageIDs(report *v4.IndexReport) *v4.IndexReport {
	out := report.CloneVT()
	packages := make(map[string]*v4.Package)
	environments := make(map[string]*v4.Environment_List)
	for id, pkg := range out.GetContents().GetPackages() {
		key := strings.Join([]string{pkg.GetName(), pkg.GetVersion(), pkg.GetArch(), pkg.GetKind(), pkg.GetPackageDb()}, "|")
		pkg.Id = key
		packages[key] = pkg
		environments[key] = out.GetContents().GetEnvironments()[id]
	}
	out.Contents.Packages = packages
	out.Contents.Environments = environments
	// The compatibility lists duplicate the maps and have independent traversal order.
	out.Contents.PackagesDEPRECATED = nil
	out.Contents.EnvironmentsDEPRECATED = nil
	out.Contents.RepositoriesDEPRECATED = nil
	out.Contents.DistributionsDEPRECATED = nil
	return out
}

func TestCaptureAssignmentsExplainsHintFallback(t *testing.T) {
	d := &indexDiagnostics{dir: t.TempDir()}
	d.captureAssignments(&claircore.IndexReport{
		Packages: map[string]*claircore.Package{
			"hinted":   {Name: "a", RepositoryHint: "repoid=baseos"},
			"unhinted": {Name: "b"},
		},
		Repositories: map[string]*claircore.Repository{"repo": {ID: "repo", URI: "repoid=baseos"}},
		Environments: map[string][]*claircore.Environment{
			"hinted": {{RepositoryIDs: []string{"repo"}}},
		},
	})
	b, err := os.ReadFile(filepath.Join(d.dir, "rpm-assignments.json"))
	require.NoError(t, err)
	var assignments map[string]struct {
		Repoid               string                  `json:"repoid"`
		Rule                 string                  `json:"rule"`
		AssignedRepositories []*claircore.Repository `json:"assigned_repositories"`
	}
	require.NoError(t, json.Unmarshal(b, &assignments))
	assert.Equal(t, "baseos", assignments["hinted"].Repoid)
	require.Len(t, assignments["hinted"].AssignedRepositories, 1)
	assert.Contains(t, assignments["unhinted"].Rule, "no repoid hint")
}
