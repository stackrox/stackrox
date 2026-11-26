package inmem

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unique"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeVuln(name, pkg string, kind types.PackageKind, mutate func(*claircore.Vulnerability)) *claircore.Vulnerability {
	v := &claircore.Vulnerability{
		Name:               name,
		Description:        "desc-" + name,
		Links:              "https://example.invalid/" + name,
		Severity:           "CVSS:3.1/AV:N/AC:L",
		NormalizedSeverity: claircore.High,
		Issued:             time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		Package: &claircore.Package{
			Name: pkg,
			Kind: kind,
		},
	}
	if mutate != nil {
		mutate(v)
	}
	return v
}

func makeRec(pkgID, pkgName string, kind types.PackageKind, mutate func(*claircore.IndexRecord)) *claircore.IndexRecord {
	r := &claircore.IndexRecord{
		Package: &claircore.Package{
			ID:   pkgID,
			Name: pkgName,
			Kind: kind,
		},
	}
	if mutate != nil {
		mutate(r)
	}
	return r
}

func newTestStore(t *testing.T, byUpdater map[string][]*claircore.Vulnerability) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewMatcherStore(Options{BlobDir: dir})
	require.NoError(t, err)
	t.Cleanup(s.Close)

	ctx := context.Background()
	for upd, vs := range byUpdater {
		_, err := s.UpdateVulnerabilities(ctx, upd, driver.Fingerprint("fp-"+upd), vs)
		require.NoError(t, err)
	}
	return s, dir
}

func vulnNames(vulns []*claircore.Vulnerability) []string {
	out := make([]string, 0, len(vulns))
	for _, v := range vulns {
		out = append(out, v.Name)
	}
	return out
}

func TestGetNameAndSourceAndKind(t *testing.T) {
	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"deb": {
			makeVuln("CVE-BIN", "libfoo", types.BinaryPackage, nil),
			makeVuln("CVE-SRC", "foo-src", types.SourcePackage, nil),
			makeVuln("CVE-WRONG-KIND", "libfoo", types.SourcePackage, nil),
		},
	})

	rec := makeRec("pkg-1", "libfoo", types.BinaryPackage, func(r *claircore.IndexRecord) {
		r.Package.Source = &claircore.Package{Name: "foo-src", Kind: types.SourcePackage}
	})
	got, err := s.Get(context.Background(), []*claircore.IndexRecord{rec}, datastore.GetOpts{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"CVE-BIN", "CVE-SRC"}, vulnNames(got["pkg-1"]))
	for _, v := range got["pkg-1"] {
		assert.Equal(t, "desc-"+v.Name, v.Description)
		assert.Equal(t, "https://example.invalid/"+v.Name, v.Links)
		assert.Equal(t, "CVSS:3.1/AV:N/AC:L", v.Severity)
	}
}

