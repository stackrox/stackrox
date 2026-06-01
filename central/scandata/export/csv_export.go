package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
)

// Format specifies how to export scan findings.
type Format string

const (
	// FormatGrouped groups findings by CVE with MAX severity, comma-separated advisory IDs.
	FormatGrouped Format = "grouped"
	// FormatRaw exports one row per advisory match with individual advisory data.
	FormatRaw Format = "raw"
)

// ExportCSV writes scan findings to a CSV writer in the specified format.
func ExportCSV(w io.Writer, findings []*storage.ScanFinding, format Format) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	switch format {
	case FormatGrouped:
		return exportGrouped(writer, findings)
	case FormatRaw:
		return exportRaw(writer, findings)
	default:
		return exportGrouped(writer, findings)
	}
}

// groupedRow represents a CVE aggregated across multiple advisories.
type groupedRow struct {
	cveName     string
	maxSeverity storage.VulnerabilitySeverity
	maxCVSS     float32
	imageCount  int
	anyFixable  bool
	fixedIn     string
	firstSeen   time.Time
	nvdCVSS     float32
	epss        float32
	advisoryIDs []string
}

func exportGrouped(writer *csv.Writer, findings []*storage.ScanFinding) error {
	// Write header
	header := []string{
		"CVE",
		"Severity",
		"CVSS",
		"Image Count",
		"Fixable",
		"Fixed In",
		"First Seen",
		"NVD CVSS",
		"EPSS",
		"Advisories",
	}
	if err := writer.Write(header); err != nil {
		return errors.Wrap(err, "failed to write CSV header")
	}

	// Group findings by CVE name
	groups := make(map[string]*groupedRow)
	imagesByCVE := make(map[string]map[string]struct{})

	for _, f := range findings {
		cveName := f.GetCveName()
		if cveName == "" {
			continue
		}

		g, exists := groups[cveName]
		if !exists {
			g = &groupedRow{
				cveName:     cveName,
				maxSeverity: f.GetSeverity(),
				maxCVSS:     f.GetCvss(),
				nvdCVSS:     f.GetNvdCvss(),
				epss:        f.GetEpssPercentile(),
				firstSeen:   f.GetFirstSystemOccurrence().AsTime(),
			}
			groups[cveName] = g
			imagesByCVE[cveName] = make(map[string]struct{})
		}

		// Update MAX severity
		if f.GetSeverity() > g.maxSeverity {
			g.maxSeverity = f.GetSeverity()
		}

		// Update MAX CVSS
		if f.GetCvss() > g.maxCVSS {
			g.maxCVSS = f.GetCvss()
		}

		// Track unique images
		if imageID := f.GetImageId(); imageID != "" {
			imagesByCVE[cveName][imageID] = struct{}{}
		}

		// BOOL_OR fixable
		if f.GetIsFixable() {
			g.anyFixable = true
			if g.fixedIn == "" {
				g.fixedIn = f.GetFixedBy()
			}
		}

		// Track first seen (earliest)
		if ts := f.GetFirstSystemOccurrence().AsTime(); !ts.IsZero() && (g.firstSeen.IsZero() || ts.Before(g.firstSeen)) {
			g.firstSeen = ts
		}

		// Collect advisory IDs from JSONB
		for _, advisoryID := range types.GetAllAdvisoryIDs(f.GetAdvisories()) {
			if !slices.Contains(g.advisoryIDs, advisoryID) {
				g.advisoryIDs = append(g.advisoryIDs, advisoryID)
			}
		}
	}

	// Write grouped rows
	for _, g := range groups {
		firstSeenStr := ""
		if !g.firstSeen.IsZero() {
			firstSeenStr = g.firstSeen.Format(time.RFC3339)
		}

		row := []string{
			g.cveName,
			g.maxSeverity.String(),
			formatFloat(g.maxCVSS),
			strconv.Itoa(len(imagesByCVE[g.cveName])),
			strconv.FormatBool(g.anyFixable),
			g.fixedIn,
			firstSeenStr,
			formatFloat(g.nvdCVSS),
			formatFloat(g.epss),
			strings.Join(g.advisoryIDs, ", "),
		}

		if err := writer.Write(row); err != nil {
			return errors.Wrap(err, "failed to write grouped row")
		}
	}

	return nil
}

func exportRaw(writer *csv.Writer, findings []*storage.ScanFinding) error {
	// Write header
	header := []string{
		"CVE",
		"Advisory ID",
		"Advisory Source",
		"Severity",
		"CVSS",
		"NVD CVSS",
		"EPSS",
		"Fixable",
		"Fixed In",
		"Data Source",
		"Description",
	}
	if err := writer.Write(header); err != nil {
		return errors.Wrap(err, "failed to write CSV header")
	}

	// Write one row per finding
	for _, f := range findings {
		row := []string{
			f.GetCveName(),
			types.GetPrimaryAdvisoryID(f.GetAdvisories()),
			types.GetPrimarySourceName(f.GetAdvisories()),
			f.GetSeverity().String(),
			formatFloat(f.GetCvss()),
			formatFloat(f.GetNvdCvss()),
			formatFloat(f.GetEpssPercentile()),
			strconv.FormatBool(f.GetIsFixable()),
			f.GetFixedBy(),
			f.GetDataSource(),
			f.GetDescription(),
		}

		if err := writer.Write(row); err != nil {
			return errors.Wrap(err, "failed to write raw row")
		}
	}

	return nil
}

func formatFloat(f float32) string {
	if f == 0 {
		return "0.0"
	}
	return fmt.Sprintf("%.2f", f)
}
