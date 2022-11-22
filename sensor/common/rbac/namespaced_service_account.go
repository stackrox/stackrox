package rbac

// NamespacedServiceAccount keeps a pair of service account and used namespace.
type NamespacedServiceAccount interface {
	GetServiceAccount() string
	GetNamespace() string
}
