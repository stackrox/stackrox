package detection

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
)

// PolicyCompiler compiles policies to CompiledPolicy objects.
//go:generate mockgen-wrapper PolicyCompiler
type PolicyCompiler interface {
	CompilePolicy(policy *storage.Policy) (CompiledPolicy, error)
}

// NewPolicyCompiler returns a new instance of PolicyCompiler using the input MatcherBuilder to build matchers.
func NewPolicyCompiler(matcherBuilder matcher.Builder) PolicyCompiler {
	return &policyCompilerImpl{
		matcherBuilder: matcherBuilder,
	}
}

type policyCompilerImpl struct {
	matcherBuilder matcher.Builder
}

// CompilePolicy returns a new instance of CompiledPolicy a build from the input policy.
func (pc *policyCompilerImpl) CompilePolicy(policy *storage.Policy) (CompiledPolicy, error) {
	cloned := protoutils.CloneStoragePolicy(policy)
	compiledMatcher, err := pc.matcherBuilder.ForPolicy(cloned)
	if err != nil {
		return nil, err
	}

	return NewCompiledPolicy(cloned, compiledMatcher)
}
