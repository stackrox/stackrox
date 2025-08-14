package eventual

import (
	"fmt"
)

// Format implements fmt.Formatter interface to provide string representation of
// the current state of the eventual value.
func (v *Value[T]) Format(f fmt.State, verb rune) {
	ff := fmt.FormatString(f, verb)
	switch {
	case f.Flag('#'):
		fmt.Fprintf(f, fmt.FormatString(f, 's'), v.verbose())
	case !v.IsSet():
		if v == nil {
			fmt.Fprintf(f, ff, "<nil>")
		} else {
			fmt.Fprintf(f, ff, "<unset>")
		}
	default:
		fmt.Fprintf(f, ff, v.Get())
	}
}

// verbose string representation for %#v formatting.
func (v *Value[T]) verbose() string {
	current, ok := v.Maybe()

	if v == nil {
		return fmt.Sprintf("(*eventual.Value[%T])(nil)", current)
	}
	if ok {
		return fmt.Sprintf("(*eventual.Value[%T]){current:%#v default:%#v}", current, current, *v.defaultValue)
	}
	return fmt.Sprintf("(*eventual.Value[%T]){current:<unset> default:%#v}", current, *v.defaultValue)
}
