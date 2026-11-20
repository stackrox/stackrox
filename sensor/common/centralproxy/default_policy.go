package centralproxy

import "k8s.io/client-go/kubernetes"

func newRequestPolicy(client kubernetes.Interface) (requestPolicy, error) {
	return &fixedClientPolicy{client: client}, nil
}
