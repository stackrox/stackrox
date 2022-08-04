package printer

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
)

// JSONPrinterFactory holds all configuration options for the JSONPrinter.
// It is an implementation of CustomPrinterFactory and acts as a factory for JSONPrinter
type JSONPrinterFactory struct {
	Compact bool
	// EscapeHTMLCharacters will escape HTML characters when set when Marshalling JSON objects. This setting is not
	// expected to be added as flags to the cobra.Command for user input, it should be decided by the command itself
	EscapeHTMLCharacters bool
}

// NewJSONPrinterFactory creates new JSONPrinterFactory with the injected default values
func NewJSONPrinterFactory(compact bool, escapeHTML bool) *JSONPrinterFactory {
	return &JSONPrinterFactory{Compact: compact, EscapeHTMLCharacters: escapeHTML}
}

// AddFlags will add all JSONPrinter specific flags to the cobra.Command
func (j *JSONPrinterFactory) AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&j.Compact, "compact-output", j.Compact, "Print JSON output compact")
}

// SupportedFormats returns the supported printer format that can be created by JSONPrinterFactory
func (j *JSONPrinterFactory) SupportedFormats() []string {
	return []string{"json"}
}

// CreatePrinter creates a JSONPrinter from the options set. If the format is unsupported, or it is not possible
// to create an ObjectPrinter with the current configuration it will return an error
func (j *JSONPrinterFactory) CreatePrinter(format string) (ObjectPrinter, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	switch strings.ToLower(format) {
	case "json":
		return printers.NewJSONPrinter(printers.WithJSONEscapeHTML(j.EscapeHTMLCharacters),
			printers.WithJSONCompact(j.Compact)), nil
	default:
		return nil, errox.InvalidArgs.Newf("invalid output format %q used for JSON printer", format)
	}
}

// validate verifies whether the current configuration can be used to create an ObjectPrinter. It will return an error
// if it is not possible
func (j *JSONPrinterFactory) validate() error {
	return nil
}
