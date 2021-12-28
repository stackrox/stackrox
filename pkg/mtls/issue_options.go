package mtls

type issueOptions struct {
	namespace     string
	signerProfile string
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

// WithEphemeralValidity requests certificates with short validity.
// This option is suitable for issuing init bundles which cannot be revoked.
func WithEphemeralValidity() IssueCertOption {
	return func(o *issueOptions) {
		o.signerProfile = ephemeralProfile
	}
}

// WithLocalScannerProfile requests certificates using the local scanner profile.
func WithLocalScannerProfile() IssueCertOption {
	return func(o *issueOptions) {
		o.signerProfile = localScannerProfile
	}
}
