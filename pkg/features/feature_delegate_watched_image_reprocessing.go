package features

//lint:file-ignore U1000 we want to introduce this feature flag unused.

// DelegateWatchedImageReprocessing when set to enabled reprocessing of watched images may be delegated to secured clusters based
// on the delegated scanning config.
var DelegateWatchedImageReprocessing = registerFeature("Enables delegating scans for watched images during reprocessing", "ROX_DELEGATE_WATCHED_IMAGE_REPROCESSING", enabled)
