package autobool

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/stackrox/rox/roxctl/common"
)

/*
 * This package provides a pflag Value implementation which encodes a tristate value as in false | true | auto.
 * Such a value is called an auto bool value.
 */

// Value models an auto bool value.
type Value struct {
	bp **bool
}

// New constructs a new auto bool value.
func New(val *bool, bp **bool) Value {
	*bp = val
	return Value{bp: bp}
}

// Set is part of the Flag interface. It implements setting of a flag.
func (v Value) Set(s string) error {
	// Check first if the user intends to set 'auto'.
	if strings.ToLower(s) == "auto" {
		*v.bp = nil
		return nil
	}

	// Then check for booleans.
	b, err := strconv.ParseBool(s)
	if err != nil {
		return common.ErrInvalidCommandOption.CausedBy(err)
	}

	*v.bp = &b
	return nil
}

// Type is part of the Flag interface. It describes the type of the modelled value.
func (v Value) Type() string {
	return "string"
}

// String is part of the Flag interface. It returns the stringified version of the value.
func (v Value) String() string {
	if *v.bp == nil {
		return "auto"
	}
	return strconv.FormatBool(**v.bp)
}

// NewFlag is a convenience function instantiating a new autobool value with `New` followed by
// setting the new flag's `NoOptDefVal` property to `true` so the user can simply use `--name`
// instead of `--name=true` as it is generally expected for boolean flags.
func NewFlag(flags *pflag.FlagSet, val **bool, name string, usage string) {
	flags.Var(New(nil, val), name, fmt.Sprintf("%s (auto, true, false)", usage))
	flags.Lookup(name).NoOptDefVal = "true"
}
