//go:build roxagent

package secretinformer

// SecretInformer is a stub for roxagent builds.
// Roxagent runs in VMs and doesn't need Kubernetes secret watching.
type SecretInformer struct{}

// NewSecretInformer creates a stub secret informer that does nothing.
// This stub is used when building roxagent to avoid pulling in k8s.io dependencies.
// All parameters are interface{} to avoid importing k8s.io types.
func NewSecretInformer(
	_ string,
	_ string,
	_ interface{}, // k8sClient
	_ interface{}, // onAddFn
	_ interface{}, // onUpdateFn
	_ interface{}, // onDeleteFn
) *SecretInformer {
	return &SecretInformer{}
}

// Start is a no-op stub.
func (c *SecretInformer) Start() error {
	return nil
}

// Stop is a no-op stub.
func (c *SecretInformer) Stop() {}

// HasSynced always returns true in stub.
func (c *SecretInformer) HasSynced() bool {
	return true
}