func TestGetMatchConstraints(t *testing.T) {
	cpeA := cpe.MustUnbind("cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*")
	cpeB := cpe.MustUnbind("cpe:2.3:o:redhat:enterprise_linux:8:*:*:*:*:*:*:*")

	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"u": {
			makeVuln("CVE-MATCH", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Package.Module = "mod1"
				v.FixedInVersion = "1.2.3"
				v.Dist = &claircore.Distribution{
					DID: "rhel", Name: "Red Hat Enterprise Linux", VersionID: "9",
					Version: "9", VersionCodeName: "plow", PrettyName: "RHEL 9",
					Arch: "x86_64", CPE: cpeA,
				}
				v.Repo = &claircore.Repository{Name: "repo-a", Key: "rhel-cpe-repository", URI: "https://rh.example"}
			}),
			makeVuln("CVE-MOD", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Package.Module = "mod2"
				v.FixedInVersion = "1.2.3"
				v.Dist = &claircore.Distribution{DID: "rhel", VersionID: "9", CPE: cpeA}
				v.Repo = &claircore.Repository{Name: "repo-a", Key: "rhel-cpe-repository"}
			}),
			makeVuln("CVE-DIST", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Package.Module = "mod1"
				v.FixedInVersion = "1.2.3"
				v.Dist = &claircore.Distribution{DID: "rhel", VersionID: "8", CPE: cpeB}
				v.Repo = &claircore.Repository{Name: "repo-a", Key: "rhel-cpe-repository"}
			}),
			makeVuln("CVE-REPO", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Package.Module = "mod1"
				v.FixedInVersion = "1.2.3"
				v.Dist = &claircore.Distribution{DID: "rhel", VersionID: "9", CPE: cpeA}
				v.Repo = &claircore.Repository{Name: "repo-b", Key: "other"}
			}),
			makeVuln("CVE-NOFIX", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Package.Module = "mod1"
				v.FixedInVersion = ""
				v.Dist = &claircore.Distribution{DID: "rhel", VersionID: "9", CPE: cpeA}
				v.Repo = &claircore.Repository{Name: "repo-a", Key: "rhel-cpe-repository"}
			}),
		},
	})

	rec := makeRec("pkg-1", "openssl", types.BinaryPackage, func(r *claircore.IndexRecord) {
		r.Package.Module = "mod1"
		r.Distribution = &claircore.Distribution{
			DID: "rhel", Name: "Red Hat Enterprise Linux", VersionID: "9",
			Version: "9", VersionCodeName: "plow", PrettyName: "RHEL 9",
			Arch: "x86_64", CPE: cpeA,
		}
		r.Repository = &claircore.Repository{Name: "repo-a", Key: "rhel-cpe-repository"}
	})

	got, err := s.Get(context.Background(), []*claircore.IndexRecord{rec}, datastore.GetOpts{
		Matchers: []driver.MatchConstraint{
			driver.PackageModule,
			driver.DistributionDID,
			driver.DistributionName,
			driver.DistributionVersionID,
			driver.DistributionVersion,
			driver.DistributionVersionCodeName,
			driver.DistributionPrettyName,
			driver.DistributionArch,
			driver.DistributionCPE,
			driver.RepositoryName,
			driver.RepositoryKey,
			driver.HasFixedInVersion,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"CVE-MATCH"}, vulnNames(got["pkg-1"]))
}

func TestGetOptsVulnerableFilterAndErrorPropagation(t *testing.T) {
	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"rhel": {
			makeVuln("CVE-KEEP", "curl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.FixedInVersion = "8.0"
			}),
			makeVuln("CVE-DROP", "curl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.FixedInVersion = "7.0"
			}),
		},
	})

	rec := makeRec("pkg-curl", "curl", types.BinaryPackage, nil)

	t.Run("filters rejected candidates before display fill", func(t *testing.T) {
		var sawEmptyDisplay bool
		got, err := s.Get(context.Background(), []*claircore.IndexRecord{rec}, datastore.GetOpts{
			Vulnerable: func(_ context.Context, _ *claircore.IndexRecord, v *claircore.Vulnerability) (bool, error) {
				if v.Description == "" && v.Links == "" && v.Severity == "" {
					sawEmptyDisplay = true
				}
				return v.Name == "CVE-KEEP", nil
			},
		})
		require.NoError(t, err)
		assert.True(t, sawEmptyDisplay, "opts.Vulnerable should run before display decompression")
		require.Len(t, got["pkg-curl"], 1)
		assert.Equal(t, "CVE-KEEP", got["pkg-curl"][0].Name)
		assert.Equal(t, "desc-CVE-KEEP", got["pkg-curl"][0].Description)
		assert.Equal(t, "https://example.invalid/CVE-KEEP", got["pkg-curl"][0].Links)
	})

	t.Run("propagates error from Vulnerable callback", func(t *testing.T) {
		sentinel := errors.New("matcher boom")
		_, err := s.Get(context.Background(), []*claircore.IndexRecord{rec}, datastore.GetOpts{
			Vulnerable: func(_ context.Context, _ *claircore.IndexRecord, _ *claircore.Vulnerability) (bool, error) {
				return false, sentinel
			},
		})
		require.ErrorIs(t, err, sentinel)
	})
}

