package printers

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopaPrinter_Print_InvalidJSONPathExpressions(t *testing.T) {
	expressions := map[string]string{
		CopaFixedVersionJSONPathExpressionKey:     "",
		CopaInstalledVersionJSONPathExpressionKey: "",
	}

	printer := NewCopaPrinter(expressions)

	err := printer.Print(nil, nil)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestCopaPrinter_Print_Success(t *testing.T) {
	obj := &testObject{
		Violations: []violation{
			{
				ID:               "first-violation",
				Name:             "first",
				InstalledVersion: "0.1.0",
				UpdateVersion:    "0.1.1",
			},
			{
				ID:               "second-violation",
				Name:             "second",
				InstalledVersion: "0.2.0",
				UpdateVersion:    "0.2.2",
			},
			{
				ID:               "third-violation",
				Name:             "third",
				InstalledVersion: "0.3.0",
				UpdateVersion:    "0.3.3",
			},
			{
				ID:               "unfixed-violation",
				Name:             "not fixed",
				InstalledVersion: "0.0.0",
				UpdateVersion:    "",
			},
		},
	}

	expressions := map[string]string{
		CopaFixedVersionJSONPathExpressionKey:     "violations.#.fixed",
		CopaInstalledVersionJSONPathExpressionKey: "violations.#.installed",
		CopaVulnerabilityIdJSONPathExpressionKey:  "violations.#.id",
		CopaNameJSONPathExpressionKey:             "violations.#.name",
	}

	out := strings.Builder{}
	expectedOutput, err := os.ReadFile(path.Join("testdata", "copa", "manifest.json"))
	require.NoError(t, err)

	printer := NewCopaPrinter(expressions)
	err = printer.Print(obj, &out)
	require.NoError(t, err)

	assert.Equal(t, string(expectedOutput), out.String())
}

func TestCopaPrinter_Print_EmptyViolations(t *testing.T) {
	obj := &testObject{Violations: nil}
	expressions := map[string]string{
		CopaFixedVersionJSONPathExpressionKey:     "",
		CopaInstalledVersionJSONPathExpressionKey: "",
		CopaVulnerabilityIdJSONPathExpressionKey:  "",
		CopaNameJSONPathExpressionKey:             "",
	}

	out := strings.Builder{}
	expectedOutput, err := os.ReadFile(path.Join("testdata", "copa", "empty.json"))
	require.NoError(t, err)

	printer := NewCopaPrinter(expressions)
	err = printer.Print(obj, &out)
	require.NoError(t, err)

	assert.Equal(t, string(expectedOutput), out.String())
}
