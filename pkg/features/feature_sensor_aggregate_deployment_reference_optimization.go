package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// SensorAggregateDeploymentReferenceOptimization enables a performance improvement by aggregating deployment references when the same reference is queued for processing
var SensorAggregateDeploymentReferenceOptimization = registerFeature("Enables a performance improvement by aggregating deployment references when the same reference is queued for processing", "ROX_AGGREGATE_DEPLOYMENT_REFERENCE_OPTIMIZATION")
