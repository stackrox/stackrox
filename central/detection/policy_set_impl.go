package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/metrics"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	policyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

type setImpl struct {
	policyStore        policyDatastore.DataStore
	compiler           PolicyCompiler
	policyIDToCompiled StringCompiledPolicyFastRMap
}

func (p *setImpl) Compiler() PolicyCompiler {
	return p.compiler
}

func (p *setImpl) ForEach(pt PolicyExecutor) error {
	m := p.policyIDToCompiled.GetMap()

	for _, compiled := range m {
		t := time.Now()
		if err := pt.Execute(compiled); err != nil {
			return err
		}
		metrics.SetPolicyEvaluationDurationTime(t, compiled.Policy().GetName())
	}
	return nil
}

func (p *setImpl) ForOne(pID string, pt PolicyExecutor) error {
	compiled, exists := p.policyIDToCompiled.Get(pID)
	if exists {
		return pt.Execute(compiled)
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *storage.Policy) error {
	compiled, err := p.compiler.CompilePolicy(policy)
	if err != nil {
		log.Errorf("unable to compile policy: %s", err)
		return err
	}

	p.policyIDToCompiled.Set(compiled.Policy().GetId(), compiled)
	return nil
}

func (p *setImpl) Recompile(policyID string) error {
	olcCompiled, exists := p.policyIDToCompiled.Get(policyID)
	if !exists {
		return fmt.Errorf("policy %s does not exist to recompile", policyID)
	}

	newCompiled, err := p.compiler.CompilePolicy(olcCompiled.Policy())
	if err != nil {
		log.Errorf("unable to compile policy: %s", err)
		return err
	}

	p.policyIDToCompiled.Set(newCompiled.Policy().GetId(), newCompiled)
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) error {
	p.policyIDToCompiled.Delete(policyID)
	return nil
}

// RemoveNotifier removes a given notifier from any policies in the set that use it.
func (p *setImpl) RemoveNotifier(notifierID string) error {
	m := p.policyIDToCompiled.GetMap()

	for _, compiled := range m {
		policy := compiled.Policy()

		filtered := policy.GetNotifiers()[:0]
		for _, n := range policy.GetNotifiers() {
			if n != notifierID {
				filtered = append(filtered, n)
			}
		}
		policy.Notifiers = filtered

		err := p.policyStore.UpdatePolicy(policyCtx, policy)
		if err != nil {
			return err
		}
	}

	return nil
}
