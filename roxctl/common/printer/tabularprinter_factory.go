package printer

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// TabularPrinterFactory holds all configuration options of tabular printers, specifically CSVPrinter and PrettyPrinter
// It is an implementation of CustomPrinterFactory and acts as a factory for tabular printers
type TabularPrinterFactory struct {
	// Merge only applies to the "table" format and merges certain cells within the output
	Merge                 bool
	Headers               []string
	RowJSONPathExpression string
	NoHeader              bool
	// HeaderAsComment only applies to the "csv" format and prints headers as comment lines in the CSV output
	HeaderAsComment bool
}

// NewTabularPrinterFactory creates new TabularPrinterFactory with the injected default values
func NewTabularPrinterFactory(merge bool, headers []string, rowJSONPathExpression string, noHeader, headerAsComment bool) *TabularPrinterFactory {
	return &TabularPrinterFactory{
		Merge:                 merge,
		Headers:               headers,
		RowJSONPathExpression: rowJSONPathExpression,
		NoHeader:              noHeader,
		HeaderAsComment:       headerAsComment,
	}
}

// AddFlags will add all tabular printer specific flags to the cobra.Command
func (t *TabularPrinterFactory) AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&t.Merge, "merge-output", t.Merge, "Merge duplicate cells in prettified tabular output")
	cmd.PersistentFlags().StringArrayVar(&t.Headers, "headers", t.Headers, "Headers to print in tabular output")
	cmd.PersistentFlags().StringVar(&t.RowJSONPathExpression, "row-jsonpath-expressions", t.RowJSONPathExpression,
		"JSON Path expression to create a row from the JSON object. This leverages gJSON (https://github.com/tidwall/gjson)."+
			" NOTE: The amount of expressions within the multi-path has to match the amount of provided headers.")
	cmd.PersistentFlags().BoolVar(&t.NoHeader, "no-header", t.NoHeader, "Print no headers for tabular output")
	cmd.PersistentFlags().BoolVar(&t.HeaderAsComment, "headers-as-comments", t.HeaderAsComment, "Print headers "+
		"as comments in CSV tabular output")
}

// SupportedFormats returns the supported printer format that can be created by TabularPrinterFactory
func (t *TabularPrinterFactory) SupportedFormats() []string {
	return []string{"table", "csv"}
}

// CreatePrinter creates a tabular printer from the options set. If the format is unsupported, or it is not possible
// to create an ObjectPrinter with the current configuration it will return an error
func (t *TabularPrinterFactory) CreatePrinter(format string) (ObjectPrinter, error) {
	if err := t.validate(); err != nil {
		return nil, err
	}
	switch strings.ToLower(format) {
	case "table":
		return newTablePrinter(t.Headers, t.RowJSONPathExpression, t.Merge, t.NoHeader), nil
	case "csv":
		panic("csv printer is not implemented")
	default:
		return nil, fmt.Errorf("invalid output format used for Tabular Printer: %q", format)
	}
}

// Validate verifies whether the current configuration can be used to create an ObjectPrinter. It will return an error
// if it is not possible
func (t *TabularPrinterFactory) validate() error {
	// verify that the GJSON multi path expression matches the amount of headers.
	// Example: multi-path expression: {some.expression,another.expression}
	amountJSONPathExpressions := 0
	if t.RowJSONPathExpression != "" {
		amountJSONPathExpressions = len(strings.Split(t.RowJSONPathExpression, ","))
	}

	if len(t.Headers) != amountJSONPathExpressions {
		return errors.New("Different number of headers and JSON Path expressions specified. Make sure you " +
			"specify the same number of arguments for both")
	}

	if t.NoHeader && t.HeaderAsComment {
		return errors.New("cannot specify both --no-header as well as --headers-as-comment flags. You must " +
			"choose either one")
	}
	return nil
}
