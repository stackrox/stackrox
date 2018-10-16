package features

import "os"

var (
	// HtpasswdAuth controls whether authentication based on
	// htpasswd-formatted files should be used.
	// This flag will be removed when the feature is complete.
	HtpasswdAuth = htpasswdAuth{}
)

type htpasswdAuth struct{}

func (htpasswdAuth) Name() string {
	return "Htpasswd Authentication"
}
func (h htpasswdAuth) Enabled() bool {
	return isEnabled(os.Getenv(h.EnvVar()), false)
}

func (htpasswdAuth) EnvVar() string {
	return "ROX_HTPASSWD_AUTH"
}
