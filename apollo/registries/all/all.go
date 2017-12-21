package all

import (
	// Import the docker plugin so that it'll be included in the available Registries
	_ "bitbucket.org/stack-rox/apollo/apollo/registries/docker"
	// Import the tenable plugin so that it'll be included in the available Registries
	_ "bitbucket.org/stack-rox/apollo/apollo/registries/tenable"
)
