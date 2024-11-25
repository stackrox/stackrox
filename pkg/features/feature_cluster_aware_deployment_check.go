package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// ClusterAwareDeploymentCheck enables roxctl deployment check to check deployments on the cluster level.
var ClusterAwareDeploymentCheck = registerFeature("Enables cluster level check for the 'roxctl deployment check' command.", "ROX_CLUSTER_AWARE_DEPLOYMENT_CHECK", enabled)
