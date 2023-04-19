package aggregatefunc

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
)

// AggrFunc is the internal enum representation of the SQL aggregate functions.
type AggrFunc struct {
	name     string
	proto    v1.Aggregation
	dataType postgres.DataType
}

// Defines all the SQL aggregate functions.
var (
	allAggrFuncs              = make(map[string]AggrFunc)
	v1AggregationToGoAggrFunc = make(map[v1.Aggregation]AggrFunc)

	Unset = newAggrFunc("", v1.Aggregation_UNSET, "")
	Count = newAggrFunc("count", v1.Aggregation_COUNT, postgres.Integer)
	// Min and Max can be performed on text or numeric. Therefore, use the underlying column's datatype.
	Min = newAggrFunc("min", v1.Aggregation_MIN, "")
	Max = newAggrFunc("max", v1.Aggregation_MAX, "")
)

func newAggrFunc(name string, proto v1.Aggregation, dataType postgres.DataType) AggrFunc {
	f := AggrFunc{name: name, proto: proto, dataType: dataType}
	allAggrFuncs[name] = f
	v1AggregationToGoAggrFunc[proto] = f

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

// Proto returns the v1 type for aggregrate function.
func (a AggrFunc) Proto() v1.Aggregation {
	return a.proto
}

// GetAggrFunc returns aggregate function registered for specified name.
func GetAggrFunc(name string) AggrFunc {
	aggrFunc, found := allAggrFuncs[name]
	if !found {
		return Unset
	}
	return aggrFunc
}

// GetAggrFuncForV1 returns aggregate function registered for specified v1.Aggregation.
func GetAggrFuncForV1(aggregation v1.Aggregation) AggrFunc {
	aggrFunc, found := v1AggregationToGoAggrFunc[aggregation]
	if !found {
		return Unset
	}
	return aggrFunc
}
