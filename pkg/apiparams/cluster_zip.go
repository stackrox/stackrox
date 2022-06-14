package apiparams

// ClusterZip represents the API params to the endpoint for downloading a cluster ZIP.
type ClusterZip struct {
	ID               string `json:"id"`
	CreateUpgraderSA *bool  `json:"createUpgraderSA"`
	SlimCollector    *bool  `json:"slimCollector"`
	IstioVersion     string `json:"istioVersion"`

	DisablePodSecurityPolicies bool `json:"disablePodSecurityPolicies"`
}
