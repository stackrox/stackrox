package eventual

import (
	"fmt"
)

// Format implements fmt.Formatter interface to provide string representation of
// the current state of the eventual value.
func (v *value[T]) Format(f fmt.State, verb rune) {
	switch {
	case f.Flag('#'):
		// For verbose format, use string formatting but preserve
		// width/precision flags.
		fmt.Fprintf(f, fmt.FormatString(f, 's'), v.verbose())
	case !v.IsSet():
		if v == nil {
			fmt.Fprintf(f, fmt.FormatString(f, verb), "<nil>")
		} else {
			fmt.Fprintf(f, fmt.FormatString(f, verb), "<unset>")
		}
	default:
		// Use the actual value with original formatting.
		fmt.Fprintf(f, fmt.FormatString(f, verb), v.value.Load().(T))
	}
}

// verbose string representation for %#v formatting.
func (v *value[T]) verbose() string {
	switch {
	case v == nil:
		return fmt.Sprintf("(eventual.Value[%T])(nil)", v.Get())
	case v.IsSet():
		return fmt.Sprintf("(eventual.Value[%T]){current:%#v default:%#v}", *v.defaultValue, v.value.Load().(T), *v.defaultValue)
	default:
		return fmt.Sprintf("(eventual.Value[%T]){current:<unset> default:%#v}", *v.defaultValue, *v.defaultValue)
	}
}
