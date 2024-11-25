package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// SensorDeploymentBuildOptimization enables a performance improvement by skipping deployments processing when no dependency or spec changed
var SensorDeploymentBuildOptimization = registerFeature("Enables a performance improvement by skipping deployments processing when no dependency or spec changed", "ROX_DEPLOYMENT_BUILD_OPTIMIZATION", enabled)
