package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/operatorbundle"
)

// severityString renders a storage.VulnerabilitySeverity as a short human-readable label.
func severityString(s storage.VulnerabilitySeverity) string {
	switch s {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return "Low"
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return "Moderate"
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return "Important"
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return "Critical"
	default:
		return "Unknown"
	}
}

// --- JSON output DTOs (severity as string, stable field names) ---

type cveView struct {
	ID       string  `json:"cve"`
	Severity string  `json:"severity"`
	CVSS     float32 `json:"cvss"`
	FixedBy  string  `json:"fixedBy,omitempty"`
}

type imageDiffView struct {
	Repository      string    `json:"repository"`
	Name            string    `json:"name,omitempty"`
	Status          string    `json:"status"`
	InstalledDigest string    `json:"installedDigest,omitempty"`
	CandidateDigest string    `json:"candidateDigest,omitempty"`
	Fixed           []cveView `json:"fixed"`
	StillActive     []cveView `json:"stillActive"`
	New             []cveView `json:"new"`
}

type reportView struct {
	Package           string          `json:"package"`
	Channel           string          `json:"channel"`
	InstalledVersion  string          `json:"installedVersion"`
	UpdateToVersion   string          `json:"updateToVersion,omitempty"`
	NoUpdateReason    string          `json:"noUpdateReason,omitempty"`
	TriggeringDigests []string        `json:"triggeringDigests"`
	ImageDiffs        []imageDiffView `json:"imageDiffs,omitempty"`
}

func toCVEViews(cves []operatorbundle.CVE) []cveView {
	out := make([]cveView, 0, len(cves))
	for _, c := range cves {
		out = append(out, cveView{ID: c.ID, Severity: severityString(c.Severity), CVSS: c.CVSS, FixedBy: c.FixedBy})
	}
	return out
}

func toReportViews(reports []operatorbundle.BundleDiffReport) []reportView {
	views := make([]reportView, 0, len(reports))
	for _, r := range reports {
		v := reportView{
			Package:           r.Package,
			Channel:           r.ChannelName,
			InstalledVersion:  r.InstalledBundle.Version,
			NoUpdateReason:    r.NoUpdateReason,
			TriggeringDigests: r.TriggeringDigests,
		}
		if r.UpdateCandidate != nil {
			v.UpdateToVersion = r.UpdateCandidate.Version
		}
		for _, d := range r.ImageDiffs {
			v.ImageDiffs = append(v.ImageDiffs, imageDiffView{
				Repository:      d.Repository,
				Name:            d.Name,
				Status:          string(d.Status),
				InstalledDigest: d.InstalledDigest,
				CandidateDigest: d.CandidateDigest,
				Fixed:           toCVEViews(d.Fixed),
				StillActive:     toCVEViews(d.StillActive),
				New:             toCVEViews(d.New),
			})
		}
		views = append(views, v)
	}
	return views
}

// renderCSV writes one row per (image, CVE): installed_image,candidate_image,cve,status, where
// status is fixed / not-fixed / new. Image cells are repository@digest, or empty when there is
// no counterpart (candidate for a REMOVED image, installed for an ADDED one).
func renderCSV(w io.Writer, reports []operatorbundle.BundleDiffReport) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"installed_image", "candidate_image", "cve", "status"}); err != nil {
		return errors.Wrap(err, "writing CSV header")
	}
	for _, r := range reports {
		for _, d := range r.ImageDiffs {
			installed := imageRef(d.Repository, d.InstalledDigest)
			candidate := imageRef(d.Repository, d.CandidateDigest)
			for _, group := range []struct {
				status string
				cves   []operatorbundle.CVE
			}{
				{"fixed", d.Fixed},
				{"not-fixed", d.StillActive},
				{"new", d.New},
			} {
				for _, c := range group.cves {
					if err := cw.Write([]string{installed, candidate, c.ID, group.status}); err != nil {
						return errors.Wrap(err, "writing CSV row")
					}
				}
			}
		}
	}
	cw.Flush()
	return errors.Wrap(cw.Error(), "flushing CSV")
}

// imageRef returns repository@digest, or "" when the digest is absent.
func imageRef(repository, digest string) string {
	if digest == "" {
		return ""
	}
	return repository + "@" + digest
}

func renderJSON(w io.Writer, reports []operatorbundle.BundleDiffReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(toReportViews(reports)); err != nil {
		return errors.Wrap(err, "encoding report as JSON")
	}
	return nil
}

func renderTable(w io.Writer, reports []operatorbundle.BundleDiffReport, unresolved []string) {
	for _, r := range reports {
		fmt.Fprintf(w, "\nOperator package: %s (channel %s)\n", r.Package, r.ChannelName)
		fmt.Fprintf(w, "  Installed bundle version: %s\n", r.InstalledBundle.Version)
		if !r.HasUpdate() {
			fmt.Fprintf(w, "  No update candidate: %s\n", r.NoUpdateReason)
			continue
		}
		fmt.Fprintf(w, "  Recommended update:       %s (install to fix CVEs below)\n", r.UpdateCandidate.Version)

		var totalFixed, totalActive, totalNew int
		for _, d := range r.ImageDiffs {
			imageLabel := d.Repository
			if d.Name != "" {
				imageLabel = fmt.Sprintf("%s (%s)", d.Repository, d.Name)
			}
			fmt.Fprintf(w, "\n  Image: %s [%s]\n", imageLabel, d.Status)
			printCVELine(w, "fixed", d.Fixed)
			printCVELine(w, "still active", d.StillActive)
			printCVELine(w, "new", d.New)
			totalFixed += len(d.Fixed)
			totalActive += len(d.StillActive)
			totalNew += len(d.New)
		}
		fmt.Fprintf(w, "\n  Summary: %d fixed, %d still active, %d newly introduced\n",
			totalFixed, totalActive, totalNew)
	}
	if len(unresolved) > 0 {
		fmt.Fprintf(w, "\nUnresolved digests (no operator bundle matched):\n")
		for _, d := range unresolved {
			fmt.Fprintf(w, "  %s\n", d)
		}
	}
}

func printCVELine(w io.Writer, label string, cves []operatorbundle.CVE) {
	if len(cves) == 0 {
		return
	}
	parts := make([]string, 0, len(cves))
	for _, c := range cves {
		parts = append(parts, fmt.Sprintf("%s (%s)", c.ID, severityString(c.Severity)))
	}
	fmt.Fprintf(w, "    %-13s %d: %s\n", label+":", len(cves), strings.Join(parts, ", "))
}
