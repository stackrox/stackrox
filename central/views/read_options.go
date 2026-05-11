package views

// ReadOptions provide functionality to skip reading specific fields. This can be used avoid reading fields that are not required.
type ReadOptions struct {
	SkipGetImagesBySeverity        bool
	SkipGetTopCVSS                 bool
	SkipGetTopNVDCVSS              bool
	SkipGetAffectedImages          bool
	SkipGetFirstDiscoveredInSystem bool
	SkipPublishedDate              bool

	// ExcludeImagesWithActiveDeployments filters out images that are referenced
	// by at least one deployment in DEPLOYMENT_STATE_ACTIVE. Used by the
	// "Inactive images" view to avoid counting images that still have active
	// workloads.
	ExcludeImagesWithActiveDeployments bool
}

// IsDefault returns true if all readoptions are set to default/false.
func (r *ReadOptions) IsDefault() bool {
	return !r.SkipGetImagesBySeverity &&
		!r.SkipGetTopCVSS &&
		!r.SkipGetAffectedImages &&
		!r.SkipGetFirstDiscoveredInSystem
}
