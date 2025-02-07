package apiparams

// SBOMRequestBody represents the HTTP API request for generating an SBOM from an image scan.
type SBOMRequestBody struct {
	// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
	// Cluster   string `json:"cluster"`
	ImageName string `json:"imageName"`
	Force     bool   `json:"force"`
}
