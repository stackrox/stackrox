package printers

import (
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
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

	// Since the report contains the version, replace it specifically here.
	exp, err := regexp.Compile(`"version": "[34].*"`)
	require.NoError(t, err)
	output := exp.ReplaceAllString(out.String(), `"version": ""`)
	assert.Equal(t, string(expectedOutput), output)
}
