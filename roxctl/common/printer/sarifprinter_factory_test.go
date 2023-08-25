package printer

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stretchr/testify/assert"
)

func TestSarifPrinterFactory_CreatePrinter(t *testing.T) {
	entityName := "entity"
	var empty string
	cases := map[string]struct {
		factory *SarifPrinterFactory
		err     error
		format  string
	}{
		"with valid values the printer should be created": {
			factory: &SarifPrinterFactory{
				jsonPathExpressions: map[string]string{},
				entity:              &entityName,
				reportType:          printers.SarifVulnerabilityReport,
			},
			format: "sarif",
		},
		"fail with empty entity": {
			factory: &SarifPrinterFactory{
				jsonPathExpressions: map[string]string{},
				entity:              &empty,
				reportType:          printers.SarifVulnerabilityReport,
			},
			format: "sarif",
			err:    errox.InvalidArgs,
		},
		"fail with invalid report type": {
			factory: &SarifPrinterFactory{
				jsonPathExpressions: map[string]string{},
				entity:              &entityName,
				reportType:          "custom report",
			},
			format: "sarif",
			err:    errox.InvariantViolation,
		},
		"fail with invalid format": {
			factory: &SarifPrinterFactory{
				jsonPathExpressions: map[string]string{},
				entity:              &entityName,
				reportType:          printers.SarifVulnerabilityReport,
			},
			format: "junit",
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
				_, ok := printer.(*printers.SarifPrinter)
				assert.True(t, ok)
			}
		})
	}
}
