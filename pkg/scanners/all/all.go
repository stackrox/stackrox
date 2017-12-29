package all

// Import all plugins so that they will be included in the available Scanners.
import (
	// Tenable
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/tenable"
	// DTR
	_ "bitbucket.org/stack-rox/apollo/pkg/scanners/dtr"
)
