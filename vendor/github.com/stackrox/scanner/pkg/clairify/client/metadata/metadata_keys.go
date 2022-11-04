package metadata

import (
	"github.com/stackrox/scanner/ext/vulnmdsrc/nvd"
	"github.com/stackrox/scanner/ext/vulnmdsrc/redhat"
)

const (
	// NVD represents the key to get NVD data in the vuln Metadata
	NVD = nvd.AppenderName
	// RedHat represents the key to get Red Hat data in the vuln Metadata
	RedHat = redhat.AppenderName
)
