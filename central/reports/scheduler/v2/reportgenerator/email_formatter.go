package reportgenerator

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
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

	// UI routes for the "View report in console" link. Kept in sync with
	// ui/apps/platform/src/routePaths.ts.
	workloadReportUIPath = "/main/vulnerabilities/reports/images/configurations/%s"
	nodeReportUIPath     = "/main/vulnerabilities/reports/nodes/configurations/%s"
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

// FormatEmailSubject formats an email subject line from the given Go template
// and report snapshot.
func FormatEmailSubject(subjectTemplate string, snapshot *storage.ReportSnapshot) (string, error) {
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

// FormatEmailBody formats an email body (the introductory paragraph) from the
// given Go template.
func FormatEmailBody(emailTemplate string) (string, error) {
	data := &reportEmailBodyFormat{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
	}

	tmpl, err := template.New("emailBody").Parse(emailTemplate)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

// FormatWorkloadReportEmailBody builds the full HTML body of a workload (image)
// CVE report notification email. introHTML is the introductory text (from the
// body template or a notifier's custom body) and reportURL, when non-empty,
// renders the "View report in console" button.
func FormatWorkloadReportEmailBody(introHTML string, snapshot *storage.ReportSnapshot,
	numDeployedImageCVEs, numWatchedImageCVEs int, hasAttachment bool, reportURL string) (string, error) {
	if err := validateSnapshot(snapshot); err != nil {
		return "", err
	}
	reportFilters := snapshot.GetVulnReportFilters()

	layout := emailLayout{
		title:         "Workload CVE Report",
		subtitle:      workloadSubtitle(snapshot),
		introHTML:     introHTML,
		hasAttachment: hasAttachment,
		reportURL:     reportURL,
	}

	if numDeployedImageCVEs == 0 && numWatchedImageCVEs == 0 {
		layout.noVulnsMsg = "No workload CVEs found for this configuration."
	} else {
		layout.statCards = []statCard{
			{value: humanizeInt(numDeployedImageCVEs), label: "CVEs · Deployed images"},
			{value: humanizeInt(numWatchedImageCVEs), label: "CVEs · Watched images"},
		}
	}

	rows := []detailRow{
		{label: "Config name", value: snapshot.GetName(), bold: true},
	}

	if entityScope := snapshot.GetResourceScope().GetEntityScope(); entityScope != nil {
		// Entity (custom) scope: severity and fixability are encoded in the raw
		// filter query, so there is no severity legend or CVE status row.
		if query := reportFilters.GetQuery(); query != "" {
			rows = append(rows, detailRow{label: "Filter", value: query, mono: true})
		}
		if scopeParts := formatEntityScope(entityScope); len(scopeParts) > 0 {
			rows = append(rows, detailRow{label: "Report scope", value: strings.Join(scopeParts, ", ")})
		}
	} else {
		// Collection scope: severities render as a legend; fixability and the
		// collection name go in the details table.
		severities := append([]storage.VulnerabilitySeverity{}, reportFilters.GetSeverities()...)
		sort.Slice(severities, func(i, j int) bool {
			return severities[i] > severities[j]
		})
		layout.severities = severities

		rows = append(rows, detailRow{label: "CVE status", value: joinFriendly(expandFixability(reportFilters.GetFixability())...)})
		rows = append(rows, detailRow{label: "Report scope", value: snapshot.GetCollection().GetName()})
	}

	imageTypes := append([]storage.VulnerabilityReportFilters_ImageType{}, reportFilters.GetImageTypes()...)
	sliceutils.NaturalSort(imageTypes)
	rows = append(rows, detailRow{label: "Image type", value: joinFriendly(imageTypes...)})

	rows = append(rows, detailRow{label: "CVEs discovered in image since", value: convertValueToFriendlyText(reportFilters.GetCvesSince())})

	layout.detailRows = rows
	return layout.render(), nil
}

// FormatNodeReportEmailBody builds the full HTML body of a node CVE report
// notification email.
func FormatNodeReportEmailBody(introHTML string, snapshot *storage.ReportSnapshot,
	numNodeCVEs int, hasAttachment bool, reportURL string) (string, error) {
	filters := snapshot.GetNodeVulnReportFilters()
	if filters == nil {
		return "", errors.New("report snapshot is missing node vulnerability report filters")
	}

	layout := emailLayout{
		title:         "Node CVE Report",
		subtitle:      snapshot.GetName(),
		introHTML:     introHTML,
		hasAttachment: hasAttachment,
		reportURL:     reportURL,
	}

	if numNodeCVEs == 0 {
		layout.noVulnsMsg = "No node CVEs found for this configuration."
	} else {
		layout.statCards = []statCard{
			{value: humanizeInt(numNodeCVEs), label: "Node CVEs found"},
		}
	}

	rows := []detailRow{
		{label: "Config name", value: snapshot.GetName(), bold: true},
	}
	if query := filters.GetQuery(); query != "" {
		rows = append(rows, detailRow{label: "Filter", value: query, mono: true})
	}
	if entityScope := snapshot.GetResourceScope().GetEntityScope(); entityScope != nil {
		if scopeParts := formatEntityScope(entityScope); len(scopeParts) > 0 {
			rows = append(rows, detailRow{label: "Report scope", value: strings.Join(scopeParts, ", ")})
		}
	}

	layout.detailRows = rows
	return layout.render(), nil
}

// BuildWorkloadReportURL builds a link to the workload report configuration page
// in the console. It returns an empty string (button omitted) when the endpoint
// or config ID is missing or the endpoint is unparseable.
func BuildWorkloadReportURL(uiEndpoint, reportConfigID string) string {
	return buildReportURL(uiEndpoint, reportConfigID, workloadReportUIPath)
}

// BuildNodeReportURL builds a link to the node report configuration page in the
// console.
func BuildNodeReportURL(uiEndpoint, reportConfigID string) string {
	return buildReportURL(uiEndpoint, reportConfigID, nodeReportUIPath)
}

func buildReportURL(uiEndpoint, reportConfigID, pathFormat string) string {
	if uiEndpoint == "" || reportConfigID == "" {
		return ""
	}
	base, err := url.Parse(uiEndpoint)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(fmt.Sprintf(pathFormat, reportConfigID))
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

func workloadSubtitle(snapshot *storage.ReportSnapshot) string {
	scope := "Custom Scope"
	if snapshot.GetCollection() != nil {
		scope = snapshot.GetCollection().GetName()
	}
	return fmt.Sprintf("%s • Scope: %s", snapshot.GetName(), scope)
}

// humanizeInt formats an integer with thousands separators, e.g. 6298 -> "6,298".
func humanizeInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}

	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func formatEntityScope(entityScope *storage.EntityScope) []string {
	parts := make([]string, 0, len(entityScope.GetRules()))
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

// joinFriendly maps each value to its human-friendly text and joins them with ", ".
func joinFriendly[T any](values ...T) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, convertValueToFriendlyText(v))
	}
	return strings.Join(parts, ", ")
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
