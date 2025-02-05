package csaf

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/snappy"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
)

func TestParseEnrichment(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	url, err := url.Parse(baseURL)
	if err != nil {
		t.Error(err)
	}

	testcases := []struct {
		name                string
		filename            string
		expectedName        string
		expectedDescription string
		expectedReleaseDate time.Time
		expectedCVEs        []string
		expectedSeverity    string
		expectedCVSSv3      csaf.CVSS
	}{
		{
			name:                "RHBA-2024:0599",
			filename:            "testdata/rhba-2024_0599.jsonl",
			expectedName:        "RHBA-2024:0599",
			expectedDescription: "Red Hat Bug Fix Advisory: Migration Toolkit for Applications bug fix and enhancement update",
			expectedReleaseDate: time.Date(2024, time.January, 30, 13, 46, 48, 0, time.UTC),
			expectedCVEs:        []string{"CVE-2023-39325", "CVE-2023-44487"},
			expectedSeverity:    "Important",
			expectedCVSSv3: csaf.CVSS{
				Score:  7.5,
				Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			},
		},
		{
			name:                "RHSA-2024:0024",
			filename:            "testdata/rhsa-2024_0024.jsonl",
			expectedName:        "RHSA-2024:0024",
			expectedDescription: "Red Hat Security Advisory: firefox security update",
			expectedReleaseDate: time.Date(2024, time.January, 2, 8, 30, 42, 0, time.UTC),
			expectedCVEs:        []string{"CVE-2023-6856", "CVE-2023-6857", "CVE-2023-6858", "CVE-2023-6859", "CVE-2023-6860", "CVE-2023-6861", "CVE-2023-6862", "CVE-2023-6863", "CVE-2023-6864", "CVE-2023-6865", "CVE-2023-6867"},
			expectedSeverity:    "Important",
			expectedCVSSv3: csaf.CVSS{
				Score:  8.8,
				Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
			},
		},
	}

	e := &Enricher{
		base: url,
		c:    http.DefaultClient,
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := zlog.Test(ctx, t)
			f, err := os.Open(tc.filename)
			if err != nil {
				t.Fatalf("failed to open test data file %s: %v", tc.filename, err)
			}

			// Ideally, you'd just use snappy.Encode() but apparently
			// the stream format and the block format are not interchangeable:
			// https://pkg.go.dev/github.com/klauspost/compress/snappy#Writer.
			b, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("failed to read file bytes: %v", err)
			}
			var buf bytes.Buffer
			sw := snappy.NewBufferedWriter(&buf)
			bLen, err := sw.Write(b)
			if err != nil {
				t.Fatalf("error writing snappy data to buffer: %v", err)
			}
			if bLen != len(b) {
				t.Error("didn't write the correct # of bytes")
			}
			if err = sw.Close(); err != nil {
				t.Errorf("failed to close snappy Writer: %v", err)
			}

			enrichments, err := e.ParseEnrichment(ctx, io.NopCloser(&buf))
			if err != nil {
				t.Fatalf("failed to parse CSAF JSON: %v", err)
			}
			if len(enrichments) != 1 {
				t.Errorf("expected %d vulns but got %d", 1, len(enrichments))
			}
			enrichment := enrichments[0]

			var record csaf.Advisory
			err = json.Unmarshal(enrichment.Enrichment, &record)
			if err != nil {
				t.Fatalf("failed to unmarshal record: %v", err)
			}

			if record.Name != tc.expectedName {
				t.Errorf("expected %s but got %s", tc.expectedName, record.Name)
			}
			if record.Description != tc.expectedDescription {
				t.Errorf("expected %s but got %s", tc.expectedDescription, record.Description)
			}
			if record.ReleaseDate != tc.expectedReleaseDate {
				t.Errorf("expected %s but got %s", tc.expectedReleaseDate, record.ReleaseDate)
			}
			if !cmp.Equal(record.CVEs, tc.expectedCVEs) {
				t.Errorf("expected %v but got %v", tc.expectedCVEs, record.CVEs)
			}
			if record.Severity != tc.expectedSeverity {
				t.Errorf("expected %s but got %s", tc.expectedSeverity, record.Severity)
			}
			if record.CVSSv3 != tc.expectedCVSSv3 {
				t.Errorf("expected %v but got %v", tc.expectedCVSSv3, record.CVSSv3)
			}
		})
	}
}
