package printer

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

var (
	// standardizedFormats holds all output formats that follow either an RFC standard or a de-facto standard
	standardizedFormats = set.NewFrozenStringSet("json", "csv", "junit", "sarif")
)

func unsupportedOutputFormatError(format string, supportedFormats []string) error {
	return errox.InvalidArgs.Newf("unsupported output format used: %q. Choose one of %s",
		format, strings.Join(supportedFormats, " | "))
}

// ObjectPrinterFactory holds all flags for specific printers implementing ObjectPrinter as well as the output format flag
// It acts as an encapsulation for all formats
type ObjectPrinterFactory struct {
	OutputFormat               string
	RegisteredPrinterFactories map[string]CustomPrinterFactory
}

// CustomPrinterFactory holds all configuration for a specific printer implementing ObjectPrinter and handles the registration
// of flags to a command. It acts as a factory for the specific printer, injecting all flag values bound to the factory
// into the created ObjectPrinter
type CustomPrinterFactory interface {
	// AddFlags adds all configuration options as flags to the cobra.Command and binds the current factory's properties
	// to the flag values
	AddFlags(cmd *cobra.Command)
	// SupportedFormats returns a list of formats that the CustomPrinterFactory is capable of creating
	SupportedFormats() []string
	// CreatePrinter creates a printer with the given format and with the factory's properties previously bound to flags
	// of a cobra.Command via AddFlags
	CreatePrinter(format string) (ObjectPrinter, error)
	// validate will verify whether a ObjectPrinter can be created with the current available properties. It returns
	// an error if the creation is not possible. This will only be called by the ObjectPrinterFactory to verify each
	// CustomPrinterFactory and is not expected to be called externally
	validate() error
}

// NewObjectPrinterFactory creates a new ObjectPrinterFactory with all CustomPrinterFactory as well as a default output format
// to use
func NewObjectPrinterFactory(defaultOutputFormat string, customPrinterFactories ...CustomPrinterFactory) (*ObjectPrinterFactory, error) {
	o := &ObjectPrinterFactory{
		OutputFormat: defaultOutputFormat,
	}

	factoryMap := map[string]CustomPrinterFactory{}
	for _, factory := range customPrinterFactories {
		// if a nil pointer is given, ensure it will not be evaluated
		if factory == nil {
			continue
		}
		supportedFormatString := strings.Join(factory.SupportedFormats(), ",")
		if _, ok := factoryMap[supportedFormatString]; !ok {
			factoryMap[supportedFormatString] = factory
		} else {
			return nil, errox.InvariantViolation.Newf("tried to register two printer "+
				"factories which support the same output formats %q: %T and %T",
				supportedFormatString, factory, factoryMap[supportedFormatString])
		}
	}

	if len(factoryMap) == 0 {
		return nil, errox.InvariantViolation.Newf("no custom printer factory added. You must specify at least one "+
			"custom printer factory that supports the %q output format", defaultOutputFormat)
	}

	o.RegisteredPrinterFactories = factoryMap

	if err := o.validate(); err != nil {
		return nil, err
	}
	return o, nil
}

// IsStandardizedFormat checks whether the currently set OutputFormat is a standardized format, i.e. JSON / CSV / YAML
// etc. The caller is expected to not print additional information when this returns true to ensure being compatible
// with the format
func (o *ObjectPrinterFactory) IsStandardizedFormat() bool {
	return standardizedFormats.Contains(o.OutputFormat)
}

// AddFlags will add all flags of registered CustomPrinterFactory as well as the format flag to specify the output format
func (o *ObjectPrinterFactory) AddFlags(cmd *cobra.Command) {
	for _, printerFactory := range o.RegisteredPrinterFactories {
		printerFactory.AddFlags(cmd)
	}

	cmd.PersistentFlags().StringVarP(&o.OutputFormat, "output", "o", o.OutputFormat,
		fmt.Sprintf("Output format. Choose one of: %s", strings.Join(o.supportedFormats(), " | ")))
}

// CreatePrinter will iterate through the registered CustomPrinterFactory and try to create a ObjectPrinter for each.
// It will return an error if the output format cannot be satisfied by any registered CustomPrinterFactory or there
// is a misconfiguration with the specific CustomPrinterFactory that disallows creation of the ObjectPrinter
func (o *ObjectPrinterFactory) CreatePrinter() (ObjectPrinter, error) {
	// only validate whether the current output format is supported or not, only the factory that is matching the format
	// should validate specific configurations set
	if err := o.validateOutputFormat(); err != nil {
		return nil, err
	}

	for supportedFormats, printerFactory := range o.RegisteredPrinterFactories {
		// only invoke factory when output format is a supported format
		if strings.Contains(supportedFormats, o.OutputFormat) {
			printer, err := printerFactory.CreatePrinter(o.OutputFormat)
			return printer, errors.Wrapf(err, "could not create printer: %q", o.OutputFormat)
		}
	}

	// should never happen as output format is declared as supported within the validation
	return nil, unsupportedOutputFormatError(o.OutputFormat, o.supportedFormats())
}

// validate will validate whether the given output format can be satisfied by the registered CustomPrinterFactory. It also
// verifies whether each registered CustomPrinterFactory is able to create a ObjectPrinter with the current configuration
func (o *ObjectPrinterFactory) validate() error {
	var validateErrs *multierror.Error
	for _, printerFactory := range o.RegisteredPrinterFactories {
		if err := printerFactory.validate(); err != nil {
			validateErrs = multierror.Append(validateErrs, err)
		}
	}
	if err := o.validateOutputFormat(); err != nil {
		validateErrs = multierror.Append(validateErrs, err)
	}

	return errors.Wrap(validateErrs.ErrorOrNil(), "invalid printer configuration")
}

// validateOutputFormat will verify whether the currently set OutputFormat is supported by a registered
// CustomPrinterFactory
func (o *ObjectPrinterFactory) validateOutputFormat() error {
	for supportedFormat := range o.RegisteredPrinterFactories {
		if strings.Contains(supportedFormat, o.OutputFormat) {
			return nil
		}
	}

	return unsupportedOutputFormatError(o.OutputFormat, o.supportedFormats())
}

// supportedFormats creates a list of all supported formats based on the CustomPrinterFactory register within the ObjectPrinterFactory
func (o *ObjectPrinterFactory) supportedFormats() []string {
	var supportedFormats []string
	for _, printerFactory := range o.RegisteredPrinterFactories {
		supportedFormats = append(supportedFormats, printerFactory.SupportedFormats()...)
	}
	return supportedFormats
}
