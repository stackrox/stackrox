package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/metrics"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

type setImpl struct {
	lock               sync.RWMutex
	policyStore        policyDatastore.DataStore
	compiler           PolicyCompiler
	policyIDToCompiled map[string]CompiledPolicy
}

func (p *setImpl) Compiler() PolicyCompiler {
	return p.compiler
}

func (p *setImpl) ForEach(pt PolicyExecutor) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, compiled := range p.policyIDToCompiled {
		t := time.Now()
		if err := pt.Execute(compiled); err != nil {
			return err
		}
		metrics.SetPolicyEvaluationDurationTime(t, compiled.Policy().GetName())
	}
	return nil
}

func (p *setImpl) ForOne(pID string, pt PolicyExecutor) error {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if compiled, exists := p.policyIDToCompiled[pID]; exists {
		return pt.Execute(compiled)
	}
	return fmt.Errorf("policy with ID not found in set: %s", pID)
}

// UpsertPolicy adds or updates a policy in the set.
func (p *setImpl) UpsertPolicy(policy *storage.Policy) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	compiled, err := p.compiler.CompilePolicy(policy)
	if err != nil {
		log.Errorf("unable to compile policy: %s", err)
		return err
	}

	p.policyIDToCompiled[compiled.Policy().GetId()] = compiled
	return nil
}

func (p *setImpl) Recompile(policyID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	olcCompiled, exists := p.policyIDToCompiled[policyID]
	if !exists {
		return fmt.Errorf("policy %s does not exist to recompile", policyID)
	}

	newCompiled, err := p.compiler.CompilePolicy(olcCompiled.Policy())
	if err != nil {
		log.Errorf("unable to compile policy: %s", err)
		return err
	}

	p.policyIDToCompiled[newCompiled.Policy().GetId()] = newCompiled
	return nil
}

// RemovePolicy removes a policy from the set.
func (p *setImpl) RemovePolicy(policyID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.policyIDToCompiled, policyID)
	return nil
}

// RemoveNotifier removes a given notifier from any policies in the set that use it.
func (p *setImpl) RemoveNotifier(notifierID string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, compiled := range p.policyIDToCompiled {
		policy := compiled.Policy()

		filtered := policy.GetNotifiers()[:0]
		for _, n := range policy.GetNotifiers() {
			if n != notifierID {
				filtered = append(filtered, n)
			}
		}
		policy.Notifiers = filtered

		err := p.policyStore.UpdatePolicy(context.TODO(), policy)
		if err != nil {
			return err
		}
	}

	return nil
}
