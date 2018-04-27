package risk

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/scopecomp"
)

// userDefinedMultiplier is a wrapper around a proto multiplier
type userDefinedMultiplier struct {
	*v1.Multiplier
}

// newUserDefinedMultiplier generates a new wrapper around the proto multiplier that implements the generic multiplier interface
func newUserDefinedMultiplier(mult *v1.Multiplier) multiplier {
	return &userDefinedMultiplier{
		Multiplier: mult,
	}
}

func formatScope(scope *v1.Scope) string {
	var vals []string
	if scope.GetCluster() != "" {
		vals = append(vals, "cluster:"+scope.GetCluster())
	}
	if scope.GetNamespace() != "" {
		vals = append(vals, "namespace:"+scope.GetNamespace())
	}
	if scope.GetLabel() != nil {
		vals = append(vals, "label:"+scope.GetLabel().GetKey()+"="+scope.GetLabel().GetValue())
	}
	return strings.Join(vals, " ")
}

// Score returns a risk result
func (u *userDefinedMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	if !scopecomp.WithinScope(u.GetScope(), deployment) {
		return nil
	}
	return &v1.Risk_Result{
		Name:  u.GetName(),
		Score: u.GetValue(),
		Factors: []string{
			fmt.Sprintf("Deployment matched scope '%s'", formatScope(u.GetScope())),
		},
	}
}
