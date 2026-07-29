package auth

// ArtifactRegistryScopes returns the OAuth2 scopes for GCP Artifact Registry.
func ArtifactRegistryScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/cloud-platform.read-only",
	}
}
