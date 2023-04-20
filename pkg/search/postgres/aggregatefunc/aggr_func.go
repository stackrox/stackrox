package aggregatefunc

import (
	"fmt"

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

// Name returns the name for aggregate function.
func (a AggrFunc) Name() string {
	return a.name
}

// String returns the string form of aggregate function applied to arg.
func (a AggrFunc) String(arg string) string {
	return fmt.Sprintf("%s(%s)", a.name, arg)
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
func GetAggrFuncForV1(aggregation *v1.AggregateBy) (AggrFunc, bool) {
	aggrFunc, found := v1AggregationToGoAggrFunc[aggregation.GetAggrFunc()]
	if !found {
		return Unset, false
	}
	return aggrFunc, aggregation.GetDistinct()
}
