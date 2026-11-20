package centralproxy

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

func newRequestPolicy(client kubernetes.Interface) (requestPolicy, error) {
	policy := NewClientPolicy(client)
	if err := policy.EnsureDefaults(context.Background()); err != nil {
		return nil, err
	}
	return policy, nil
}
