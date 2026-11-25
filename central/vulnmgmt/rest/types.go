package rest

// CVEListItem represents a single CVE in the list response.
type CVEListItem struct {
	CVE                        string  `json:"cve"`
	TopCVSS                    float32 `json:"topCvss"`
	TopNvdCVSS                 float32 `json:"topNvdCvss"`
	TopEPSSProbability         float32 `json:"topEpssProbability"`
	AffectedImageCount         int     `json:"affectedImageCount"`
	FirstDiscoveredInSystem    string  `json:"firstDiscoveredInSystem,omitempty"`
	PublishedOn                string  `json:"publishedOn,omitempty"`
	PendingExceptionCount      int32   `json:"pendingExceptionCount"`
	KnownExploit               bool    `json:"knownExploit"`
	KnownRansomwareCampaignUse bool    `json:"knownRansomwareCampaignUse"`
}

// CVEListResponse is the response for the CVE list endpoint.
type CVEListResponse struct {
	Items []CVEListItem `json:"items"`
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

// ImageName is the display name of an image.
type ImageName struct {
	Registry string `json:"registry"`
	Remote   string `json:"remote"`
	Tag      string `json:"tag"`
	FullName string `json:"fullName"`
}

// SignatureVerificationResult is one signature verification outcome for an image.
type SignatureVerificationResult struct {
	Description             string   `json:"description,omitempty"`
	Status                  string   `json:"status"`
	VerificationTime        string   `json:"verificationTime,omitempty"`
	VerifiedImageReferences []string `json:"verifiedImageReferences"`
	VerifierID              string   `json:"verifierId"`
	VerifierName            string   `json:"verifierName"`
}

// ImageSignatureVerificationData holds signature verification results for an image.
type ImageSignatureVerificationData struct {
	Results []SignatureVerificationResult `json:"results"`
}

// ImageListItem represents a single image in the list response.
type ImageListItem struct {
	ID                        string                          `json:"id"`
	Digest                    string                          `json:"digest"`
	Name                      ImageName                       `json:"name"`
	OperatingSystem           string                          `json:"operatingSystem"`
	ScanTime                  string                          `json:"scanTime,omitempty"`
	CreatedAt                 string                          `json:"createdAt,omitempty"`
	ScanNotes                 []string                        `json:"scanNotes"`
	Notes                     []string                        `json:"notes"`
	WatchStatus               string                          `json:"watchStatus"`
	DeploymentCount           int                             `json:"deploymentCount"`
	MatchedCVECount           int                             `json:"matchedCveCount"`
	SignatureVerificationData *ImageSignatureVerificationData `json:"signatureVerificationData,omitempty"`
}

// ImageListResponse is the response for the image list endpoint.
type ImageListResponse struct {
	Items []ImageListItem `json:"items"`
}

// DeploymentListItem represents a single deployment in the list response.
type DeploymentListItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Cluster         string `json:"cluster"`
	Namespace       string `json:"namespace"`
	CreatedAt       string `json:"createdAt,omitempty"`
	ImageCount      int    `json:"imageCount"`
	MatchedCVECount int    `json:"matchedCveCount"`
}

// DeploymentListResponse is the response for the deployment list endpoint.
type DeploymentListResponse struct {
	Items []DeploymentListItem `json:"items"`
}

// SeverityCounts holds CVE counts per severity.
type SeverityCounts struct {
	Critical  int `json:"critical"`
	Important int `json:"important"`
	Moderate  int `json:"moderate"`
	Low       int `json:"low"`
	Unknown   int `json:"unknown"`
}

// CountsResponse holds the entity and severity tab counts.
type CountsResponse struct {
	CVE        int            `json:"cve"`
	Image      int            `json:"image"`
	Deployment int            `json:"deployment"`
	Severity   SeverityCounts `json:"severity"`
}
