package matcher

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	configMatcher "github.com/stackrox/rox/pkg/compiledpolicies/containerconfig/matcher"
)

func init() {
	compilers = append(compilers, newConfigMatcher)
}

func newConfigMatcher(policy *v1.Policy) (Matcher, error) {
	matcher, err := configMatcher.Compile(policy)
	if err != nil {
		return nil, err
	} else if matcher == nil {
		return nil, nil
	}

	return func(container *storage.Container) []*v1.Alert_Violation {
		return matcher(container.GetConfig())
	}, nil
}
