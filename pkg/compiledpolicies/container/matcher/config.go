package matcher

import (
	"github.com/stackrox/rox/generated/storage"
	configMatcher "github.com/stackrox/rox/pkg/compiledpolicies/containerconfig/matcher"
)

func init() {
	compilers = append(compilers, newConfigMatcher)
}

func newConfigMatcher(policy *storage.Policy) (Matcher, error) {
	matcher, err := configMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *storage.Container) []*storage.Alert_Violation {
		return matcher(container.GetConfig())
	}, nil
}
