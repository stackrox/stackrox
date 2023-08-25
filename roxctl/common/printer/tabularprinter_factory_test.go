package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
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