func TestGetUnionsMultipleRecordsSharingPackageID(t *testing.T) {
	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"rhel": {
			makeVuln("CVE-REPO-1", "glibc", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe-1", Key: "rhel-cpe-repository"}
			}),
			makeVuln("CVE-REPO-2", "glibc", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe-2", Key: "rhel-cpe-repository"}
			}),
		},
	})

	// Two IndexRecords for the same Package.ID with different CPE repos, plus a
	// duplicate record to verify ID deduplication.
	recs := []*claircore.IndexRecord{
		makeRec("pkg-42", "glibc", types.BinaryPackage, func(r *claircore.IndexRecord) {
			r.Repository = &claircore.Repository{Name: "cpe-1", Key: "rhel-cpe-repository"}
		}),
		makeRec("pkg-42", "glibc", types.BinaryPackage, func(r *claircore.IndexRecord) {
			r.Repository = &claircore.Repository{Name: "cpe-2", Key: "rhel-cpe-repository"}
		}),
		makeRec("pkg-42", "glibc", types.BinaryPackage, func(r *claircore.IndexRecord) {
			r.Repository = &claircore.Repository{Name: "cpe-1", Key: "rhel-cpe-repository"}
		}),
	}

	got, err := s.Get(context.Background(), recs, datastore.GetOpts{
		Matchers: []driver.MatchConstraint{driver.RepositoryName, driver.RepositoryKey},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"CVE-REPO-1", "CVE-REPO-2"}, vulnNames(got["pkg-42"]))
}

func TestVersionFiltering(t *testing.T) {
	mkRange := func(kind string, lo, hi [10]int32) *claircore.Range {
		return &claircore.Range{
			Lower: claircore.Version{Kind: kind, V: lo},
			Upper: claircore.Version{Kind: kind, V: hi},
		}
	}
	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"osv": {
			makeVuln("CVE-IN", "django", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Range = mkRange("pep440", [10]int32{3, 0}, [10]int32{3, 2})
			}),
			makeVuln("CVE-UPPER-EXCL", "django", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Range = mkRange("pep440", [10]int32{2, 0}, [10]int32{3, 1})
			}),
			makeVuln("CVE-OTHER-KIND", "django", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Range = mkRange("semver", [10]int32{3, 0}, [10]int32{4, 0})
			}),
			makeVuln("CVE-NORANGE", "django", types.BinaryPackage, nil),
		},
	})

	rec := makeRec("pkg-dj", "django", types.BinaryPackage, func(r *claircore.IndexRecord) {
		r.Package.NormalizedVersion = claircore.Version{Kind: "pep440", V: [10]int32{3, 1}}
	})
	got, err := s.Get(context.Background(), []*claircore.IndexRecord{rec}, datastore.GetOpts{VersionFiltering: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"CVE-IN"}, vulnNames(got["pkg-dj"]))
}

func TestAliasesAndSelfPreserved(t *testing.T) {
	a1 := claircore.Alias{Space: unique.Make("cve"), Name: "CVE-2025-0001"}
	a2 := claircore.Alias{Space: unique.Make("ghsa"), Name: "GHSA-xxxx"}
	self := claircore.Alias{Space: unique.Make("rhsa"), Name: "RHSA-2025:0001"}

	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"u": {
			makeVuln("RHSA-2025:0001", "bash", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Aliases = []claircore.Alias{a1, a2}
				v.Self = self
				v.Invert = true
			}),
		},
	})

	got, err := s.Get(context.Background(), []*claircore.IndexRecord{makeRec("p1", "bash", types.BinaryPackage, nil)}, datastore.GetOpts{})
	require.NoError(t, err)
	require.Len(t, got["p1"], 1)
	assert.Equal(t, []claircore.Alias{a1, a2}, got["p1"][0].Aliases)
	assert.Equal(t, self, got["p1"][0].Self)
	assert.True(t, got["p1"][0].Invert)
}

