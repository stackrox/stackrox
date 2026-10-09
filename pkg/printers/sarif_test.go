package printers

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/owenrumney/go-sarif/v3/pkg/report/v210/sarif"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testObject struct {
	Violations []violation `json:"violations"`
}

type violation struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	Severity    string `json:"severity"`
}

func TestSarifPrinter_Print_InvalidJSONPathExpressions(t *testing.T) {
	expressions := map[string]string{
		SarifRuleJSONPathExpressionKey: "",
		SarifHelpJSONPathExpressionKey: "",
	}

	printer := NewSarifPrinter(expressions, "", "")

	err := printer.Print(nil, nil)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestSarifPrinter_Print_Success(t *testing.T) {
	obj := &testObject{
		Violations: []violation{
			{
				ID:          "first-violation",
				Description: "something about violation one",
				Reason:      "something about misconfiguration",
				Severity:    "IMPORTANT",
			},
			{
				ID:          "second-violation",
				Description: "something about violation two",
				Reason:      "something about vulnerabilities",
				Severity:    "LOW",
			},
			{
				ID:          "third-violation",
				Description: "something about violation three",
				Reason:      "something about secrets",
				Severity:    "CRITICAL",
			},
		},
	}

	expressions := map[string]string{
		SarifRuleJSONPathExpressionKey:     "violations.#.id",
		SarifHelpJSONPathExpressionKey:     "violations.#.reason",
		SarifSeverityJSONPathExpressionKey: "violations.#.severity",
	}

	out := strings.Builder{}
	expectedOutput, err := os.ReadFile(path.Join("testdata", "sarif_report.json"))
	require.NoError(t, err)

	printer := NewSarifPrinter(expressions, "docker.io/nginx:1.19", SarifPolicyReport)
	err = printer.Print(obj, &out)
	require.NoError(t, err)

	// Strict parsing preserves null arrays so validation checks the emitted output.
	report, err := sarif.FromString(out.String(), sarif.WithStrictValidation())
	require.NoError(t, err)
	assert.NoError(t, report.Validate())

	// Since the report contains the version, replace it specifically here.
	exp, err := regexp.Compile(fmt.Sprintf(`"version": "%s"`, version.GetMainVersion()))
	require.NoError(t, err)
	output := exp.ReplaceAllString(out.String(), `"version": ""`)
	assert.JSONEq(t, string(expectedOutput), output)
}

func TestSarifPrinter_Print_RuleIndexes(t *testing.T) {
	for name, tc := range map[string]struct {
		ruleIDs []string
		indexes []int
	}{
		"distinct rules": {
			ruleIDs: []string{"A", "B", "C"},
			indexes: []int{0, 1, 2},
		},
		"repeated rule": {
			ruleIDs: []string{"A", "B", "A", "C"},
			indexes: []int{0, 1, 0, 2},
		},
	} {
		t.Run(name, func(t *testing.T) {
			obj := &testObject{}
			for _, ruleID := range tc.ruleIDs {
				obj.Violations = append(obj.Violations, violation{
					ID:       ruleID,
					Reason:   "remediation",
					Severity: "IMPORTANT",
				})
			}
			printer := NewSarifPrinter(map[string]string{
				SarifRuleJSONPathExpressionKey:     "violations.#.id",
				SarifHelpJSONPathExpressionKey:     "violations.#.reason",
				SarifSeverityJSONPathExpressionKey: "violations.#.severity",
			}, "docker.io/nginx:1.19", SarifPolicyReport)
			var out strings.Builder
			require.NoError(t, printer.Print(obj, &out))
			report, err := sarif.FromString(out.String(), sarif.WithStrictValidation())
			require.NoError(t, err)
			assert.NoError(t, report.Validate())
			require.Len(t, report.Runs, 1)
			run := report.Runs[0]
			require.Len(t, run.Tool.Driver.Rules, 3)
			require.Len(t, run.Results, len(tc.ruleIDs))
			for i, result := range run.Results {
				assert.Equal(t, tc.indexes[i], result.RuleIndex)
				require.GreaterOrEqual(t, result.RuleIndex, 0)
				require.Less(t, result.RuleIndex, len(run.Tool.Driver.Rules))
				assert.Equal(t, result.RuleID, run.Tool.Driver.Rules[result.RuleIndex].ID)
			}
		})
	}
}

func TestSarifPrinter_Print_EmptyViolations(t *testing.T) {
	obj := &testObject{Violations: nil}
	expressions := map[string]string{
		SarifRuleJSONPathExpressionKey:     "{violations.#.id}.@text",
		SarifHelpJSONPathExpressionKey:     "violations.#.reason",
		SarifSeverityJSONPathExpressionKey: "violations.#.severity",
	}

	out := strings.Builder{}
	expectedOutput, err := os.ReadFile(path.Join("testdata", "empty_sarif_report.json"))
	require.NoError(t, err)

	printer := NewSarifPrinter(expressions, "docker.io/nginx:1.19", SarifPolicyReport)
	err = printer.Print(obj, &out)
	require.NoError(t, err)

	// Strict parsing preserves null arrays so validation checks the emitted output.
	report, err := sarif.FromString(out.String(), sarif.WithStrictValidation())
	require.NoError(t, err)
	assert.NoError(t, report.Validate())

	// Since the report contains the version, replace it specifically here.
	exp, err := regexp.Compile(fmt.Sprintf(`"version": "%s"`, version.GetMainVersion()))
	require.NoError(t, err)
	output := exp.ReplaceAllString(out.String(), `"version": ""`)
	assert.JSONEq(t, string(expectedOutput), output)
}
