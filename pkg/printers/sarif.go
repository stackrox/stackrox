package printers

import (
	"io"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

// Required JSON path expressions for the sarif printer.
const (
	SarifRuleJSONPathExpressionKey     string = "rule-id"
	SarifHelpJSONPathExpressionKey     string = "help"
	SarifSeverityJSONPathExpressionKey string = "severity"
	SarifHelpLinkJSONPathExpressionKey string = "help-link"
)

// Supported reports for the sarif printer. This will be converted into a rule name, with which you can then query
// within the report for "types" of violations.
const (
	SarifVulnerabilityReport = "Vulnerabilities"
	SarifPolicyReport        = "PolicyViolations"
)

var requiredKeys = []string{
	SarifRuleJSONPathExpressionKey,
	SarifHelpJSONPathExpressionKey,
	SarifSeverityJSONPathExpressionKey,
}

// SarifPrinter is capable of printing sarif reports from JSON objects, retrieving all relevant data for the report via
// JSON path expressions.
type SarifPrinter struct {
	jsonPathExpressions map[string]string
	entity              string
	reportType          string
}

type sarifEntry struct {
	ruleID   string
	help     string
	severity string
	helpLink string
}

// NewSarifPrinter creates a printer capable of printing a sarif report.
// (https://docs.github.com/en/code-security/code-scanning/integrating-with-code-scanning/sarif-support-for-code-scanning#about-sarif-support)
// A SarifPrinter expects a JSON Object and a map of JSON path expressions that are compatible with
// GJOSN (https://github.com/tidwall/gjson).
//
// When printing, the SarifPrinter will take the given JSON object and apply the JSON path expressions within the map
// to retrieve all required data to generate a sarif report.
//
// The given JSON object itself MUST be passable to json.Marshal, so it CAN NOT be a direct JSON input.
// For the structure of the JSON object, it is preferred to have arrays of structs instead of
// array of elements, since structs will provide default values if a field is missing.
//
// The map of JSON path expressions given MUST contain JSON path expressions for the keys:
//   - SarifRuleJSONPathExpressionKey, yields the rule ID to use in the sarif report (e.g. CVE-XYZ-component-version).
//   - SarifHelpJSONPathExpressionKey, yields the help text to use in the sarif report. This should include remediation steps.
//   - SarifSeverityJSONPathExpressionKey, yields the severity to use in the sarif report.
//
// Optionally, you MAY specify the JSON path expression SarifHelpLinkJSONPathExpressionKey in case your data contains a valid
// link for the reported violation (e.g. NVD CVE page).
//
// The values yielded from each JSON path expressions MUST be equal to one another, as each set of values will be used
// to construct the report entries (result and rule).
//
// The GJSON expression syntax (https://github.com/tidwall/gjson/blob/master/SYNTAX.md) offers more complex
// and advanced scenarios, if you require them and the below example is not sufficient.
// Additionally, there are custom GJSON modifiers, which will post-process expression results. Currently,
// the gjson.ListModifier, gjson.TextModifier and gjson.BoolReplaceModifier are available, see their documentation on
// usage and GJSON's syntax expression to read more about modifiers.
//
// The following example illustrates a JSON compatible structure and an example for the map of JSON path expressions
// to generate a sarif report.
//
// JSON structure:
//
//		type data struct {
//					Violations []violation `json:"violations`
//		}
//
//		type violation struct {
//		      Id          string `json:"id"`
//	          Reason      string `json:"reason"`
//	          Severity    string `json:"severity"`
//		}
//
// Example:
//
//	 expressions := map[string] {
//			SarifRuleJSONPathExpressionKey: "violations.#.id",
//		    SarifHelpJSONPathExpressionKey: "violations.#.reason",
//		    SarifSeverityJSONPathExpressionKey: "violations.#.severity",
//		}
//
// For an example sarif report, see testdata/sarif_report.json.
//
// Advanced usages:
// For constructing multiline help messages with values from different JSON fields, see roxctl/image/check/check.go
// as an example usage of constructing 1) a rule ID with multiple values using gjson.TextModifier and 2) creating the
// multiline help text via gjson.TextModifier.
func NewSarifPrinter(jsonPathExpressions map[string]string, entity string, reportType string) *SarifPrinter {
	return &SarifPrinter{jsonPathExpressions: jsonPathExpressions, entity: entity, reportType: reportType}
}

// Print will create a sarif report from the given object and write the output to the given io.Writer.
func (s *SarifPrinter) Print(object interface{}, out io.Writer) error {
	sarifEntries, err := sarifEntriesFromJSONObject(object, s.jsonPathExpressions)
	if err != nil {
		return err
	}

	run := sarif.NewRunWithInformationURI("roxctl", "https://github.com/stackrox/stackrox")
	// Set the version information and the full name on the tool driver.
	run.Tool.Driver.
		WithVersion(version.GetMainVersion()).
		WithFullName("roxctl command line utility")

	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return errors.Wrap(err, "creating sarif report")
	}

	for _, entry := range sarifEntries {
		s.addEntry(run, entry)
	}

	report.AddRun(run)
	return report.PrettyWrite(out)
}