func TestEnrichment(t *testing.T) {
	s, _ := newTestStore(t, nil)
	ctx := context.Background()

	_, err := s.UpdateEnrichments(ctx, "epss", "fp-1", []driver.EnrichmentRecord{
		{Tags: []string{"CVE-1", "CVE-1-alt"}, Enrichment: []byte(`{"score":0.9}`)},
		{Tags: []string{"CVE-2"}, Enrichment: []byte(`{"score":0.1}`)},
	})
	require.NoError(t, err)

	got, err := s.GetEnrichment(ctx, "epss", []string{"CVE-999", "CVE-1-alt"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, `{"score":0.9}`, string(got[0].Enrichment))

	miss, err := s.GetEnrichment(ctx, "epss", []string{"CVE-999"})
	require.NoError(t, err)
	assert.Empty(t, miss)

	wrongKind, err := s.GetEnrichment(ctx, "nvd", []string{"CVE-1"})
	require.NoError(t, err)
	assert.Empty(t, wrongKind)
}

func TestDistributionsMatchesPostgresSemantics(t *testing.T) {
	s, _ := newTestStore(t, map[string][]*claircore.Vulnerability{
		"rhel-vex": {
			makeVuln("CVE-RHEL9", "kernel", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*"}
			}),
			makeVuln("CVE-RHEL10-0", "kernel", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:10.0:*:*:*:*:*:*:*"}
			}),
			makeVuln("CVE-RHEL10-1", "kernel", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:10.1:*:*:*:*:*:*:*"}
			}),
			makeVuln("CVE-RHEL-BAD", "kernel", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe:2.3:o:redhat:enterprise_linux:%:*:*:*:*:*:*:*"}
			}),
			makeVuln("CVE-HB", "curl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Repo = &claircore.Repository{Name: "cpe:2.3:a:redhat:hummingbird:1:*:*:*:*:*:*:*"}
			}),
		},
		"os-distros": {
			makeVuln("CVE-UBU", "openssl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Dist = &claircore.Distribution{DID: "ubuntu", VersionID: "22.04", Version: "22.04 (Jammy)"}
			}),
			makeVuln("CVE-ALP17", "musl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Dist = &claircore.Distribution{DID: "alpine", Version: "3.17"}
			}),
			makeVuln("CVE-ALP18", "musl", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Dist = &claircore.Distribution{DID: "alpine", Version: "3.18"}
			}),
			makeVuln("CVE-DEB", "glibc", types.BinaryPackage, func(v *claircore.Vulnerability) {
				v.Dist = &claircore.Distribution{DID: "debian", VersionID: "10", Version: "10 (buster)"}
			}),
		},
	})

	dists, err := s.Distributions(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []claircore.Distribution{
		{DID: "rhel", VersionID: "9", Version: "9"},
		{DID: "rhel", VersionID: "10", Version: "10"},
		{DID: "hummingbird"},
		{DID: "ubuntu", VersionID: "22.04", Version: "22.04 (Jammy)"},
		{DID: "alpine", Version: "3.17"},
		{DID: "alpine", Version: "3.18"},
		{DID: "debian", VersionID: "10", Version: "10 (buster)"},
	}, dists)
}

func TestPartialReupdateAndDeleteUpdateOperations(t *testing.T) {
	ctx := context.Background()
	s, dir := newTestStore(t, nil)

	refA, err := s.UpdateVulnerabilities(ctx, "upd-a", "fp-a1", []*claircore.Vulnerability{
		makeVuln("CVE-A1", "pkg", types.BinaryPackage, func(v *claircore.Vulnerability) {
			v.Dist = &claircore.Distribution{DID: "alpine", Version: "3.18"}
		}),
	})
	require.NoError(t, err)
	refB, err := s.UpdateVulnerabilities(ctx, "upd-b", "fp-b1", []*claircore.Vulnerability{
		makeVuln("CVE-B1", "pkg", types.BinaryPackage, func(v *claircore.Vulnerability) {
			v.Dist = &claircore.Distribution{DID: "debian", VersionID: "12"}
		}),
	})
	require.NoError(t, err)

	// Re-update upd-a only; upd-b must survive and old upd-a blob must be unlinked.
	refA2, err := s.UpdateVulnerabilities(ctx, "upd-a", "fp-a2", []*claircore.Vulnerability{
		makeVuln("CVE-A2", "pkg", types.BinaryPackage, func(v *claircore.Vulnerability) {
			v.Dist = &claircore.Distribution{DID: "alpine", Version: "3.19"}
		}),
	})
	require.NoError(t, err)
	assert.NotEqual(t, refA, refA2)

	got, err := s.Get(ctx, []*claircore.IndexRecord{makeRec("p1", "pkg", types.BinaryPackage, nil)}, datastore.GetOpts{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"CVE-A2", "CVE-B1"}, vulnNames(got["p1"]))

	// Only 2 blob files should exist on disk (old upd-a blob released immediately).
	blobs, err := filepath.Glob(filepath.Join(dir, "inmem-*.blob"))
	require.NoError(t, err)
	assert.Len(t, blobs, 2)
	cols, err := filepath.Glob(filepath.Join(dir, "inmem-cols-*.col"))
	require.NoError(t, err)
	assert.Len(t, cols, 2)
	assert.Positive(t, s.Stats().ColsOffHeap)

	// Delete upd-b operation -> fragment and its blob must be pruned.
	deleted, err := s.DeleteUpdateOperations(ctx, refB)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	got, err = s.Get(ctx, []*claircore.IndexRecord{makeRec("p1", "pkg", types.BinaryPackage, nil)}, datastore.GetOpts{})
	require.NoError(t, err)
	assert.Equal(t, []string{"CVE-A2"}, vulnNames(got["p1"]))
	assert.Equal(t, 1, s.Stats().Fragments)

	dists, err := s.Distributions(ctx)
	require.NoError(t, err)
	assert.Equal(t, []claircore.Distribution{{DID: "alpine", Version: "3.19"}}, dists)

	blobs, err = filepath.Glob(filepath.Join(dir, "inmem-*.blob"))
	require.NoError(t, err)
	assert.Len(t, blobs, 1)
	cols, err = filepath.Glob(filepath.Join(dir, "inmem-cols-*.col"))
	require.NoError(t, err)
	assert.Len(t, cols, 1)
}

