package resolver

import "bitbucket.org/stack-rox/apollo/pkg/urlfmt"

// Registry resolves a registry into its fully qualified form
func Registry(url string, insecure bool) string {
	switch url {
	case "docker.io":
		return "https://registry-1.docker.io"
	default:
		val, err := urlfmt.FormatURL(url, !insecure, false)
		if err != nil {
			return url
		}
		return val
	}
}
