package printer

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/stackrox/rox/roxctl/common/printer/mapper"
)

type tablePrinter struct {
	headers               []string
	rowJSONPathExpression string
	// autoMergeCells will instruct the table writer to merge all identical cells.
	// There will be no precedence in any fashion applied.
	autoMergeCells bool
	noHeader       bool
}

// newTablePrinter returns a table printer with injected options capable of printing data in
// prettified tabular format.
func newTablePrinter(headers []string, rowJSONPathExpression string, merge, noHeader bool) *tablePrinter {
	return &tablePrinter{
		headers:               headers,
		rowJSONPathExpression: rowJSONPathExpression,
		autoMergeCells:        merge,
		noHeader:              noHeader,
	}
}

func (p *tablePrinter) createTableWriter(out io.Writer) *tablewriter.Table {
	tw := tablewriter.NewWriter(out)
	if !p.noHeader {
		tw.SetHeader(p.headers)
	}
	tw.SetAutoMergeCells(p.autoMergeCells)
	tw.SetRowLine(true)
	tw.SetReflowDuringAutoWrap(false)
	return tw
}

func (p *tablePrinter) Print(jsonObject interface{}, out io.Writer) error {
	// create table writer with headers and options.
	tw := p.createTableWriter(out)

	// retrieve rows from JSON object via JSON path expression.
	rowMapper, err := mapper.NewRowMapper(jsonObject, p.rowJSONPathExpression)
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
