package printer

import (
	"io"
)

// ObjectPrinter abstracts different printers which are capable of printing formatted data of a JSON compatible
// object
type ObjectPrinter interface {
	// Print takes a jsonObject as input and prints the contents of it in a formatted way to the given io.Writer
	// NOTE: The passed jsonObject MUST be able to be passed to json.Marshal
	Print(jsonObject interface{}, out io.Writer) error
}
