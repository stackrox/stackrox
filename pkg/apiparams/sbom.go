package apiparams

// SBOMRequestBody represents the HTTP API request for generating an SBOM from an image scan.
// Any changes to this struct should be reflected in the central/docs/api_custom_routes/image_service_swagger.yaml
type SBOMRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"imageName"`
	Force     bool   `json:"force"`
}
