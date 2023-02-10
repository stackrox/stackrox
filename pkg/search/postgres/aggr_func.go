package postgres

import "github.com/stackrox/rox/pkg/postgres/walker"

// AggrFunc is the internal enum representation of the SQL aggregate functions.
type AggrFunc string

// Defines all the SQL aggregate functions.
const (
	Unset AggrFunc = ""
	Count AggrFunc = "count"
	Min   AggrFunc = "min"
	Max   AggrFunc = "max"
)

func (a AggrFunc) String() string {
	return string(a)
}

var (
	aggrFuncToDataType = map[AggrFunc]walker.DataType{
		Count: walker.Integer,

		// Min and max can be performed on text or numeric. Therefore, use the underlaying column's datatype.
		Min: "",
		Max: "",
	}
)
