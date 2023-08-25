package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitPrinterFactory_CreatePrinter(t *testing.T) {
	cases := map[string]struct {
		j          *JUnitPrinterFactory
		shouldFail bool
		format     string
		error      error
	}{
		"should not fail and return a junit printer": {
			j: &JUnitPrinterFactory{
				suiteName: "testsuite",
				jsonPathExpressions: map[string]string{
					printers.JUnitTestCasesExpressionKey:            "test",
					printers.JUnitFailedTestCasesExpressionKey:      "test",
					printers.JUnitFailedTestCaseErrMsgExpressionKey: "test",
				},
			},
			format: "junit",
		},
		"should fail if validate fails": {
			j:          &JUnitPrinterFactory{},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should fail if output format is invalid": {
			j: &JUnitPrinterFactory{
				suiteName: "testsuite",
				jsonPathExpressions: map[string]string{
					printers.JUnitTestCasesExpressionKey:            "test",
					printers.JUnitFailedTestCasesExpressionKey:      "test",
					printers.JUnitFailedTestCaseErrMsgExpressionKey: "test",
				},
			},
			format:     "json",
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := c.j.CreatePrinter(c.format)
			if c.shouldFail {
				require.Error(t, err)
				assert.Nil(t, p)
				assert.ErrorIs(t, err, c.error)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
			}
		})
	}
}

func TestJUnitPrinterFactory_Validate(t *testing.T) {
	cases := map[string]struct {
		suiteName             string
		jsonPathExpressionMap map[string]string
		shouldFail            bool
		error                 error
	}{
		"should not return an error if suite name is set and json path map is valid": {
			suiteName: "testsuite",
			jsonPathExpressionMap: map[string]string{
				printers.JUnitTestCasesExpressionKey:            "test",
				printers.JUnitFailedTestCasesExpressionKey:      "test",
				printers.JUnitFailedTestCaseErrMsgExpressionKey: "test",
			},
		},
		"should return an invalid args error if suite name is not set": {
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should return an invariant violation error if json path map is invalid": {
			suiteName: "testsuite",
			jsonPathExpressionMap: map[string]string{
				printers.JUnitTestCasesExpressionKey: "test",
			},
			shouldFail: true,
			error:      errox.InvariantViolation,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			j := &JUnitPrinterFactory{
				suiteName:           c.suiteName,
				jsonPathExpressions: c.jsonPathExpressionMap,
			}
			err := j.validate()
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
				return
			}
			assert.NoError(t, err)
		})
	}
}
