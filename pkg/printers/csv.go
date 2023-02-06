package printers

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// CSVPrinterOption is a functional option for the CSVPrinter.
type CSVPrinterOption func(*CSVPrinter)

// WithCSVColumnHeaders is a functional option for setting the CSV column headers.
func WithCSVColumnHeaders(headers []string) CSVPrinterOption {
	return func(p *CSVPrinter) {
		p.columnHeaders = sliceutils.ShallowClone(headers)
	}
}

// WithCSVHeaderOptions is a functional option for printing headers. Headers can be
// either printed as comments in the CSV output or not at all.
// By default, headers will be printed.
func WithCSVHeaderOptions(noHeader bool, headerAsComment bool) CSVPrinterOption {
	return func(p *CSVPrinter) {
		p.headerPrintOption = createHeaderPrintOption(noHeader, headerAsComment)
	}
}

type headerPrintOption int

const (
	printHeaders headerPrintOption = iota
	printHeadersAsComments
	printNoHeaders
)

func createHeaderPrintOption(noHeaderFlag, headerAsCommentFlag bool) headerPrintOption {
	if noHeaderFlag {
		return printNoHeaders
	}
	if headerAsCommentFlag {
		return printHeadersAsComments
	}
	return printHeaders
}

// CSVPrinter will print a CSV output from a given JSON object.
type CSVPrinter struct {
	columnHeaders         []string
	rowJSONPathExpression string
	headerPrintOption     headerPrintOption
}

// NewCSVPrinter creates a CSVPrinter from the options set.
// A CSVPrinter expects a JSON Object and JSON Path expression that is compatible
// with GJSON (https://github.com/tidwall/gjson).
// When printing, the CSVPrinter will take the given JSON object, apply a row expression via a gjson
// multi path expression to retrieve the data from the JSON object and print the result in tabular format.
// The JSON Object itself MUST be passable to json.Marshal, so it CAN NOT be a direct JSON input.
// For the structure of the JSON object, it is preferred to have arrays of structs instead of
// array of elements, since structs will provide default values if the field is missing.
// The gjson expression syntax (https://github.com/tidwall/gjson/blob/master/SYNTAX.md) offers more complex
// and advanced scenarios, if you require them and the below example is not sufficient.
// Additionally, there are custom GJSON modifiers, which will post-process expression results. Currently,
// the gjson.ListModifier and gjson.BoolReplaceModifier are available, see their documentation on usage and
// GJSON's syntax expression to read more about modifiers.
// The following example illustrates a JSON compatible structure and its gjson multi path expression
// JSON structure:
//
//	type data struct {
//			Infos 	[]info `json:"infos"`
//			Name 	string `json:"name"`
//	}
//
//	type info struct {
//			info 	string `json:"info"`
//			topic 	string `json:"topic"`
//	}
//
// Data:
//
//	data := &data{Name: "example", Infos: []info{
//											{info: "info1", topic: "topic1"},
//											{info: "info2", topic: "topic2"},
//											{info: "info3", topic: "topic3"},
//											}
//
// gjson multi path expression: "{name,infos.#.info,infos.#.topic}"
//   - bundle multiple gjson expression surrounded by "{}" to form a multi path expression
//   - specify "#" to visit each element in the array
//   - each expression in the multi path expression is correlated with the given header(s)!
//
// headers := []string{"name", "info", "topic"}
//
// This would result in the following rows for the CSVPrinter
// | name	 | info  | topic  |
// | example | info1 | topic1 |
// | example | info2 | topic2 |
// | example | info3 | topic3 |
func NewCSVPrinter(rowJSONPathExpression string, options ...CSVPrinterOption) *CSVPrinter {
	printer := &CSVPrinter{}

	for _, opt := range options {
		opt(printer)
	}

	printer.rowJSONPathExpression = rowJSONPathExpression
	return printer
}

// Print prints the given json object into csv row(s) and writes it to the given writer.
// It will return an error if there are any issues with the JSON object, constructing the rows or
// writing to the io.Writer.
func (c *CSVPrinter) Print(jsonObject interface{}, out io.Writer) error {
	csvWriter := csv.NewWriter(out)

	rowMapper, err := gjson.NewRowMapper(jsonObject, c.rowJSONPathExpression)
	if err != nil {
		return err
	}

	rows, err := rowMapper.CreateRows()
	if err != nil {
		return err
	}

	switch c.headerPrintOption {
	case printNoHeaders:
	case printHeadersAsComments:
		if err := printHeadersAsComment(c.columnHeaders, csvWriter, out); err != nil {
			return err
		}
	case printHeaders:
		if err := csvWriter.Write(c.columnHeaders); err != nil {
			return err
		}
	}

	return printRows(rows, csvWriter)
}

func printHeadersAsComment(headers []string, csvWriter *csv.Writer, out io.Writer) error {
	// Silently ignore if headers should be printed as comments but no headers are available
	if len(headers) == 0 {
		return nil
	}

	// ensure that anything internally buffered is written to out before the fmt.FPrint call
	csvWriter.Flush()

	// Print a preceding ";" to mark the following line as commented
	fmt.Fprint(out, "; ")

	return csvWriter.Write(headers)
}

func printRows(rows [][]string, csvWriter *csv.Writer) error {
	for _, row := range rows {
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
