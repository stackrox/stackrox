package features

var (
	// HtpasswdAuth controls whether authentication based on
	// htpasswd-formatted files should be used.
	// This flag will be removed when the feature is complete.
	HtpasswdAuth = registerFeature("Htpasswd Authentication", "ROX_HTPASSWD_AUTH", true)
)
