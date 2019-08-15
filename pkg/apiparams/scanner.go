// Package apiparams contains type definitions for parameters for APIs which are served only through HTTP,
// and thus are never serialized as protocol buffers.
package apiparams

import (
	"github.com/stackrox/rox/pkg/renderer"
)

// Scanner represents the API parameters to the API in Central that generates
// a scanner bundle.
type Scanner struct {
	ClusterType      string                   `json:"clusterType"`
	ScannerImage     string                   `json:"scannerImage"`
	OfflineMode      bool                     `json:"offlineMode"`
	ScannerV2Config  renderer.ScannerV2Config `json:"scannerV2Config,omitempty"`
	ScannerV2Image   string                   `json:"scannerV2Image"`
	ScannerV2DBImage string                   `json:"scannerV2DBImage"`
}
