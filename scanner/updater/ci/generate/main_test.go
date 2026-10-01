package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/require"
)

func testFixtures() []operation {
	return []operation{{Member: "alpine.json.zst", Updater: "alpine-v3.9", Vulnerabilities: []*claircore.Vulnerability{{Name: "CVE-2019-20372", Description: "nginx request smuggling", Package: &claircore.Package{Name: "nginx"}, Dist: &claircore.Distribution{DID: "alpine", VersionID: "3.9"}, FixedInVersion: "1.14.2-r5"}}}}
}

func TestGenerate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	require.NoError(t, generate(path, false, testFixtures()))
	original, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, generate(path, true, testFixtures()))
	require.NoError(t, generate(path, false, testFixtures()))
	repeated, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, repeated)
	changed := testFixtures()
	changed[0].Vulnerabilities[0].FixedInVersion = "1.14.2-r6"
	require.ErrorContains(t, generate(path, true, changed), "differs")
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, unchanged)
	require.NoError(t, generate(path, false, changed))
	modified, err := os.ReadFile(path)
	require.NoError(t, err)
	require.False(t, bytes.Equal(original, modified))
}

func TestInvalidFixturesPreserveOutput(t *testing.T) {
	for name, mutate := range map[string]func([]operation){
		"empty operation": func(f []operation) { f[0].Vulnerabilities = nil },
		"unconnected enrichment": func(f []operation) {
			f[0].Enrichments = []enrichmentFixture{{Tags: []string{"different"}, Payload: map[string]string{"id": "CVE-2019-20372"}}}
		},
		"missing package":  func(f []operation) { f[0].Vulnerabilities[0].Package = nil },
		"missing identity": func(f []operation) { f[0].Vulnerabilities[0].Dist = nil },
		"missing updater":  func(f []operation) { f[0].Updater = "" },
		"invalid enrichment": func(f []operation) {
			f[0].Enrichments = []enrichmentFixture{{Tags: []string{"CVE-2019-20372"}, Payload: make(chan int)}}
		},
		"unsafe member": func(f []operation) { f[0].Member = "../alpine.json.zst" },
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bundle.zip")
			require.NoError(t, os.WriteFile(path, []byte("keep"), 0644))
			f := testFixtures()
			mutate(f)
			require.Error(t, generate(path, false, f))
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "keep", string(got))
		})
	}
}

func TestFingerprint(t *testing.T) {
	f := testFixtures()
	a, err := entries(f)
	require.NoError(t, err)
	f[0].Vulnerabilities[0].Description += " changed"
	b, err := entries(f)
	require.NoError(t, err)
	require.NotEqual(t, a[0].Fingerprint, b[0].Fingerprint)
	require.NotEqual(t, a[0].Ref, b[0].Ref)
	f[0].Updater += "-other"
	c, err := entries(f)
	require.NoError(t, err)
	require.NotEqual(t, b[0].Fingerprint, c[0].Fingerprint)
}

func TestCanonicalOrdering(t *testing.T) {
	a := testFixtures()
	v := *a[0].Vulnerabilities[0]
	v.Name = "CVE-2017-7529"
	v.FixedInVersion = "1.12.1-r0"
	a[0].Vulnerabilities = append(a[0].Vulnerabilities, &v)
	b := testFixtures()
	b[0].Vulnerabilities = append([]*claircore.Vulnerability{&v}, b[0].Vulnerabilities...)
	// Operation fragments from separate scenario tables must be combined.
	b = append(b, operation{Member: b[0].Member, Updater: b[0].Updater, Vulnerabilities: b[0].Vulnerabilities[1:]})
	b[0].Vulnerabilities = b[0].Vulnerabilities[:1]
	first, err := entries(a)
	require.NoError(t, err)
	second, err := entries(b)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, first[0].Ref, first[1].Ref)
}

func TestFixtureArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "all.zip")
	require.NoError(t, generate(path, false, fixtures()))
	require.NoError(t, validateArchive(path))
	require.NoError(t, generate(path, true, fixtures()))
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteError(t *testing.T) {
	records, err := entries(testFixtures())
	require.NoError(t, err)
	require.ErrorIs(t, writeArchive(brokenWriter{}, records), io.ErrClosedPipe)
}

func TestCheckedInBundle(t *testing.T) {
	path := filepath.Join("..", "..", "..", "image", "scanner", "bundles", "ci-minimal", "vulnerabilities.zip")
	require.NoError(t, generate(path, true, fixtures()))
}
