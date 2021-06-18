package mtls

type issueOptions struct {
	namespace string
}

func (o *issueOptions) apply(opts []IssueCertOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// IssueCertOption is an additional option for certificate issuance.
type IssueCertOption func(o *issueOptions)

// WithNamespace requests certificates to be issued for the given namespace.
func WithNamespace(namespace string) IssueCertOption {
	return func(o *issueOptions) {
		o.namespace = namespace
	}
}
