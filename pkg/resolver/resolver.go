package resolver

import "github.com/stackrox/rox/pkg/urlfmt"

// Registry resolves a registry into its fully qualified form
func Registry(url string) string {
	switch url {
	case "docker.io":
		return "https://registry-1.docker.io"
	default:
		return urlfmt.FormatURL(url, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	}
}
