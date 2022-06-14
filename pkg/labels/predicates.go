package labels

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

type setPredicate struct {
	values     cofiniteSet
	noneResult bool // what to return if the value is not present
}

func (p *setPredicate) Matches(valp *string) bool {
	if valp == nil {
		return p.noneResult
	}
	return p.values.Contains(*valp)
}

func (p *setPredicate) And(other *setPredicate) *setPredicate {
	return &setPredicate{
		values:     p.values.Intersect(other.values),
		noneResult: p.noneResult && other.noneResult,
	}
}

func predicateForRequirement(req *storage.LabelSelector_Requirement) (*setPredicate, error) {
	switch req.GetOp() {
	case storage.LabelSelector_IN:
		return &setPredicate{
			values:     makeCofiniteSet(false, req.GetValues()...),
			noneResult: false,
		}, nil
	case storage.LabelSelector_NOT_IN:
		return &setPredicate{
			values:     makeCofiniteSet(true, req.GetValues()...),
			noneResult: true,
		}, nil
	case storage.LabelSelector_EXISTS:
		return &setPredicate{
			values:     makeCofiniteSet(true),
			noneResult: false,
		}, nil
	case storage.LabelSelector_NOT_EXISTS:
		return &setPredicate{
			values:     makeCofiniteSet(false),
			noneResult: true,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported requirement operator %v", req.GetOp())
	}
}
