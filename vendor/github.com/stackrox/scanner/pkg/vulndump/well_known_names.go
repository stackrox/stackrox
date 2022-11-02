package vulndump

import (
	"time"

	"github.com/stackrox/scanner/database"
)

// This block enumerates the files/directories in the vuln dump.
// The vuln dump itself is a zip with all these directories.
const (
	ManifestFileName      = "manifest.json"
	OSVulnsFileName       = "os_vulns.json"
	RHELv2DirName         = "rhelv2"
	RHELv2VulnsSubDirName = "vulns"
	NVDDirName            = "nvd"
	RedHatDirName         = "redhat"
	K8sDirName            = "k8s"
)

// Manifest is used to JSON marshal/unmarshal the manifest.json file.
type Manifest struct {
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

// RHELv2 is used to JSON marshal/unmarshal each RHELv2 JSON file.
type RHELv2 struct {
	LastModified time.Time                       `json:"last_modified"`
	Vulns        []*database.RHELv2Vulnerability `json:"vulns"`
}
