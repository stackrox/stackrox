package main

import (
	"bytes"
	"testing"

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
