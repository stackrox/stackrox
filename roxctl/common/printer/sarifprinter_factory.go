package printer

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
)

var _ CustomPrinterFactory = (*SarifPrinterFactory)(nil)

// SarifPrinterFactory holds all configuration options for a Sarif printer.
type SarifPrinterFactory struct {
	// jsonPathExpressions hold all required expressions to build a sarif report.
	// The data is currently NOT expected to be given by the user. The map itself MUST contain the keys
	jsonPathExpressions map[string]string

	// The entity for which the report is created. This can either be a container image, or a Kubernetes YAML file.
	entity *string

	// The type of report that should be created.
	// This is currently NOT expected to be given by the user, but rather be set by the command using the format.
	// The values can be either printers.SarifVulnerabilityReport or printers.SarifPolicyReport.
	reportType string
}

// NewSarifPrinterFactory creates a factory that is able to construct a printers.SarifPrinter from the given options.
func NewSarifPrinterFactory(reportType string, jsonPathExpressions map[string]string, entity *string) *SarifPrinterFactory {
	return &SarifPrinterFactory{
		jsonPathExpressions: jsonPathExpressions,
		entity:              entity,
		reportType:          reportType,
	}
}

// SupportedFormats list the supported formats of the factory.
func (s *SarifPrinterFactory) SupportedFormats() []string {
	return []string{"sarif"}
}

// AddFlags adds all printer specific flags to the cobra.Command.
func (s *SarifPrinterFactory) AddFlags(_ *cobra.Command) {
}

// CreatePrinter will create a printers.SarifPrinter with the provided settings.
func (s *SarifPrinterFactory) CreatePrinter(format string) (ObjectPrinter, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}

	if *s.entity == "" {
		return nil, errox.InvalidArgs.New("empty entity name given, please provide a name")
	}

	switch strings.ToLower(format) {
	case "sarif":
		return printers.NewSarifPrinter(s.jsonPathExpressions, *s.entity, s.reportType), nil
	default:
		return nil, errox.InvalidArgs.Newf("invalid output format used for sarif printer %q", format)
	}
}

func (s *SarifPrinterFactory) validate() error {
	if s.reportType != printers.SarifVulnerabilityReport && s.reportType != printers.SarifPolicyReport {
		return errox.InvariantViolation.Newf("report type must be either %s or %s, but was %s",
			printers.SarifVulnerabilityReport, printers.SarifPolicyReport, s.reportType)
	}

	return nil
}
