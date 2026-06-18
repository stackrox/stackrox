package satoken

import "github.com/stackrox/rox/pkg/env"

var (
	//#nosec G101 -- This is a false positive
	serviceAccountTokenDirSetting = env.RegisterSetting("ROX_SA_TOKEN_DIR",
		env.WithDefault("/run/secrets/kubernetes.io/serviceaccount"))
)

// ServiceAccountTokenDir returns the directory into which the secret data from the Kubernetes
// service account token is mounted.
func ServiceAccountTokenDir() string {
	return serviceAccountTokenDirSetting.Setting()
}

// ServiceAccountTokenJWTPath returns the path of the file containing the Kubernetes service account JWT.
func ServiceAccountTokenJWTPath() string {
	return ServiceAccountTokenDir() + "/token"
}

// ServiceAccountNamespacePath returns the path of the file containing the Kubernetes namespace.
func ServiceAccountNamespacePath() string {
	return ServiceAccountTokenDir() + "/namespace"
}
