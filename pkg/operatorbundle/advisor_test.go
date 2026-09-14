package operatorbundle

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fmtSprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

func joined(msgs []string) string { return strings.Join(msgs, "\n") }

// --- in-memory fakes for the three interfaces ---

type fakeCatalog struct {
	installed  map[string]*Bundle // digest -> bundle
	candidates map[string][]Bundle
}

func (f *fakeCatalog) FindInstalledBundle(_ context.Context, digest string) (*Bundle, error) {
	return f.installed[digest], nil
}

func (f *fakeCatalog) FindCandidateBundles(_ context.Context, pkg, _, _ string) ([]Bundle, error) {
	return f.candidates[pkg], nil
}

type fakeScanner struct {
	byRef map[string]*ImageCVEs
	scans []string
}

func (f *fakeScanner) ScanImage(_ context.Context, reference string) (*ImageCVEs, error) {
	f.scans = append(f.scans, reference)
	return f.byRef[reference], nil
}

type fakeInstalled struct {
	byDigest map[string]*ImageCVEs
}

func (f *fakeInstalled) GetCVEsByDigest(_ context.Context, digest string) (*ImageCVEs, error) {
	return f.byDigest[digest], nil
}

func TestAdvise(t *testing.T) {
	sev := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	const repo = "registry.redhat.io/albo/controller-rhel9"

	installedBundle := &Bundle{
		Package:      "aws-load-balancer-operator",
		ChannelName:  "stable-v1",
		Version:      "1.4.0",
		CSVName:      "aws-load-balancer-operator.v1.4.0",
		CreationDate: "2026-06-01",
		RelatedImages: []RelatedImage{
			{Image: repo, Digest: "sha256:old"},
		},
	}
	candidateBundle := Bundle{
		Package:     "aws-load-balancer-operator",
		ChannelName: "stable-v1",
		Version:     "1.4.3",
		CSVName:     "aws-load-balancer-operator.v1.4.3",
		RelatedImages: []RelatedImage{
			{Image: repo, Digest: "sha256:new"},
		},
	}

	catalog := &fakeCatalog{
		installed: map[string]*Bundle{
			"sha256:img-a": installedBundle,
			"sha256:img-b": installedBundle, // second image, same bundle
		},
		candidates: map[string][]Bundle{
			"aws-load-balancer-operator": {
				candidateBundle,
				{Package: "aws-load-balancer-operator", Version: "1.5.0"}, // ignored: higher minor
			},
		},
	}
	installedSrc := &fakeInstalled{byDigest: map[string]*ImageCVEs{
		"sha256:old": {Repository: repo, Digest: "sha256:old", CVEs: []CVE{
			cve("CVE-1", sev, "1.1"), cve("CVE-2", sev, ""),
		}},
	}}
	scanner := &fakeScanner{byRef: map[string]*ImageCVEs{
		repo + "@sha256:new": {Repository: repo, Digest: "sha256:new", CVEs: []CVE{
			cve("CVE-2", sev, ""), cve("CVE-3", sev, "2.0"),
		}},
	}}

	advisor := NewAdvisor(catalog, scanner, installedSrc)

	t.Run("groups digests sharing a bundle and diffs correctly", func(t *testing.T) {
		reports, unresolved, err := advisor.Advise(context.Background(), []string{"sha256:img-a", "sha256:img-b"})
		require.NoError(t, err)
		assert.Empty(t, unresolved)
		require.Len(t, reports, 1, "two images of the same bundle must produce one report")

		r := reports[0]
		require.True(t, r.HasUpdate())
		assert.Equal(t, "1.4.3", r.UpdateCandidate.Version)
		assert.ElementsMatch(t, []string{"sha256:img-a", "sha256:img-b"}, r.TriggeringDigests)

		require.Len(t, r.ImageDiffs, 1)
		d := r.ImageDiffs[0]
		assert.Equal(t, ImagePaired, d.Status)
		assert.Equal(t, []string{"CVE-1"}, cveIDs(d.Fixed))
		assert.Equal(t, []string{"CVE-2"}, cveIDs(d.StillActive))
		assert.Equal(t, []string{"CVE-3"}, cveIDs(d.New))

		// The candidate image is scanned by pinned digest reference.
		assert.Contains(t, scanner.scans, repo+"@sha256:new")
	})

	t.Run("unresolved digests are reported", func(t *testing.T) {
		reports, unresolved, err := advisor.Advise(context.Background(), []string{"sha256:unknown"})
		require.NoError(t, err)
		assert.Empty(t, reports)
		assert.Equal(t, []string{"sha256:unknown"}, unresolved)
	})

	t.Run("progress callback receives step-by-step messages", func(t *testing.T) {
		var messages []string
		a := NewAdvisor(catalog, scanner, installedSrc, WithProgress(func(format string, args ...any) {
			messages = append(messages, fmtSprintf(format, args...))
		}))
		_, _, err := a.Advise(context.Background(), []string{"sha256:img-a"})
		require.NoError(t, err)
		require.NotEmpty(t, messages)
		assert.Contains(t, joined(messages), "Resolving installed bundle")
		assert.Contains(t, joined(messages), "scanning")
	})

	t.Run("no update candidate yields report with reason", func(t *testing.T) {
		noCandidateCatalog := &fakeCatalog{
			installed:  map[string]*Bundle{"sha256:img-a": installedBundle},
			candidates: map[string][]Bundle{}, // none
		}
		a := NewAdvisor(noCandidateCatalog, scanner, installedSrc)
		reports, _, err := a.Advise(context.Background(), []string{"sha256:img-a"})
		require.NoError(t, err)
		require.Len(t, reports, 1)
		assert.False(t, reports[0].HasUpdate())
		assert.NotEmpty(t, reports[0].NoUpdateReason)
		assert.Empty(t, reports[0].ImageDiffs)
	})
}
