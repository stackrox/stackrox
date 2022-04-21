package printer

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObjectPrinterFactory(t *testing.T) {
	cases := map[string]struct {
		defaultFormat  string
		shouldFail     bool
		error          error
		printerFactory []CustomPrinterFactory
	}{
		"should fail when no CustomPrinterFactory is added": {
			defaultFormat:  "table",
			shouldFail:     true,
			error:          errox.InvariantViolation,
			printerFactory: []CustomPrinterFactory{nil},
		},
		"should not fail if format is supported by registered CustomPrinterFactory": {
			defaultFormat:  "table",
			printerFactory: []CustomPrinterFactory{NewTabularPrinterFactory(nil, "")},
		},
		"should not fail if format is supported and valid values for CustomPrinterFactory": {
			defaultFormat:  "table",
			printerFactory: []CustomPrinterFactory{NewTabularPrinterFactory([]string{"a", "b"}, "a,b")},
		},
		"should fail if default output format is not supported by registered CustomPrinterFactory": {
			defaultFormat:  "table",
			shouldFail:     true,
			error:          errox.InvalidArgs,
			printerFactory: []CustomPrinterFactory{NewJSONPrinterFactory(false, false)},
		},
		"should fail if duplicate CustomPrinterFactory is being registered": {
			defaultFormat:  "json",
			shouldFail:     true,
			error:          errox.InvariantViolation,
			printerFactory: []CustomPrinterFactory{NewJSONPrinterFactory(false, false), NewJSONPrinterFactory(false, false)},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewObjectPrinterFactory(c.defaultFormat, c.printerFactory...)
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestObjectPrinterFactory_AddFlags(t *testing.T) {
	o := ObjectPrinterFactory{
		OutputFormat: "table",
		RegisteredPrinterFactories: map[string]CustomPrinterFactory{
			"json":      NewJSONPrinterFactory(false, false),
			"table,csv": NewTabularPrinterFactory(nil, ""),
		},
	}
	cmd := &cobra.Command{
		Use: "test",
	}
	o.AddFlags(cmd)
	formatFlag := cmd.Flag("output")
	require.NotNil(t, formatFlag)
	assert.Equal(t, "o", formatFlag.Shorthand)
	assert.Equal(t, "table", formatFlag.DefValue)
	assert.True(t, strings.Contains(formatFlag.Usage, "json"))
	assert.True(t, strings.Contains(formatFlag.Usage, "table"))
	assert.True(t, strings.Contains(formatFlag.Usage, "csv"))
}

func TestObjectPrinterFactory_validateOutputFormat(t *testing.T) {
	cases := map[string]struct {
		o          ObjectPrinterFactory
		shouldFail bool
		error      error
	}{
		"should not return an error when output format is supported": {
			o: ObjectPrinterFactory{
				OutputFormat: "table",
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"table,csv": NewTabularPrinterFactory(nil, ""),
					"json":      NewJSONPrinterFactory(false, false),
				},
			},
			shouldFail: false,
		},
		"should return an error when output format is not supported": {
			o: ObjectPrinterFactory{
				OutputFormat: "junit",
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"table,csv": NewTabularPrinterFactory(nil, ""),
					"json":      NewJSONPrinterFactory(false, false),
				},
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.o.validateOutputFormat()
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestObjectPrinterFactory_IsStandardizedFormat(t *testing.T) {
	cases := map[string]struct {
		res    bool
		format string
	}{
		"should be true for JSON format": {
			res:    true,
			format: "json",
		},
		"should be true for CSV format": {
			res:    true,
			format: "csv",
		},
		"should be false for table format": {
			res:    false,
			format: "table",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			o := ObjectPrinterFactory{OutputFormat: c.format}
			assert.Equal(t, c.res, o.IsStandardizedFormat())
		})
	}
}

func TestObjectPrinterFactory_CreatePrinter(t *testing.T) {
	cases := map[string]struct {
		o     ObjectPrinterFactory
		error error
	}{
		"should return an error when the output format is not supported": {
			o: ObjectPrinterFactory{
				OutputFormat: "table",
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"json": NewJSONPrinterFactory(false, false),
				},
			},
			error: errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			printer, err := c.o.CreatePrinter()
			require.Error(t, err)
			assert.ErrorIs(t, err, c.error)
			assert.Nil(t, printer)
		})
	}
}

func TestObjectPrinterFactory_validate(t *testing.T) {
	cases := map[string]struct {
		o          ObjectPrinterFactory
		shouldFail bool
		error      error
	}{
		"should not fail with valid CustomPrinterFactory and valid output format": {
			o: ObjectPrinterFactory{
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"json": NewJSONPrinterFactory(false, false),
				},
				OutputFormat: "json",
			},
		},
		"should fail with invalid CustomPrinterFactory": {
			o: ObjectPrinterFactory{
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"table": &TabularPrinterFactory{
						Headers:               []string{"a", "b"},
						RowJSONPathExpression: "a",
						NoHeader:              true,
						HeaderAsComment:       true,
					},
				},
				OutputFormat: "table",
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
		"should fail with unsupported OutputFormat": {
			o: ObjectPrinterFactory{
				RegisteredPrinterFactories: map[string]CustomPrinterFactory{
					"json": NewJSONPrinterFactory(false, false),
				},
				OutputFormat: "table",
			},
			shouldFail: true,
			error:      errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.o.validate()
			if c.shouldFail {
				require.Error(t, err)
				assert.ErrorIs(t, err, c.error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
