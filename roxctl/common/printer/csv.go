package printer

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/roxctl/common/printer/mapper"
)

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

// CSVPrinter contains fields related to formatting data into csv format
type CSVPrinter struct {
	columnHeaders         []string
	rowJSONPathExpression string
	headerPrintOption     headerPrintOption
}

func newCSVPrinter(columnHeaders []string, rowJSONPathExpression string, noHeader, headerAsComment bool) *CSVPrinter {
	// since we are potentially mutating the column headers, ensure we copy them to not impact
	// the given slice's reference.
	headers := sliceutils.StringClone(columnHeaders)
	return &CSVPrinter{
		columnHeaders:         headers,
		rowJSONPathExpression: rowJSONPathExpression,
		headerPrintOption:     createHeaderPrintOption(noHeader, headerAsComment),
	}
}

// Print prints the given json object into csv row(s) and writes it to the given writer
func (c *CSVPrinter) Print(jsonObject interface{}, out io.Writer) error {
	csvWriter := csv.NewWriter(out)

	rowMapper, err := mapper.NewRowMapper(jsonObject, c.rowJSONPathExpression)
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
