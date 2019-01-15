package stringutils

import (
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

// Wrap performs line-wrapping of the given text at 80 characters in length.
// Intended to be uses as a test template function.
func Wrap(text string) string {
	wrapped := wordwrap.WrapString(text, 80)
	wrapped = strings.TrimSpace(wrapped)
	wrapped = strings.Replace(wrapped, "\n", "\n      ", -1)
	return wrapped
}
