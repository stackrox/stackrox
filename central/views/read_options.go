package views

// ReadOptions provide functionality to skip reading specific fields. This can be used avoid reading fields that are not required.
type ReadOptions struct {
	SkipGetImagesBySeverity        bool
	SkipGetTopCVSS                 bool
	SkipGetAffectedImages          bool
	SkipGetFirstDiscoveredInSystem bool
}
