package printer

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// UnsupportedOutputFormatError creates a standardized error for unsupported format inputs in combination with ObjectPrinter
type UnsupportedOutputFormatError struct {
	OutputFormat     string
	SupportedFormats []string
}

func (u UnsupportedOutputFormatError) Error() string {
	return fmt.Sprintf("unsupported output format used: %q. Please choose one of the supported formats: %s",
		u.OutputFormat, strings.Join(u.SupportedFormats, "|"))
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

	o.RegisteredPrinterFactories = make(map[string]CustomPrinterFactory, len(customPrinterFactories))
	for _, factory := range customPrinterFactories {
		supportedFormatString := strings.Join(factory.SupportedFormats(), ",")
		if _, ok := o.RegisteredPrinterFactories[supportedFormatString]; !ok {
			o.RegisteredPrinterFactories[supportedFormatString] = factory
		} else {
			return nil, fmt.Errorf("tried to registere two printer factories which support the same output formats %q: %v and %v",
				supportedFormatString, factory, o.RegisteredPrinterFactories[supportedFormatString])
		}
	}

	if err := o.validate(); err != nil {
		return nil, err
	}
	return o, nil
}

// SupportedFormats creates a list of all supported formats based on the CustomPrinterFactory register within the ObjectPrinterFactory
func (o *ObjectPrinterFactory) SupportedFormats() []string {
	var supportedFormats []string
	for _, printerFactory := range o.RegisteredPrinterFactories {
		supportedFormats = append(supportedFormats, printerFactory.SupportedFormats()...)
	}
	return supportedFormats
}

// validate will validate whether the given output format can be satisfied by the registered CustomPrinterFactory. It also
// verifies whether each registered CustomPrinterFactory is able to create a ObjectPrinter with the current configuration
func (o *ObjectPrinterFactory) validate() error {
	for _, printerFactory := range o.RegisteredPrinterFactories {
		if err := printerFactory.validate(); err != nil {
			return err
		}
	}

	return o.validateOutputFormat()
}

func (o *ObjectPrinterFactory) validateOutputFormat() error {
	for supportedFormat := range o.RegisteredPrinterFactories {
		if strings.Contains(o.OutputFormat, supportedFormat) {
			return nil
		}
	}

	return UnsupportedOutputFormatError{OutputFormat: o.OutputFormat, SupportedFormats: o.SupportedFormats()}
}

// AddFlags will add all flags of registered CustomPrinterFactory as well as the format flag to specify the output format
func (o *ObjectPrinterFactory) AddFlags(cmd *cobra.Command) {
	for _, printerFactory := range o.RegisteredPrinterFactories {
		printerFactory.AddFlags(cmd)
	}

	cmd.PersistentFlags().StringVar(&o.OutputFormat, "format", o.OutputFormat,
		fmt.Sprintf("Output format. Choose one of: %s", strings.Join(o.SupportedFormats(), "|")))
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
		if strings.Contains(o.OutputFormat, supportedFormats) {
			return printerFactory.CreatePrinter(o.OutputFormat)
		}
	}

	// should never happen as output format is declared as supported within the validation
	return nil, UnsupportedOutputFormatError{OutputFormat: o.OutputFormat, SupportedFormats: o.SupportedFormats()}
}
