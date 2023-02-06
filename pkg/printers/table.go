package printers

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/set"
)

// TablePrinterOptions is a functional option for the TablePrinter.
type TablePrinterOptions func(*TablePrinter)

// WithTableHeadersOption is a functional option for setting the headers of the table,
// which headers to merge and whether to print headers at all.
func WithTableHeadersOption(headers []string, headersToMerge []string, noHeader bool) TablePrinterOptions {
	return func(p *TablePrinter) {
		p.headers = headers
		p.columnIndexesToMerge = indexesToMergeFromColumnNames(p.headers, set.NewStringSet(headersToMerge...))
		p.noHeader = noHeader
	}
}

// TablePrinter will print a table output from a given JSON Object.
type TablePrinter struct {
	headers               []string
	rowJSONPathExpression string
	// columnIndexesToMerge set to non nil will instruct the table writer to merge all identical cells.
	// There will be no precedence in any fashion applied.
	columnIndexesToMerge []int
	noHeader             bool
}

// NewTablePrinter creates a TablePrinter from the options set.
// A tabular printer expects a JSON Object and JSON Path expression that is compatible
// with GJSON (https://github.com/tidwall/gjson).
// When printing, the TablePrinter will take the given JSON object, apply a row expression via a gjson
// multi path expression to retrieve the data from the JSON object and print the result in tabular format.
// The JSON Object itself MUST be passable to json.Marshal, so it CAN NOT be a direct JSON input.
// For the structure of the JSON object, it is preferred to have arrays of structs instead of
// array of elements, since structs will provide default values if the field is missing.
// The gjson expression syntax (https://github.com/tidwall/gjson/blob/master/SYNTAX.md) offers more complex
// and advanced scenarios, if you require them and the below example is not sufficient.
// Additionally, there are custom GJSON modifiers, which will post-process expression results. Currently,
// the mapper.ListModifier and mapper.BoolReplaceModifier are available, see their documentation on usage and
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
// This would result in the following rows for the tabular printers
// | name	 | info  | topic  |
// | example | info1 | topic1 |
// | example | info2 | topic2 |
// | example | info3 | topic3 |
func NewTablePrinter(rowJSONPathExpression string, options ...TablePrinterOptions) *TablePrinter {
	printer := &TablePrinter{}

	for _, opt := range options {
		opt(printer)
	}

	printer.rowJSONPathExpression = rowJSONPathExpression
	return printer
}

func indexesToMergeFromColumnNames(headers []string, columnsToMerge set.StringSet) []int {
	if columnsToMerge.Cardinality() == 0 {
		return nil
	}
	indexesToMerge := make([]int, 0, len(columnsToMerge))
	for i, header := range headers {
		if columnsToMerge.Contains(header) {
			indexesToMerge = append(indexesToMerge, i)
		}
	}
	return indexesToMerge
}

func (p *TablePrinter) createTableWriter(out io.Writer) *tablewriter.Table {
	tw := tablewriter.NewWriter(out)
	if !p.noHeader {
		tw.SetHeader(p.headers)
	}
	if p.columnIndexesToMerge != nil {
		tw.SetAutoMergeCellsByColumnIndex(p.columnIndexesToMerge)
	}
	tw.SetRowLine(true)
	tw.SetReflowDuringAutoWrap(false)

	tw.SetAlignment(tablewriter.ALIGN_CENTER)
	return tw
}

// Print will print the given JSON object as a tabular format to the io.Writer.
// It will return an error if there is an issue with the JSON object, creation of the rows failed or
// it was not possible to write to the io.Writer.
func (p *TablePrinter) Print(jsonObject interface{}, out io.Writer) error {
	// create table writer with headers and options.
	tw := p.createTableWriter(out)

	// retrieve rows from JSON object via JSON path expression.
	rowMapper, err := gjson.NewRowMapper(jsonObject, p.rowJSONPathExpression)
	if err != nil {
		return err
	}

	rows, err := rowMapper.CreateRows()
	if err != nil {
		return err
	}
	tw.AppendBulk(rows)
	// TableWriter library does not offer any way to check for errors when rendering.
	tw.Render()

	// Handle potential writing errors to io.Writer since table library is ignoring errors.
	if _, err := out.Write(nil); err != nil {
		return err
	}
	return nil
}
