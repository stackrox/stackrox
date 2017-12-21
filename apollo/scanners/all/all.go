package all

import (
	// Import the dtr plugin so that it'll be included in the available Scanners
	_ "bitbucket.org/stack-rox/apollo/apollo/scanners/dtr"
	// Import the tenable plugin so that it'll be included in the available Scanners
	_ "bitbucket.org/stack-rox/apollo/apollo/scanners/tenable"
)
