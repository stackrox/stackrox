package all

// Import all plugins so that they will be included in the available auth providers.
import (
	// Auth0
	_ "github.com/stackrox/rox/pkg/auth/authproviders/auth0"
)
