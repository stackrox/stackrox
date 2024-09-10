package proxy

const (
	// ProxyConfigFailedStatusType is the status condition type to indicate that the proxy configuration could not be
	// applied.
	ProxyConfigFailedStatusType = `ProxyConfigFailed`
)

// The following are the valid reasons for the ProxyConfigFailedStatusType condition.
const (
	//#nosec G101 -- This is a false positive
	SecretReconcileErrorReason = `ProxyConfigSecretReconcileError`
	NoProxyConfigReason        = `NoProxyConfig`
	ProxyConfigAppliedReason   = `ProxyConfigApplied`
)
