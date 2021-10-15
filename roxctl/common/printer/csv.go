package printer

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/stackrox/rox/pkg/sliceutils"
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

type csvPrinter struct {
	columnHeaders         []string
	rowJSONPathExpression string
	headerPrintOption     headerPrintOption
}

// newCSVPrinter returns a printer with given configuration capable of printing output formatted as CSV
func newCSVPrinter(columnHeaders []string, rowJSONPathExpression string, noHeader, headerAsComment bool) *csvPrinter {
	// since we are potentially mutating the column headers, ensure we copy them to not impact
	// the given slice's reference.
	headers := sliceutils.StringClone(columnHeaders)
	return &csvPrinter{
		columnHeaders:         headers,
		rowJSONPathExpression: rowJSONPathExpression,
		headerPrintOption:     createHeaderPrintOption(noHeader, headerAsComment),
	}
}

func (c *csvPrinter) Print(jsonObject interface{}, out io.Writer) error {
	csvWriter := csv.NewWriter(out)

	rows, err := getRowsFromObject(jsonObject, c.rowJSONPathExpression)
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
