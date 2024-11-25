package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// ClusterRegistrationSecrets enables support for Cluster Registration Secrets (CRS), the next-gen init-bundles.
var ClusterRegistrationSecrets = registerFeature("Enable support for Cluster Registration Secrets (CRS)", "ROX_CLUSTER_REGISTRATION_SECRETS", enabled)
