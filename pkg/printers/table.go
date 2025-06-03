package printers

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/gjson"
)

// TablePrinterOptions is a functional option for the TablePrinter.
type TablePrinterOptions func(*TablePrinter)

// WithTableHeadersOption is a functional option for setting the headers of the table,
// which headers to merge and whether to print headers at all.
func WithTableHeadersOption(headers []string, merge bool, noHeader bool) TablePrinterOptions {
	return func(p *TablePrinter) {
		p.headers = headers
		p.mergeHierarchical = merge
		p.noHeader = noHeader
	}
}

// WithTableHideUnpopulatedRowsOption is a functional option for hiding rows of the table,
// when those rows have an unpopulated spot from a column marked as required.
func WithTableHideUnpopulatedRowsOption(requiredColumns []string) TablePrinterOptions {
	return func(p *TablePrinter) {
		p.columnTreeOptions = []gjson.ColumnTreeOptions{gjson.HideRowsIfColumnNotPopulated(requiredColumns)}
	}
}

// TablePrinter will print a table output from a given JSON Object.
type TablePrinter struct {
	headers               []string
	rowJSONPathExpression string
	mergeHierarchical     bool
	noHeader              bool
	columnTreeOptions     []gjson.ColumnTreeOptions
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

func (p *TablePrinter) createTableWriter(out io.Writer) *tablewriter.Table {
	mergeMode := tw.MergeNone
	if p.mergeHierarchical {
		mergeMode = tw.MergeHierarchical
	}
	table := tablewriter.NewTable(out,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleASCII),
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenRows: tw.On},
				Lines:      tw.Lines{ShowFooterLine: tw.On},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{Alignment: tw.AlignCenter},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{
					MergeMode: mergeMode,
					Alignment: tw.AlignCenter,
					AutoWrap:  tw.WrapNormal,
				},
			},
		}),
	)
	if !p.noHeader {
		table.Header(p.headers)
	}
	return table
}

// Print will print the given JSON object as a tabular format to the io.Writer.
// It will return an error if there is an issue with the JSON object, creation of the rows failed or
// it was not possible to write to the io.Writer.
func (p *TablePrinter) Print(jsonObject interface{}, out io.Writer) error {
	// create table writer with headers and options.
	table := p.createTableWriter(out)

	// retrieve rows from JSON object via JSON path expression.
	rowMapper, err := gjson.NewRowMapper(jsonObject, p.rowJSONPathExpression, p.columnTreeOptions...)
	if err != nil {
		return err
	}

	rows, err := rowMapper.CreateRows()
	if err != nil {
		return errors.Wrap(err, "could not create rows")
	}
	if err := table.Bulk(rows); err != nil {
		return errors.Wrap(err, "could not bulk add rows")
	}
	if err := table.Render(); err != nil {
		return errors.Wrap(err, "could not render the table")
	}
	return nil
}
