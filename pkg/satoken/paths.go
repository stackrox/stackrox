package satoken

const (
	// ServiceAccountTokenDir is the directory into which the secret data from the Kubernetes service account token is
	// mounted.
	//#nosec G101 -- This is a false positive
	ServiceAccountTokenDir = `/run/secrets/kubernetes.io/serviceaccount`
	// ServiceAccountTokenJWTPath is the path of the file containing the Kubernetes service account JWT.
	ServiceAccountTokenJWTPath = ServiceAccountTokenDir + `/token`
	// ServiceAccountTokenNamespacePath is the path of the file containing the Kubernetes service account namespace.
	ServiceAccountTokenNamespacePath = ServiceAccountTokenDir + `/namespace`
)
