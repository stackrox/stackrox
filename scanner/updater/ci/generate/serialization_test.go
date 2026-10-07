package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRecords(t *testing.T) {
	ref := uuid.NewV5(namespaceOID, "test")
	date := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, input := range map[string]string{
		"vulnerability": `{"Ref":"old","Date":"old","Unknown":{"nested":[1,"<>&"]},"Vuln":{"Name":"CVE","extra":true}}`,
		"enrichment":    `{"Ref":"old","Date":"old","Enrichment":{"Tags":["CVE"],"Enrichment":{"id":"CVE","extra":[1,2]}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			require.NoError(t, normalizeRecords(&output, bytes.NewBufferString(input), ref, date))
			var before, after map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(input), &before))
			require.NoError(t, json.Unmarshal(output.Bytes(), &after))
			require.JSONEq(t, `"`+ref.String()+`"`, string(after["Ref"]))
			require.JSONEq(t, `"2000-01-01T00:00:00Z"`, string(after["Date"]))
			delete(before, "Ref")
			delete(before, "Date")
			delete(after, "Ref")
			delete(after, "Date")
			require.Equal(t, before, after)
			require.NotContains(t, output.String(), `\u003c`)
		})
	}
	for name, input := range map[string]string{
		"missing ref":  `{"Date":"old"}`,
		"missing date": `{"Ref":"old"}`,
		"invalid":      `{`,
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, normalizeRecords(&bytes.Buffer{}, bytes.NewBufferString(input), ref, date))
		})
	}
	require.ErrorIs(t, normalizeRecords(brokenWriter{}, bytes.NewBufferString(`{"Ref":"old","Date":"old"}`), ref, date),
		io.ErrClosedPipe)
}

func TestEnrichmentFingerprint(t *testing.T) {
	f := []operation{{Member: "nvd.json.zst", Updater: "nvd", Enrichments: []enrichmentFixture{
		{Tags: []string{"CVE-test"}, Payload: map[string]string{"id": "CVE-test", "description": "original"}},
	}}}
	first, err := entries(f)
	require.NoError(t, err)
	f[0].Enrichments[0].Payload = map[string]string{"id": "CVE-test", "description": "changed"}
	second, err := entries(f)
	require.NoError(t, err)
	require.NotEqual(t, first[0].Fingerprint, second[0].Fingerprint)
	require.NotEqual(t, first[0].Ref, second[0].Ref)
	f[0].Member = "other.json.zst"
	third, err := entries(f)
	require.NoError(t, err)
	require.NotEqual(t, second[0].Fingerprint, third[0].Fingerprint)
	require.NotEqual(t, second[0].Ref, third[0].Ref)
}
