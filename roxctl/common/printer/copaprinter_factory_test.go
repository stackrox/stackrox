package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stretchr/testify/assert"
)

func TestCopaPrinterFactory_CreatePrinter(t *testing.T) {
	cases := map[string]struct {
		factory *CopaPrinterFactory
		err     error
		format  string
	}{
		"with valid values the printer should be created": {
			factory: &CopaPrinterFactory{
				jsonPathExpressions: map[string]string{},
			},
			format: "copa",
		},
		"fail with invalid format": {
			factory: &CopaPrinterFactory{
				jsonPathExpressions: map[string]string{},
			},
			format: "invalid",
			err:    errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			printer, err := c.factory.CreatePrinter(c.format)
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			} else {
				assert.NoError(t, err)
				_, ok := printer.(*printers.CopaPrinter)
				assert.True(t, ok)
			}
		})
	}
}
