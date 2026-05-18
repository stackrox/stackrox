package generate

// Deprecated: istioSupportWrapper is kept for backward compatibility with the deprecated --istio-support flag.
type istioSupportWrapper struct {
	istioSupport *string
}

func (w istioSupportWrapper) String() string {
	return *w.istioSupport
}

func (w istioSupportWrapper) Set(input string) error {
	*w.istioSupport = input
	return nil
}

func (w istioSupportWrapper) Type() string {
	return "version"
}
