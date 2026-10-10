package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"slices"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/stretchr/testify/require"
)

func archiveBytes(t *testing.T, fixtures []operation) []byte {
	t.Helper()
	ops, err := entries(fixtures)
	require.NoError(t, err)
	var b bytes.Buffer
	require.NoError(t, writeArchive(&b, ops))
	return b.Bytes()
}

func TestArchiveOrdering(t *testing.T) {
	f := testFixtures()
	v := *f[0].Vulnerabilities[0]
	v.Name = "CVE-other"
	f[0].Vulnerabilities = append(f[0].Vulnerabilities, &v)
	f[0].Enrichments = []enrichmentFixture{
		{Tags: []string{"extra", v.Name}, Payload: map[string]any{"id": v.Name, "nested": map[string]any{"list": []int{1, 2}}}},
		{Tags: []string{f[0].Vulnerabilities[0].Name}, Payload: map[string]string{"id": f[0].Vulnerabilities[0].Name}},
	}
	f = append(f, operation{Member: "other.json.zst", Updater: "another", Vulnerabilities: []*claircore.Vulnerability{&v}})
	expected := archiveBytes(t, f)
	shuffled := []operation{f[1],
		{Member: f[0].Member, Updater: f[0].Updater, Enrichments: []enrichmentFixture{f[0].Enrichments[1]}},
		{Member: f[0].Member, Updater: f[0].Updater, Vulnerabilities: []*claircore.Vulnerability{&v}},
		{Member: f[0].Member, Updater: f[0].Updater, Vulnerabilities: []*claircore.Vulnerability{f[0].Vulnerabilities[0], &v}, Enrichments: []enrichmentFixture{f[0].Enrichments[0]}},
	}
	require.Equal(t, expected, archiveBytes(t, shuffled))
	require.Equal(t, expected, archiveBytes(t, f))
}

func decodedArchive(t *testing.T, path string) map[string][]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, r.Close()) }()
	result := make(map[string][]string)
	for _, member := range r.File {
		rc, err := member.Open()
		require.NoError(t, err)
		zr, err := zstd.NewReader(rc)
		require.NoError(t, err)
		dec := json.NewDecoder(zr)
		for {
			var record map[string]json.RawMessage
			err := dec.Decode(&record)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			// Remaining fields include updater, kind and the complete native payload.
			delete(record, "Ref")
			delete(record, "Date")
			delete(record, "Fingerprint")
			b, err := json.Marshal(record)
			require.NoError(t, err)
			result[member.Name] = append(result[member.Name], string(b))
		}
		zr.Close()
		require.NoError(t, rc.Close())
		slices.Sort(result[member.Name])
	}
	return result
}

func TestArchivePayloadCompatibility(t *testing.T) {
	candidate := filepath.Join(t.TempDir(), "candidate.zip")
	require.NoError(t, generate(candidate, false, fixtures()))
	baseline := filepath.Join("..", "..", "..", "image", "scanner", "bundles", "ci-minimal", "vulnerabilities.zip")
	require.Equal(t, decodedArchive(t, baseline), decodedArchive(t, candidate))
}
