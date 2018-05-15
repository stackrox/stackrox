package all

// Import all plugins so that they will be included in the available Registries.
import (
	// Tenable
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/tenable"
	// Docker
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/docker"
	// Docker Trusted Registry
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/dtr"
	// Google
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/google"
	// Quay
	_ "bitbucket.org/stack-rox/apollo/pkg/registries/quay"
)
