package predicate

import (
	"reflect"

	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
)

func createTimestampPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForTimestamp(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		instanceTS, ok := instance.Interface().(*protocompat.Timestamp)

		if ok && baseMatcher(instanceTS) {
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, "%d", instanceTS.GetSeconds()),
			}, true
		}
		return nil, false
	}), nil
}
