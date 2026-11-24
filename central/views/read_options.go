package views

// ReadOptions provide functionality to skip reading specific fields. This can be used avoid reading fields that are not required.
type ReadOptions struct {
	SkipGetImagesBySeverity        bool
	SkipGetTopCVSS                 bool
	SkipGetTopNVDCVSS              bool
	SkipGetAffectedImages          bool
	SkipGetFirstDiscoveredInSystem bool
	SkipPublishedDate              bool
	// UseCVENameFiltering makes the two-phase pagination fetch the filtered page by
	// CVE name instead of by row ID. It avoids the jsonb_agg aggregation of row IDs in
	// phase 1, at the cost of phase 2 potentially seeing rows from a concurrently
	// written scan. Callers that rely on stable row IDs (for example to load full
	// protos afterwards) must leave this false.
	UseCVENameFiltering bool
	// IncludeExploitData adds known-exploit and known-ransomware-campaign flags to the
	// CVE rows. It costs two filtered counts and is intended for list consumers that
	// render those labels.
	IncludeExploitData bool
}

// IsDefault returns true if all readoptions are set to default/false.
func (r *ReadOptions) IsDefault() bool {
	return !r.SkipGetImagesBySeverity &&
		!r.SkipGetTopCVSS &&
		!r.SkipGetAffectedImages &&
		!r.SkipGetFirstDiscoveredInSystem &&
		!r.UseCVENameFiltering &&
		!r.IncludeExploitData
}
