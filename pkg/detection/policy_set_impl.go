package detection

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/maputil"
)

type setImpl struct {
	policyIDToCompiled *maputil.FastRMap[string, CompiledPolicy]
}

func (p *setImpl) ForEach(f func(policy CompiledPolicy) error) error {
	m := p.policyIDToCompiled.GetMap()

	errList := errorhelpers.NewErrorList("policy evaluation")
	for _, compiled := range m {
		if err := f(compiled); err != nil {
			errList.AddError(err)
		}
	}
	return errList.ToError()
}

func (p *setImpl) ForOne(pID string, f func(CompiledPolicy) error) error {
	compiled, exists := p.policyIDToCompiled.Get(pID)
	if exists {
		return f(compiled)
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *storage.Policy) error {
	compiled, err := CompilePolicy(policy)
	if err != nil {
		log.Errorf("unable to compile policy: %s", err)
		return err
	}

	p.policyIDToCompiled.Set(compiled.Policy().GetId(), compiled)
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) {
	p.policyIDToCompiled.Delete(policyID)
}

// GetCompiledPolicies returns all of the compiled policies
func (p *setImpl) GetCompiledPolicies() map[string]CompiledPolicy {
	return p.policyIDToCompiled.GetMap()
}

// Exists returns if the specific policy id exists in the set
func (p *setImpl) Exists(id string) bool {
	_, exists := p.policyIDToCompiled.Get(id)
	return exists
}