func (s *SarifPrinter) addEntry(run *sarif.Run, entry sarifEntry) {
	rule := run.AddRule(entry.ruleID)
	rule.WithName(s.reportType).
		// Setting the description is also setting the title displayed in GitHub.
		WithShortDescription(sarif.NewMultiformatMessageString(entry.ruleID)).
		WithFullDescription(sarif.NewMultiformatMessageString(entry.ruleID)).
		WithHelp(sarif.NewMultiformatMessageString(entry.help))

	if entry.helpLink != "" {
		rule.WithHelpURI(entry.helpLink)
	}

	properties := sarif.Properties{
		// Precision very-high ensures this violation is shown first within GitHub.
		"precision": "very-high",
		// Tags allow filtering, which is desirable to have.
		"tags": []string{
			utils.IfThenElse(s.reportType == SarifVulnerabilityReport, "security", "policy-violation"),
			entry.severity,
		},
	}
	// For vulnerability reports, generated the security severity based of the severity reported.
	if s.reportType == SarifVulnerabilityReport {
		properties["security-severity"] = toSecuritySeverity(entry.severity)
	}
	rule.WithProperties(properties)

	run.AddResult(sarif.NewRuleResult(entry.ruleID).
		WithLevel(toSarifLevel(entry.severity)).
		// Reusing the help here, since the help includes remediation information.
		WithMessage(sarif.NewMessage().WithText(entry.help)).
		WithLocations([]*sarif.Location{
			{
				PhysicalLocation: &sarif.PhysicalLocation{
					ArtifactLocation: sarif.NewArtifactLocation().WithUri(s.entity),
					Region:           sarif.NewSimpleRegion(1, 1),
				},
			},
		}))
}

func sarifEntriesFromJSONObject(jsonObject interface{}, pathExpressions map[string]string) ([]sarifEntry, error) {
	pathExpr := set.NewStringSet(maputil.Keys(pathExpressions)...)
	for _, requiredKey := range requiredKeys {
		if !pathExpr.Contains(requiredKey) {
			return nil, errox.InvalidArgs.Newf("not all required JSON path expressions given, ensure JSON "+
				"path expression are given for: [%s]", strings.Join(requiredKeys, ","))
		}
	}

	sliceMapper, err := gjson.NewSliceMapper(jsonObject, pathExpressions)
	if err != nil {
		return nil, err
	}
	data := sliceMapper.CreateSlices()

	numberOfValues := len(data[SarifRuleJSONPathExpressionKey])
	for key, values := range data {
		if len(values) != numberOfValues {
			return nil, errox.InvalidArgs.Newf("the amount of values retrieved from JSON path expressions "+
				"should be %d, but got %d for key %s", numberOfValues, len(values), key)
		}
	}

	sarifEntries := make([]sarifEntry, 0, numberOfValues)
	for i := 0; i < numberOfValues; i++ {
		entry := sarifEntry{
			ruleID:   data[SarifRuleJSONPathExpressionKey][i],
			help:     data[SarifHelpJSONPathExpressionKey][i],
			severity: data[SarifSeverityJSONPathExpressionKey][i],
		}
		if len(data[SarifHelpLinkJSONPathExpressionKey]) > 0 {
			entry.helpLink = data[SarifHelpLinkJSONPathExpressionKey][i]
		}
		sarifEntries = append(sarifEntries, entry)
	}
	return sarifEntries, nil
}

// All our supported severities. We have different severities for policy violations and CVE violations, and the sarif
// report printer shall be capable of handling both.
var (
	criticalVulnSeverity  = strings.TrimSuffix(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(), "_VULNERABILITY_SEVERITY")
	importantVulnSeverity = strings.TrimSuffix(storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY.String(), "_VULNERABILITY_SEVERITY")
	moderateVulnSeverity  = strings.TrimSuffix(storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(), "_VULNERABILITY_SEVERITY")
	lowVulnSeverity       = strings.TrimSuffix(storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(), "_VULNERABILITY_SEVERITY")

	criticalPolicySeverity = strings.TrimSuffix(storage.Severity_CRITICAL_SEVERITY.String(), "_SEVERITY")
	highPolicySeverity     = strings.TrimSuffix(storage.Severity_HIGH_SEVERITY.String(), "_SEVERITY")
	mediumPolicySeverity   = strings.TrimSuffix(storage.Severity_MEDIUM_SEVERITY.String(), "_SEVERITY")
	lowPolicySeverity      = strings.TrimSuffix(storage.Severity_LOW_SEVERITY.String(), "_SEVERITY")
)

func toSarifLevel(severity string) string {
	// While this shouldn't be the case, let's be on the safe side and strip the enum suffix.
	severity = strings.TrimSuffix(severity, "_VULNERABILITY_SEVERITY")
	severity = strings.TrimSuffix(severity, "_SEVERITY")

	switch severity {
	case criticalVulnSeverity, criticalPolicySeverity, importantVulnSeverity, highPolicySeverity:
		return "error"
	case moderateVulnSeverity, mediumPolicySeverity:
		return "warning"
	case lowVulnSeverity, lowPolicySeverity:
		return "note"
	default:
		return "none"
	}
}

func toSecuritySeverity(severity string) string {
	// While this shouldn't be the case, let's be on the safe side and strip the enum suffix.
	severity = strings.TrimSuffix(severity, "_VULNERABILITY_SEVERITY")
	severity = strings.TrimSuffix(severity, "_SEVERITY")
	switch severity {
	case criticalVulnSeverity, criticalPolicySeverity:
		return "9.1"
	case importantVulnSeverity, highPolicySeverity:
		return "7.9"
	case moderateVulnSeverity, mediumPolicySeverity:
		return "4.8"
	case lowVulnSeverity, lowPolicySeverity:
		return "3.3"
	default:
		return "0.0"
	}
}
