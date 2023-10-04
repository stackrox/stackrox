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
		storage.VulnerabilityReportFilters_NOT_FIXABLE: "Not fixable",
	}

	imageTypeToText = map[storage.VulnerabilityReportFilters_ImageType]string{
		storage.VulnerabilityReportFilters_DEPLOYED: "Deployed images",
		storage.VulnerabilityReportFilters_WATCHED:  "Watched images",
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
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
	}

	tmpl, err := template.New("emailBody").Parse(emailTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func addReportConfigDetails(emailBody, configDetailsHTML string) string {
	var writer strings.Builder
	writer.WriteString(emailBody)
	writer.WriteString("<br><br>")
	writer.WriteString(configDetailsHTML)

	return writer.String()
}

func formatReportConfigDetails(snapshot *storage.ReportSnapshot, numDeployedImageCVEs, numWatchedImageCVEs int) (string, error) {
	var writer strings.Builder

	err := validateSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	reportFilters := snapshot.GetVulnReportFilters()

	writer.WriteString("<div>")

	// Config name
	formatSingleDetail(&writer, "Config name", snapshot.GetName())

	// Number of CVEs found
	formatSingleDetail(&writer, "Number of CVEs found",
		fmt.Sprintf("%d in Deployed images", numDeployedImageCVEs),
		fmt.Sprintf("%d in Watched images", numWatchedImageCVEs))

	// Severities
	// create a copy because severities will be sorted in descending order (critical, important, moderate, low)
	severities := append([]storage.VulnerabilitySeverity{}, reportFilters.GetSeverities()...)
	slices.SortFunc(severities, func(s1, s2 storage.VulnerabilitySeverity) bool {
		return s1 > s2
	})
	formatSingleDetail(&writer, "CVE severity", severities...)

	// Fixability
	fixabilities := expandFixability(reportFilters.GetFixability())
	formatSingleDetail(&writer, "CVE status", fixabilities...)

	// Collection
	formatSingleDetail(&writer, "Report scope", snapshot.GetCollection())

	// Image types
	// create a copy because image types will be sorted in ascending order (deployed, watched)
	imageTypes := append([]storage.VulnerabilityReportFilters_ImageType{}, reportFilters.GetImageTypes()...)
	slices.Sort(imageTypes)
	formatSingleDetail(&writer, "Image type", imageTypes...)

	// CVEs discovered since
	formatSingleDetail(&writer, "CVEs discovered since", reportFilters.GetCvesSince())

	writer.WriteString("</div>")

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

func formatSingleDetail[T any](writer *strings.Builder, heading string, values ...T) {
	writer.WriteString("<div style=\"padding: 0 0 10px 0\">")

	// Add heading
	writer.WriteString("<span style=\"font-weight: bold; margin-right: 10px\">")
	writer.WriteString(fmt.Sprintf("%s: ", heading))
	writer.WriteString("</span>")

	// Add values
	if len(values) > 0 {
		writer.WriteString("<span>")
		for i, valI := range values {
			writer.WriteString(convertValueToFriendlyText(valI))
			if i < (len(values) - 1) {
				writer.WriteString(", ")
			}
		}
		writer.WriteString("</span>")
	}

	writer.WriteString("</div>")
}

func convertValueToFriendlyText(valI interface{}) string {
	switch val := valI.(type) {
	case string:
		return val
	case *storage.CollectionSnapshot:
		return val.GetName()
	case storage.VulnerabilitySeverity:
		return cveSeverityToText[val]
	case storage.VulnerabilityReportFilters_Fixability:
		return fixabilityToText[val]
	case storage.VulnerabilityReportFilters_ImageType:
		return imageTypeToText[val]
	case *storage.VulnerabilityReportFilters_AllVuln:
		return "All time"
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
