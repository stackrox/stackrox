package apiparams

// SBOMRequestBody represents the HTTP API request for generating an SBOM from an image scan.
type SBOMRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"imageName"`
	Force     bool   `json:"force"`
}
