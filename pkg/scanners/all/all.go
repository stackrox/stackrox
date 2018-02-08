package all

// Import all plugins so that they will be included in the available Scanners.
import (
	// Clair
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/clair"
	// DTR
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/dtr"
	// Tenable
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/tenable"
	// Quay
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/quay"
)
