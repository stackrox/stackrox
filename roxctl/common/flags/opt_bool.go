package flags

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type optBool struct {
	boolp    **bool
	unsetRep string
}

func (v optBool) String() string {
	if *v.boolp != nil {
		return strconv.FormatBool(**v.boolp)
	}
	return v.unsetRep
}

func (v optBool) Set(strVal string) error {
	if strVal == v.unsetRep {
		*v.boolp = nil
		return nil
	}
	b, err := strconv.ParseBool(strVal)
	if err != nil {
		return errors.Wrapf(err, "invalid boolean value: %q", strVal)
	}
	*v.boolp = &b
	return nil
}

func (v optBool) Type() string {
	return fmt.Sprintf("false|true|%s", v.unsetRep)
}

// OptBoolFlagVarPF register a given "optional bool" (represented as a bool pointer) flag variable, using unsetRep
// as the representation for the unset value.
func OptBoolFlagVarPF(flagSet *pflag.FlagSet, boolp **bool, name, shorthand, usage, unsetRep string) *pflag.Flag {
	f := flagSet.VarPF(optBool{boolp: boolp, unsetRep: unsetRep}, name, shorthand, usage)
	f.NoOptDefVal = "true"
	return f
}