func TestSkipNoPackageNameAndCleanBlobDir(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "inmem-display-stale.blob")
	require.NoError(t, os.WriteFile(stale, []byte("junk"), 0o600))
	staleCols := filepath.Join(dir, "inmem-cols-stale.col")
	require.NoError(t, os.WriteFile(staleCols, []byte("junk"), 0o600))
	unrelated := filepath.Join(dir, "keep-me.txt")
	require.NoError(t, os.WriteFile(unrelated, []byte("keep"), 0o600))

	s, err := NewMatcherStore(Options{BlobDir: dir, CleanBlobDir: true})
	require.NoError(t, err)
	defer s.Close()

	_, err = os.Stat(stale)
	assert.True(t, os.IsNotExist(err), "stale blob should be cleaned on startup")
	_, err = os.Stat(staleCols)
	assert.True(t, os.IsNotExist(err), "stale cols file should be cleaned on startup")
	_, err = os.Stat(unrelated)
	assert.NoError(t, err, "unrelated files must not be removed")

	ctx := context.Background()
	_, err = s.UpdateVulnerabilities(ctx, "u", "fp", []*claircore.Vulnerability{
		{Name: "CVE-NOPKG"},
		makeVuln("CVE-EMPTY", "", types.BinaryPackage, nil),
		makeVuln("CVE-OK", "zlib", types.BinaryPackage, nil),
	})
	require.NoError(t, err)
	assert.Equal(t, 1, s.Stats().Rows)

	initd, err := s.Initialized(ctx)
	require.NoError(t, err)
	assert.True(t, initd)
}

