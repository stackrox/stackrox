package aggregatefunc

import (
	"github.com/stackrox/rox/pkg/postgres"
)

// AggrFunc is the internal enum representation of the SQL aggregate functions.
type AggrFunc struct {
	name     string
	dataType postgres.DataType
}

// Defines all the SQL aggregate functions.
var (
	allAggrFuncs = make(map[string]AggrFunc)

	Unset = newAggrFunc("", "")
	Count = newAggrFunc("count", postgres.Integer)
	// Min and Max can be performed on text or numeric. Therefore, use the underlying column's datatype.
	Min = newAggrFunc("min", "")
	Max = newAggrFunc("max", "")
)

func newAggrFunc(name string, dataType postgres.DataType) AggrFunc {
	f := AggrFunc{name: name, dataType: dataType}
	allAggrFuncs[name] = f
	return f
}

// DataType returns the response datatype of the aggregate function. If empty, the datatype of underlying field applies.
func (a AggrFunc) DataType() postgres.DataType {
	return a.dataType
}

// String returns the name for aggregate function.
func (a AggrFunc) String() string {
	return a.name
}

// GetAggrFunc returns aggregate function registered for specified name.
func GetAggrFunc(name string) AggrFunc {
	return allAggrFuncs[name]
}
