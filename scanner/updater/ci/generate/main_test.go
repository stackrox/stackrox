package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
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
	require.NoError(t, generateAt(path, false, changed, bundleRevision().Add(2*time.Second)))
	modified, err := os.ReadFile(path)
	require.NoError(t, err)
	require.False(t, bytes.Equal(original, modified))
}

func TestGenerateRevisionMustAdvanceWhenBundleChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	initialRevision := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	require.NoError(t, generateAt(path, false, testFixtures(), initialRevision))
	initial, err := os.ReadFile(path)
	require.NoError(t, err)

	// Repeating the same archive is safe even when its revision is unchanged.
	require.NoError(t, generateAt(path, false, testFixtures(), initialRevision))
	repeated, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, initial, repeated)

	changed := testFixtures()
	changed[0].Vulnerabilities[0].FixedInVersion = "1.14.2-r6"
	for name, revision := range map[string]time.Time{
		"equal": initialRevision,
		"older": initialRevision.Add(-time.Second),
	} {
		t.Run(name, func(t *testing.T) {
			err := generateAt(path, false, changed, revision)
			require.ErrorContains(t, err, "must be later than existing member")
			got, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			require.Equal(t, initial, got, "failed generation must preserve the destination")
		})
	}

	newRevision := initialRevision.Add(2 * time.Second)
	require.NoError(t, generateAt(path, false, changed, newRevision))
	reader, err := zip.OpenReader(path)
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	require.True(t, reader.File[0].Modified.Equal(newRevision))
	rc, err := reader.File[0].Open()
	require.NoError(t, err)
	decoder, err := zstd.NewReader(rc)
	require.NoError(t, err)
	var firstRecord map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(decoder).Decode(&firstRecord))
	var gotRevision time.Time
	require.NoError(t, json.Unmarshal(firstRecord["Date"], &gotRevision))
	require.Equal(t, newRevision, gotRevision)
	decoder.Close()
	require.NoError(t, rc.Close())
	require.NoError(t, reader.Close())

	// The accepted revision produces deterministic bytes for the same inputs.
	accepted, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, generateAt(path, false, changed, newRevision))
	again, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, accepted, again)
}

func TestGenerateRejectsInvalidExistingArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	const invalid = "not a zip archive"
	require.NoError(t, os.WriteFile(path, []byte(invalid), 0644))
	err := generateAt(path, false, testFixtures(), time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
	require.ErrorContains(t, err, "existing bundle")
	got, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, invalid, string(got))
}

func TestGenerateRevisionMustExceedEveryExistingMember(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	fixtures := testFixtures()
	second := testFixtures()[0]
	second.Member = "second.json.zst"
	second.Updater = "second-updater"
	fixtures = append(fixtures, second)
	initialRevision := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	require.NoError(t, generateAt(path, false, fixtures, initialRevision))
	latestMemberRevision := initialRevision.Add(10 * time.Second)
	rewriteArchiveMemberTime(t, path, "second.json.zst", latestMemberRevision)
	initial, err := os.ReadFile(path)
	require.NoError(t, err)

	changed := testFixtures()
	changed[0].Vulnerabilities[0].FixedInVersion = "1.14.2-r6"
	changed = append(changed, second)
	err = generateAt(path, false, changed, initialRevision.Add(5*time.Second))
	require.ErrorContains(t, err, `member "second.json.zst"`)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, initial, got, "a stale candidate must preserve the mixed-revision archive")
}

func rewriteArchiveMemberTime(t *testing.T, path, memberName string, revision time.Time) {
	t.Helper()
	source, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { _ = source.Close() }()
	tmp, err := os.CreateTemp(filepath.Dir(path), ".timestamp-rewrite-*.zip")
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(tmp.Name())
	}()
	writer := zip.NewWriter(tmp)
	for _, member := range source.File {
		header := member.FileHeader
		if member.Name == memberName {
			header.SetModTime(revision)
		}
		out, err := writer.CreateHeader(&header)
		require.NoError(t, err)
		in, err := member.Open()
		require.NoError(t, err)
		_, copyErr := io.Copy(out, in)
		require.NoError(t, copyErr)
		require.NoError(t, in.Close())
	}
	require.NoError(t, writer.Close())
	require.NoError(t, tmp.Close())
	require.NoError(t, source.Close())
	require.NoError(t, os.Rename(tmp.Name(), path))
}

func TestGenerateRevisionAllowsMissingDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	require.NoError(t, generateAt(path, false, testFixtures(), time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)))
	require.NoError(t, validateArchive(path))
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
		"invalid updater":  func(f []operation) { f[0].Updater = "updater\x00invalid" },
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
	require.Len(t, first, 1)
	require.Len(t, first[0].Vulnerabilities, 2)
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
