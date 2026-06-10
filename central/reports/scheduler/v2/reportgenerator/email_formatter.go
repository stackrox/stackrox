package reportgenerator

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/timestamp"
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
	scopeName := "Custom Scope"
	if snapshot.GetCollection() != nil {
		scopeName = snapshot.GetCollection().GetName()
	}

	if len(scopeName) > maxCollectionNameLenInSubject {
		scopeName = fmt.Sprintf("%s...", scopeName[0:maxCollectionNameLenInSubject])
	}

	data := &reportEmailSubjectFormat{
		BrandedProductNameShort: branding.GetProductNameShort(),
		ReportConfigName:        configName,
		CollectionName:          scopeName,
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

	writer.WriteString("<div>")

	// Config name
	formatSingleDetail(&writer, "Config name", snapshot.GetName())

	// Number of CVEs found
	formatSingleDetail(&writer, "Number of CVEs found",
		fmt.Sprintf("%d in Deployed images", numDeployedImageCVEs),
		fmt.Sprintf("%d in Watched images", numWatchedImageCVEs))
	// Collection scope: show severity, fixability, collection, image types, CVEs since
	reportFilters := snapshot.GetVulnReportFilters()

	if entityScope := snapshot.GetResourceScope().GetEntityScope(); entityScope != nil {
		// Entity scope: show filter query and scope rules
		if query := reportFilters.GetQuery(); query != "" {
			formatSingleDetail(&writer, "Filter", query)
		}
		scopeParts := formatEntityScope(entityScope)
		if len(scopeParts) > 0 {
			formatSingleDetail(&writer, "Report scope", scopeParts...)
		}
	} else {

		// Severities
		severities := append([]storage.VulnerabilitySeverity{}, reportFilters.GetSeverities()...)
		sort.Slice(severities, func(i, j int) bool {
			return severities[i] > severities[j]
		})
		formatSingleDetail(&writer, "CVE severity", severities...)

		// Fixability
		fixabilities := expandFixability(reportFilters.GetFixability())
		formatSingleDetail(&writer, "CVE status", fixabilities...)

		// Collection
		formatSingleDetail(&writer, "Report scope", snapshot.GetCollection())
	}
	// Image types
	imageTypes := append([]storage.VulnerabilityReportFilters_ImageType{}, reportFilters.GetImageTypes()...)
	sliceutils.NaturalSort(imageTypes)
	formatSingleDetail(&writer, "Image type", imageTypes...)

	// CVEs discovered since
	formatSingleDetail(&writer, "CVEs discovered since", reportFilters.GetCvesSince())

	writer.WriteString("</div>")

	return writer.String(), nil
}

func formatEntityScope(entityScope *storage.EntityScope) []string {
	var parts []string
	for _, rule := range entityScope.GetRules() {
		entityType := friendlyEntityType(rule.GetEntity())
		field := friendlyEntityField(rule.GetField())
		values := make([]string, 0, len(rule.GetValues()))
		for _, v := range rule.GetValues() {
			values = append(values, v.GetValue())
		}
		parts = append(parts, fmt.Sprintf("%s %s: %s", entityType, field, strings.Join(values, ", ")))
	}
	return parts
}

var entityTypeToText = map[storage.EntityType]string{
	storage.EntityType_ENTITY_TYPE_DEPLOYMENT: "Deployment",
	storage.EntityType_ENTITY_TYPE_NAMESPACE:  "Namespace",
	storage.EntityType_ENTITY_TYPE_CLUSTER:    "Cluster",
}

var entityFieldToText = map[storage.EntityField]string{
	storage.EntityField_FIELD_ID:         "ID",
	storage.EntityField_FIELD_NAME:       "Name",
	storage.EntityField_FIELD_LABEL:      "Label",
	storage.EntityField_FIELD_ANNOTATION: "Annotation",
}

func friendlyEntityType(t storage.EntityType) string {
	if text, ok := entityTypeToText[t]; ok {
		return text
	}
	return t.String()
}

func friendlyEntityField(f storage.EntityField) string {
	if text, ok := entityFieldToText[f]; ok {
		return text
	}
	return f.String()
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

func convertValueToFriendlyText(valI any) string {
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

	hasCollection := snapshot.GetCollection() != nil
	hasEntityScope := snapshot.GetResourceScope().GetEntityScope() != nil
	if !hasCollection && !hasEntityScope {
		return errors.New("Report snapshot is missing both collection snapshot and entity scope")
	}
	if len(reportFilters.GetImageTypes()) == 0 {
		return errors.New("Report snapshot is missing image type filters")
	}
	if reportFilters.GetCvesSince() == nil {
		return errors.New("Report snapshot is missing 'CVEs since' filter")
	}
	return nil
}
