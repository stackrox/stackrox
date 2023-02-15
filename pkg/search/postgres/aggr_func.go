package postgres

import "github.com/stackrox/rox/pkg/postgres/walker"

// AggrFunc is the internal enum representation of the SQL aggregate functions.
type AggrFunc string

// Defines all the SQL aggregate functions.
const (
	UnsetAggrFunc AggrFunc = ""
	CountAggrFunc AggrFunc = "count"
	MinAggrFunc   AggrFunc = "min"
	MaxAggrFunc   AggrFunc = "max"
)

func (a AggrFunc) String() string {
	return string(a)
}

var (
	aggrFuncToDataType = map[AggrFunc]walker.DataType{
		CountAggrFunc: walker.Integer,

		// MinAggrFunc and MaxAggrFunc can be performed on text or numeric. Therefore, use the underlying column's datatype.
		MinAggrFunc: "",
		MaxAggrFunc: "",
	}
)