func TestConcurrentGetAndUpdateNoRaceOrMunmapCrash(t *testing.T) {
	dir := t.TempDir()
	s, err := NewMatcherStore(Options{BlobDir: dir})
	require.NoError(t, err)

	ctx := context.Background()
	makeBatch := func(gen int) []*claircore.Vulnerability {
		vs := make([]*claircore.Vulnerability, 0, 20)
		for i := range 20 {
			vs = append(vs, makeVuln(
				fmt.Sprintf("CVE-%d-%d", gen, i),
				fmt.Sprintf("pkg-%d", i%5),
				types.BinaryPackage,
				func(v *claircore.Vulnerability) {
					v.Description = fmt.Sprintf("long description payload gen=%d idx=%d", gen, i)
					v.Dist = &claircore.Distribution{DID: "debian", VersionID: "12"}
				},
			))
		}
		return vs
	}

	_, err = s.UpdateVulnerabilities(ctx, "u1", "fp-0", makeBatch(0))
	require.NoError(t, err)
	_, err = s.UpdateEnrichments(ctx, "epss", "fp-0", []driver.EnrichmentRecord{
		{Tags: []string{"CVE-0-0"}, Enrichment: []byte(`{"score":0.5}`)},
	})
	require.NoError(t, err)

	var stop atomic.Bool
	var wg sync.WaitGroup

	// Readers continuously read vulnerabilities (triggering mmap display
	// decompression), enrichments, distributions, and stats.
	for r := range 6 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			recs := []*claircore.IndexRecord{
				makeRec("p0", "pkg-0", types.BinaryPackage, nil),
				makeRec("p1", fmt.Sprintf("pkg-%d", id%5), types.BinaryPackage, nil),
			}
			for !stop.Load() {
				got, err := s.Get(ctx, recs, datastore.GetOpts{
					Vulnerable: func(_ context.Context, _ *claircore.IndexRecord, _ *claircore.Vulnerability) (bool, error) {
						return true, nil
					},
				})
				if !assert.NoError(t, err) {
					return
				}
				for _, vs := range got {
					for _, v := range vs {
						_ = v.Description
					}
				}
				_, err = s.GetEnrichment(ctx, "epss", []string{"CVE-0-0"})
				if !assert.NoError(t, err) {
					return
				}
				_, err = s.Distributions(ctx)
				if !assert.NoError(t, err) {
					return
				}
				_ = s.Stats()
			}
		}(r)
	}

	// Writer rapidly swaps fragments (unmapping old blobs once readers drain).
	for gen := 1; gen <= 40; gen++ {
		_, err := s.UpdateVulnerabilities(ctx, "u1", driver.Fingerprint(fmt.Sprintf("fp-%d", gen)), makeBatch(gen))
		require.NoError(t, err)
		_, err = s.UpdateEnrichments(ctx, "epss", driver.Fingerprint(fmt.Sprintf("fp-%d", gen)), []driver.EnrichmentRecord{
			{Tags: []string{"CVE-0-0"}, Enrichment: []byte(fmt.Sprintf(`{"gen":%d}`, gen))},
		})
		require.NoError(t, err)
	}

	// Close while readers are still active, then stop and drain readers.
	s.Close()
	stop.Store(true)
	wg.Wait()

	// Every mmap blob must be unmapped and removed once Close + readers finish.
	blobs, err := filepath.Glob(filepath.Join(dir, "inmem-*.blob"))
	require.NoError(t, err)
	assert.Empty(t, blobs, "all blob files should be released after Close and reader drain")
	cols, err := filepath.Glob(filepath.Join(dir, "inmem-cols-*.col"))
	require.NoError(t, err)
	assert.Empty(t, cols, "all cols files should be released after Close and reader drain")
}

func TestParseColsValidation(t *testing.T) {
	validHeader := func() []byte {
		buf := make([]byte, colsMetaSize+16)
		copy(buf[0:8], colsMagic)
		buf[8] = colsVersion
		buf[12] = hostEndian()
		buf[16] = byte(colSectionCount)
		return buf
	}

	t.Run("truncated file", func(t *testing.T) {
		_, err := parseCols(&mmapRegion{data: make([]byte, 16)}, 16)
		assert.ErrorContains(t, err, "too short")
	})
	t.Run("bad magic", func(t *testing.T) {
		buf := validHeader()
		copy(buf[0:8], "BADMAGIC")
		_, err := parseCols(&mmapRegion{data: buf}, int64(len(buf)))
		assert.ErrorContains(t, err, "invalid columns magic")
	})
	t.Run("bad version", func(t *testing.T) {
		buf := validHeader()
		buf[8] = 99
		_, err := parseCols(&mmapRegion{data: buf}, int64(len(buf)))
		assert.ErrorContains(t, err, "unsupported columns version")
	})
	t.Run("foreign endianness", func(t *testing.T) {
		buf := validHeader()
		buf[12] = 1 - hostEndian()
		_, err := parseCols(&mmapRegion{data: buf}, int64(len(buf)))
		assert.ErrorContains(t, err, "endianness mismatch")
	})
	t.Run("unaligned section offset", func(t *testing.T) {
		buf := validHeader()
		// Section 0: offset = colsMetaSize+3 (not 8-byte aligned), count = 1.
		buf[colsHeaderSize] = byte((colsMetaSize + 3) & 0xff)
		buf[colsHeaderSize+1] = byte((colsMetaSize + 3) >> 8)
		buf[colsHeaderSize+8] = 1
		_, err := parseCols(&mmapRegion{data: buf}, int64(len(buf)))
		assert.ErrorContains(t, err, "unaligned")
	})
	t.Run("out of bounds section", func(t *testing.T) {
		buf := validHeader()
		buf[colsHeaderSize] = byte(colsMetaSize & 0xff)
		buf[colsHeaderSize+1] = byte(colsMetaSize >> 8)
		buf[colsHeaderSize+8] = 100 // 100*4 > 16 bytes available
		_, err := parseCols(&mmapRegion{data: buf}, int64(len(buf)))
		assert.ErrorContains(t, err, "out of bounds")
	})
}
