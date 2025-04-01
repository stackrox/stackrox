package printer

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
)

var _ CustomPrinterFactory = (*CopaPrinterFactory)(nil)

// CopaPrinterFactory holds all configuration options for a Copa printer.
type CopaPrinterFactory struct {
	// jsonPathExpressions hold all required expressions to build a copa manifest.
	// The data is currently NOT expected to be given by the user. The map itself MUST contain the keys
	jsonPathExpressions map[string]string
}

// NewCopaPrinterFactory creates a factory that is able to construct a printers.CopaPrinter from the given options.
func NewCopaPrinterFactory(jsonPathExpressions map[string]string) *CopaPrinterFactory {
	return &CopaPrinterFactory{
		jsonPathExpressions: jsonPathExpressions,
	}
}

// SupportedFormats list the supported formats of the factory.
func (s *CopaPrinterFactory) SupportedFormats() []string {
	return []string{"copa"}
}

// AddFlags adds all printer specific flags to the cobra.Command.
func (s *CopaPrinterFactory) AddFlags(_ *cobra.Command) {
}

// CreatePrinter will create a printers.CopaPrinter with the provided settings.
func (s *CopaPrinterFactory) CreatePrinter(format string) (ObjectPrinter, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "copa":
		return printers.NewCopaPrinter(s.jsonPathExpressions), nil
	default:
		return nil, errox.InvalidArgs.Newf("invalid output format used for sarif printer %q", format)
	}
}

func (s *CopaPrinterFactory) validate() error {
	return nil
}
