package multipliers

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// userDefinedMultiplier is a wrapper around a proto multiplier
type userDefinedMultiplier struct {
	*storage.Multiplier
}

// NewUserDefined generates a new wrapper around the proto multiplier that implements the generic multiplier interface
func NewUserDefined(mult *storage.Multiplier) Multiplier {
	return &userDefinedMultiplier{
		Multiplier: mult,
	}
}

// Score returns a risk result
func (u *userDefinedMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	if !scopecomp.WithinScope(u.GetScope(), deployment) {
		return nil
	}
	return &v1.Risk_Result{
		Name:  u.GetName(),
		Score: u.GetValue(),
		Factors: []*v1.Risk_Result_Factor{
			{Message: fmt.Sprintf("Deployment matched scope '%s'", formatScope(u.GetScope()))},
		},
	}
}

func formatScope(scope *storage.Scope) string {
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
