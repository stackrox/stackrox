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
	Merge                     bool
	Columns                   []string
	ColumnJSONPathExpressions []string
	NoHeader                  bool
	// HeaderAsComment only applies to the "csv" format and prints headers as comment lines in the CSV output
	HeaderAsComment bool
}

// NewTabularPrinterFactory creates new TabularPrinterFactory with the injected default values
func NewTabularPrinterFactory(merge bool, columns, columnJSONPathExpressions []string, noHeader, headerAsComment bool) *TabularPrinterFactory {
	return &TabularPrinterFactory{
		Merge:                     merge,
		Columns:                   columns,
		ColumnJSONPathExpressions: columnJSONPathExpressions,
		NoHeader:                  noHeader,
		HeaderAsComment:           headerAsComment,
	}
}

// AddFlags will add all tabular printer specific flags to the cobra.Command
func (t *TabularPrinterFactory) AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&t.Merge, "merge-output", t.Merge, "Merge duplicate cells in prettified tabular output")
	cmd.PersistentFlags().StringArrayVar(&t.Columns, "columns", t.Columns, "Columns to print in tabular output")
	cmd.PersistentFlags().StringArrayVar(&t.ColumnJSONPathExpressions, "columns-jsonpath-expressions", t.ColumnJSONPathExpressions,
		"JSON Path expressions to fill the columns in tabular output. NOTE: These need to be compliant with gjson "+
			"as well as have the same length as specified columns")
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
		panic("pretty printer is not implemented")
	case "csv":
		panic("csv printer is not implemented")
	default:
		return nil, fmt.Errorf("invalid output format used for Tabular Printer: %q", format)
	}
}

// Validate verifies whether the current configuration can be used to create an ObjectPrinter. It will return an error
// if it is not possible
func (t *TabularPrinterFactory) validate() error {
	if len(t.Columns) != len(t.ColumnJSONPathExpressions) {
		return errors.New("Different number of columns and JSON Path expressions specified. Make sure you " +
			"specify the same number of arguments for both")
	}

	if t.NoHeader && t.HeaderAsComment {
		return errors.New("cannot specify both --no-header as well as --headers-as-comment flags. You must " +
			"choose either one")
	}
	return nil
}
