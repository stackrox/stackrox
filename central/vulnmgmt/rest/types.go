package rest

// CVEListItem represents a single CVE in the list response.
type CVEListItem struct {
	CVE                     string  `json:"cve"`
	TopSeverity             string  `json:"topSeverity"`
	TopCVSS                 float32 `json:"topCvss"`
	TopNvdCVSS              float32 `json:"topNvdCvss"`
	TopEPSSProbability      float32 `json:"topEpssProbability"`
	AffectedImageCount      int     `json:"affectedImageCount"`
	FirstDiscoveredInSystem string  `json:"firstDiscoveredInSystem,omitempty"`
	PublishedOn             string  `json:"publishedOn,omitempty"`
	PendingExceptionCount   int32   `json:"pendingExceptionCount"`
}

// CVEListResponse is the response for the CVE list endpoint.
type CVEListResponse struct {
	Items      []CVEListItem `json:"items"`
	TotalCount int           `json:"totalCount"`
}

// CVEDetailItem represents a single distro detail for a CVE.
type CVEDetailItem struct {
	CVE             string `json:"cve"`
	Summary         string `json:"summary"`
	Link            string `json:"link"`
	OperatingSystem string `json:"operatingSystem"`
}

// CVEDetailResponse is the response for the CVE detail endpoint.
type CVEDetailResponse struct {
	DistroDetails []CVEDetailItem `json:"distroDetails"`
}
