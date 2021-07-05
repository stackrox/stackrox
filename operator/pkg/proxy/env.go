package proxy

import (
	"os"
	"strings"
)

var (
	lowercaseProxyEnvVars = []string{"http_proxy", "https_proxy", "all_proxy", "no_proxy"}
)

// GetProxyEnvVars returns a map of proxy-relevant environment variables.
func GetProxyEnvVars() map[string]string {
	res := make(map[string]string, 2*len(lowercaseProxyEnvVars))

	for _, lcEnvVar := range lowercaseProxyEnvVars {
		val := os.Getenv(lcEnvVar)
		if val != "" {
			res[lcEnvVar] = val
		}

		ucEnvVar := strings.ToUpper(lcEnvVar)
		val = os.Getenv(ucEnvVar)
		if val != "" {
			res[ucEnvVar] = val
		}
	}

	if len(res) == 0 {
		return nil
	}
	return res
}
