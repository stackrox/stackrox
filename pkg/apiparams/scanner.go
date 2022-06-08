// Package apiparams contains type definitions for parameters for APIs which are served only through HTTP,
// and thus are never serialized as protocol buffers.
package apiparams

// Scanner represents the API parameters to the API in Central that generates
// a scanner bundle.
type Scanner struct {
	ClusterType    string `json:"clusterType"`
	ScannerImage   string `json:"scannerImage"`
	ScannerDBImage string `json:"scannerDBImage"`
	OfflineMode    bool   `json:"offlineMode"`

	IstioVersion string `json:"istioVersion"`

	DisablePodSecurityPolicies bool `json:"disablePodSecurityPolicies"`
}
