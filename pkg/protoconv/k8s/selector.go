package k8s

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	roxToK8sOpMap = map[storage.LabelSelector_Operator]v1.LabelSelectorOperator{
		storage.LabelSelector_IN:         v1.LabelSelectorOpIn,
		storage.LabelSelector_NOT_IN:     v1.LabelSelectorOpNotIn,
		storage.LabelSelector_EXISTS:     v1.LabelSelectorOpExists,
		storage.LabelSelector_NOT_EXISTS: v1.LabelSelectorOpDoesNotExist,
	}

	k8sToRoxOpMap = func() map[v1.LabelSelectorOperator]storage.LabelSelector_Operator {
		result := make(map[v1.LabelSelectorOperator]storage.LabelSelector_Operator, len(roxToK8sOpMap))
		for k, v := range roxToK8sOpMap {
			result[v] = k
		}
		return result
	}
)

// FromRoxLabelSelector converts a StackRox LabelSelector protobuf object into the corresponding Kubernetes
// representation.
func FromRoxLabelSelector(sel *storage.LabelSelector) (*v1.LabelSelector, error) {
	if sel == nil {
		return nil, nil
	}

	var k8sReqs []v1.LabelSelectorRequirement
	if sel.GetRequirements() != nil {
		k8sReqs = make([]v1.LabelSelectorRequirement, len(sel.GetRequirements()))
		for i, roxReq := range sel.GetRequirements() {
			k8sReq, err := FromRoxLabelSelectorRequirement(roxReq)
			if err != nil {
				return nil, errors.Wrap(err, "converting requirement")
			}
			k8sReqs[i] = *k8sReq
		}
	}
	return &v1.LabelSelector{
		MatchLabels:      sel.GetMatchLabels(),
		MatchExpressions: k8sReqs,
	}, nil
}

// FromRoxLabelSelectorRequirement converts a StackRox LabelSelector Requirement protobuf object into the corresponding
// Kubernetes representation.
func FromRoxLabelSelectorRequirement(req *storage.LabelSelector_Requirement) (*v1.LabelSelectorRequirement, error) {
	op, ok := roxToK8sOpMap[req.GetOp()]
	if !ok {
		return nil, fmt.Errorf("label selector operator %v not supported in Kubernetes", req.GetOp())
	}

	return &v1.LabelSelectorRequirement{
		Key:      req.GetKey(),
		Operator: op,
		Values:   req.GetValues(),
	}, nil
}

// ToRoxLabelSelector converts a Kubernetes LabelSelector into the corresponding StackRox protobuf representation.
func ToRoxLabelSelector(sel *v1.LabelSelector) (*storage.LabelSelector, error) {
	if sel == nil {
		return nil, nil
	}

	var roxReqs []*storage.LabelSelector_Requirement
	if sel.MatchExpressions != nil {
		roxReqs = make([]*storage.LabelSelector_Requirement, len(sel.MatchExpressions))
		for i := range sel.MatchExpressions {
			k8sReq := sel.MatchExpressions[i]
			roxReq, err := ToRoxLabelSelectorRequirement(&k8sReq)
			if err != nil {
				return nil, errors.Wrap(err, "converting requirement")
			}
			roxReqs[i] = roxReq
		}
	}
	return &storage.LabelSelector{
		MatchLabels:  sel.MatchLabels,
		Requirements: roxReqs,
	}, nil
}

// ToRoxLabelSelectorRequirement converts a Kubernetes LabelSelectorRequirement into the corresponding StackRox
// protobuf representation.
func ToRoxLabelSelectorRequirement(req *v1.LabelSelectorRequirement) (*storage.LabelSelector_Requirement, error) {
	op, ok := k8sToRoxOpMap()[req.Operator]
	if !ok {
		return nil, fmt.Errorf("label selector operator %v not supported by StackRox", req.Operator)
	}

	return &storage.LabelSelector_Requirement{
		Key:    req.Key,
		Op:     op,
		Values: req.Values,
	}, nil
}
