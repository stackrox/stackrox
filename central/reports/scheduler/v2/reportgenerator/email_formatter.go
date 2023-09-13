package reportgenerator

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/timestamp"
	"golang.org/x/exp/slices"
)

const (
	maxConfigNameLenInSubject     = 40
	maxCollectionNameLenInSubject = 40
)

var (
	cveSeverityToText = map[storage.VulnerabilitySeverity]string{
		storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY:   "Unknown",
		storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:       "Low",
		storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:  "Moderate",
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY: "Important",
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:  "Critical",
	}

	fixabilityToText = map[storage.VulnerabilityReportFilters_Fixability]string{
		storage.VulnerabilityReportFilters_FIXABLE:     "Fixable",
		storage.VulnerabilityReportFilters_NOT_FIXABLE: "Not Fixable",
	}

	imageTypeToText = map[storage.VulnerabilityReportFilters_ImageType]string{
		storage.VulnerabilityReportFilters_DEPLOYED: "Deployed Images",
		storage.VulnerabilityReportFilters_WATCHED:  "Watched Images",
	}
)

func formatEmailSubject(subjectTemplate string, snapshot *storage.ReportSnapshot) (string, error) {
	configName := snapshot.GetName()
	if len(configName) > maxConfigNameLenInSubject {
		configName = fmt.Sprintf("%s...", configName[0:maxConfigNameLenInSubject])
	}
	collectionName := snapshot.GetCollection().GetName()
	if len(collectionName) > maxCollectionNameLenInSubject {
		collectionName = fmt.Sprintf("%s...", collectionName[0:maxCollectionNameLenInSubject])
	}

	data := &reportEmailSubjectFormat{
		BrandedProductNameShort: branding.GetProductNameShort(),
		ReportConfigName:        configName,
		CollectionName:          collectionName,
	}
	tmpl, err := template.New("emailSubject").Parse(subjectTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func formatEmailBody(emailTemplate string) (string, error) {
	data := &reportEmailBodyFormat{
		BrandedProductName:      branding.GetProductName(),
		BrandedProductNameShort: branding.GetProductNameShort(),
	}

	tmpl, err := template.New("emailBody").Parse(emailTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func addReportConfigDetails(emailBody, configDetailsHtml string) string {
	var writer strings.Builder
	writer.WriteString(emailBody)
	writer.WriteString("\n\n")
	writer.WriteString(configDetailsHtml)

	return writer.String()
}

func formatReportConfigDetails(snapshot *storage.ReportSnapshot) (string, error) {
	var writer strings.Builder

	err := validateSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	reportFilters := snapshot.GetVulnReportFilters()

	writer.WriteString("<table style=\"width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;\">")

	// Add severities and fixabilities
	fillTableHeadings(&writer, "CVE Severity", "CVE Status")
	writer.WriteString("<tr>")
	// create a copy because severities will be sorted in descending order (critical, important, moderate, low)
	severities := append([]storage.VulnerabilitySeverity{}, reportFilters.GetSeverities()...)
	slices.SortFunc(severities, func(s1, s2 storage.VulnerabilitySeverity) bool {
		return s1 > s2
	})
	fillTableCellWithValues(&writer, severities...)
	fixabilities := expandFixability(reportFilters.GetFixability())
	fillTableCellWithValues(&writer, fixabilities...)
	writer.WriteString("</tr>")

	// Add collection, image types and CVEs discovered since filters
	fillTableHeadings(&writer, "Report Scope", "Image Type", "CVEs discovered since")
	writer.WriteString("<tr>")
	fillTableCellWithValues(&writer, snapshot.GetCollection())
	// create a copy because image types will be sorted in ascending order (deployed, watched)
	imageTypes := append([]storage.VulnerabilityReportFilters_ImageType{}, reportFilters.GetImageTypes()...)
	slices.Sort(imageTypes)
	fillTableCellWithValues(&writer, imageTypes...)
	fillTableCellWithValues(&writer, reportFilters.GetCvesSince())
	writer.WriteString("</tr>")

	writer.WriteString("</table>")

	return writer.String(), nil
}

func expandFixability(fixability storage.VulnerabilityReportFilters_Fixability) []storage.VulnerabilityReportFilters_Fixability {
	if fixability == storage.VulnerabilityReportFilters_BOTH {
		return []storage.VulnerabilityReportFilters_Fixability{
			storage.VulnerabilityReportFilters_FIXABLE,
			storage.VulnerabilityReportFilters_NOT_FIXABLE,
		}
	}
	return []storage.VulnerabilityReportFilters_Fixability{fixability}
}

func fillTableHeadings(writer *strings.Builder, headings ...string) {
	writer.WriteString("<tr>")
	for _, h := range headings {
		writer.WriteString(fmt.Sprintf("<th style=\"background-color: #f0f0f0; padding: 10px;\">%s</th>", h))
	}
	writer.WriteString("</tr>")
}

func fillTableCellWithValues[T any](writer *strings.Builder, values ...T) {
	writer.WriteString("<td style=\"padding: 10px; word-wrap: break-word; white-space: normal;\">")
	if len(values) > 0 {
		writer.WriteString("<table style=\"width: 100%; border-collapse: collapse; table-layout: fixed; border: none; text-align: left;\">")
		for _, valI := range values {
			writer.WriteString("<tr><td style=\"padding: 10px;\">")
			writer.WriteString(convertValueToFriendlyText(valI))
			writer.WriteString("</td></tr>")
		}
		writer.WriteString("</table>")
	}
	writer.WriteString("</td>")
}

func convertValueToFriendlyText(valI interface{}) string {
	switch val := valI.(type) {
	case *storage.CollectionSnapshot:
		return val.GetName()
	case storage.VulnerabilitySeverity:
		return cveSeverityToText[val]
	case storage.VulnerabilityReportFilters_Fixability:
		return fixabilityToText[val]
	case storage.VulnerabilityReportFilters_ImageType:
		return imageTypeToText[val]
	case *storage.VulnerabilityReportFilters_AllVuln:
		return "All Time"
	case *storage.VulnerabilityReportFilters_SinceLastSentScheduledReport:
		return "Last successful scheduled report"
	case *storage.VulnerabilityReportFilters_SinceStartDate:
		return timestamp.FromProtobuf(val.SinceStartDate).GoTime().Format("January 02, 2006")
	default:
		return ""
	}
}

func validateSnapshot(snapshot *storage.ReportSnapshot) error {
	reportFilters := snapshot.GetVulnReportFilters()
	if reportFilters == nil {
		return errors.New("Report snapshot is missing vulnerability report filters")
	}
	if snapshot.GetCollection() == nil {
		return errors.New("Report snapshot is missing collection snapshot")
	}
	if len(reportFilters.GetImageTypes()) == 0 {
		return errors.New("Report snapshot is missing image type filters")
	}
	if reportFilters.GetCvesSince() == nil {
		return errors.New("Report snapshot is missing 'CVEs since' filter")
	}
	return nil
}
