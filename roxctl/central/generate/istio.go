package generate

// Deprecated: istioSupportWrapper is kept for backward compatibility with the deprecated --istio-support flag.
type istioSupportWrapper struct{}

func (w istioSupportWrapper) String() string {
	return ""
}

func (w istioSupportWrapper) Set(_ string) error {
	return nil
}

func (w istioSupportWrapper) Type() string {
	return "version"
}
