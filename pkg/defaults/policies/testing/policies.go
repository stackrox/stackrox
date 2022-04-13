package testing

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/defaults/policies"
)

// GetDefaultPolicy returns a default policy by its name
func GetDefaultPolicy(t *testing.T, name string) (*storage.Policy, error) {
	if t == nil {
		panic("This function must be called inside a test.")
	}

	policies, err := policies.DefaultPolicies()
	if err != nil {
		return nil, err
	}

	for _, p := range policies {
		if p.GetName() == name {
			return p, nil
		}
	}
	return nil, errors.Errorf("Could not find default policy: %q", name)
}
