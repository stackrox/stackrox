package apiparams

// SaveAsCustomResourcesRequest represents the API params to the endpoint for save policies as custom resources
type SaveAsCustomResourcesRequest struct {
	IDs []string `json:"ids"`
}

// SbomRequestBody represents the HTTP API request for generating an SBOM from an image scan.
type SbomRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"imageName"`
	Force     bool   `json:"force"`
}
