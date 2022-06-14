package labels

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// CompiledSelector is a representation of a label selector optimized for quick evaluations.
type CompiledSelector map[string]*setPredicate

// CompileSelector converts a LabelSelector proto into a CompiledSelector that is optimized for quick evaluations.
func CompileSelector(proto *storage.LabelSelector) (CompiledSelector, error) {
	if proto == nil {
		return nil, nil
	}

	cs := make(CompiledSelector)
	for k, v := range proto.GetMatchLabels() {
		cs[k] = &setPredicate{
			values:     makeCofiniteSet(false, v),
			noneResult: false,
		}
	}

	for _, req := range proto.GetRequirements() {
		pred, err := predicateForRequirement(req)
		if err != nil {
			return nil, err
		}

		k := req.GetKey()
		if existingPred := cs[k]; existingPred != nil {
			cs[k] = existingPred.And(pred)
		} else {
			cs[k] = pred
		}
	}

	return cs, nil
}

// Matches checks if the given label set is matched by the label selector.
func (s CompiledSelector) Matches(labels map[string]string) bool {
	if s == nil {
		return false
	}

	for key, pred := range s {
		value, ok := labels[key]
		valp := &value
		if !ok {
			valp = nil
		}
		if !pred.Matches(valp) {
			return false
		}
	}
	return true
}

// MatchesNone checks if this selector cannot match anything. Note that only a `true` return
// value is 100% reliable; it is possible that a CompiledSelector for which this method returns
// `false` will never match anything as well.
func (s CompiledSelector) MatchesNone() bool {
	return s == nil
}

// MatchesAll checks if this selector matches every set of labels (including an empty one).
func (s CompiledSelector) MatchesAll() bool {
	return s != nil && len(s) == 0
}

// MatchLabels checks if the given label selector matches the given label map. If the label selector proto is invalid,
// false will be returned.
func MatchLabels(sel *storage.LabelSelector, labels map[string]string) bool {
	cs, _ := CompileSelector(sel)
	return cs.Matches(labels)
}
