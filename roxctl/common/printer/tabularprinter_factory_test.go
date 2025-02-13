package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTabularPrinterFactory_CreatePrinter(t *testing.T) {
	cases := map[string]struct {
		t          *TabularPrinterFactory
		format     string
		shouldFail bool
		error      error
	}{
		"should not fail with valid factory and format": {
			t:      &TabularPrinterFactory{},
			format: "csv",
		},
		"should fail with invalid factory": {
			t:          &TabularPrinterFactory{HeaderAsComment: true, NoHeader: true},
			format:     "csv",
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should fail with invalid format": {
			t:          &TabularPrinterFactory{},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := c.t.CreatePrinter(c.format)
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

func TestTabularPrinterFactory_Validate(t *testing.T) {
	cases := map[string]struct {
		t          *TabularPrinterFactory
		shouldFail bool
		error      error
	}{
		"should not fail with empty headers and json path expressions": {
			t: &TabularPrinterFactory{},
		},
		"should fail with no header and header as comment set": {
			t: &TabularPrinterFactory{
				NoHeader:        true,
				HeaderAsComment: true,
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should fail with columns to merge not matching header": {
			t: &TabularPrinterFactory{
				Headers:        []string{"a", "b", "c"},
				columnsToMerge: []string{"a", "d", "c", "e"},
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.t.validate()
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTabularPrinterFactory_CustomHeaderValidation(t *testing.T) {
	cases := map[string]struct {
		t          *TabularPrinterFactory
		shouldFail bool
		error      error
	}{
		"should not fail with default values": {
			t: &TabularPrinterFactory{},
		},
		"should fail with invalid headers": {
			t: &TabularPrinterFactory{
				NoHeader:        true,
				HeaderAsComment: true,
				Headers:         []string{"FOO", "BAR"},
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should not fail with reordered allowed headers": {
			t: &TabularPrinterFactory{
				NoHeader:        true,
				HeaderAsComment: true,
				Headers:         []string{"LINK", "SEVERITY", "VERSION"},
			},
			shouldFail: false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.t.propagateCustomHeaders()
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTabularPrinterFactory_CustomHeaderPropagation(t *testing.T) {
	cases := map[string]struct {
		t                          *TabularPrinterFactory
		expectedJSONPathExpression string
	}{
		"Default": {
			t: &TabularPrinterFactory{
				Headers: defaultImageScanHeaders,
			},
			expectedJSONPathExpression: "{" +
				"result.vulnerabilities.#.componentName," +
				"result.vulnerabilities.#.componentVersion," +
				"result.vulnerabilities.#.cveId," +
				"result.vulnerabilities.#.cveSeverity," +
				"result.vulnerabilities.#.cveInfo," +
				"result.vulnerabilities.#.componentFixedVersion}",
		},
		"Reversed": {
			t: &TabularPrinterFactory{
				Headers: sliceutils.Reversed(defaultImageScanHeaders),
			},
			expectedJSONPathExpression: "{" +
				"result.vulnerabilities.#.componentFixedVersion," +
				"result.vulnerabilities.#.cveInfo," +
				"result.vulnerabilities.#.cveSeverity," +
				"result.vulnerabilities.#.cveId," +
				"result.vulnerabilities.#.componentVersion," +
				"result.vulnerabilities.#.componentName}",
		},
		"Duplicate": {
			t: &TabularPrinterFactory{
				Headers: []string{"CVE", "CVE"},
			},
			expectedJSONPathExpression: "{" +
				"result.vulnerabilities.#.cveId," +
				"result.vulnerabilities.#.cveId}",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.t.propagateCustomHeaders()
			assert.NoError(t, err)
			assert.Equal(t, c.expectedJSONPathExpression, c.t.RowJSONPathExpression)
		})
	}
}
