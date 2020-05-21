package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
)

// PolicyCompiler compiles policies to CompiledPolicy objects.
//go:generate mockgen-wrapper
type PolicyCompiler interface {
	CompilePolicy(policy *storage.Policy) (CompiledPolicy, error)
}

// NewPolicyCompiler returns a new policy compiler.
func NewPolicyCompiler() PolicyCompiler {
	return &policyCompilerImpl{}
}

// NewLegacyPolicyCompiler returns a new instance of PolicyCompiler using the input MatcherBuilder to build matchers.
// Deprecated: This will go away once we get rid of searchbasedpolicies.
func NewLegacyPolicyCompiler(matcherBuilder matcher.Builder) PolicyCompiler {
	return &policyCompilerImpl{
		matcherBuilder: matcherBuilder,
	}
}

type policyCompilerImpl struct {
	matcherBuilder matcher.Builder
}

// CompilePolicy returns a new instance of CompiledPolicy a build from the input policy.
func (pc *policyCompilerImpl) CompilePolicy(policy *storage.Policy) (CompiledPolicy, error) {
	cloned := policy.Clone()
	var compiledMatcher searchbasedpolicies.Matcher
	if !features.BooleanPolicyLogic.Enabled() {
		var err error
		compiledMatcher, err = pc.matcherBuilder.ForPolicy(cloned)
		if err != nil {
			return nil, err
		}
	}

	return newCompiledPolicy(cloned, compiledMatcher)
}
