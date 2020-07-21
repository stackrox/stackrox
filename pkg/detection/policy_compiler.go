package detection

import (
	"github.com/stackrox/rox/generated/storage"
)

// CompilePolicy compiles the given policy, making it ready for matching.
func CompilePolicy(policy *storage.Policy) (CompiledPolicy, error) {
	cloned := policy.Clone()
	return newCompiledPolicy(cloned)
}
