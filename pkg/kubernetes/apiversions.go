package kubernetes

import (
	"regexp"
	"strings"
)

var (
	apiVersionRegex = regexp.MustCompile(`.*\..*\..*`)
)

// IsNativeAPI returns true if the API is a native Kubernetes API
// and is either of the form v1, v1beta1/apps, or networking.k8s.io/networkpolicy
// It excludes resources from Istio and KNative
func IsNativeAPI(apiVersion string) bool {
	// Split for the cases like v1beta1/apps
	split := strings.SplitN(apiVersion, "/", 2)
	// If fewer than two segments, then return true because the apiVersion was
	// something like "v1"
	if len(split) != 2 {
		return true
	}

	url := split[0]
	// validate the first part of the api version against a regex checking if it is
	// a URL. If it is not, then it IS a native k8s API
	if !apiVersionRegex.MatchString(url) {
		return true
	}
	// return true if it has the native k8s suffix of .k8s.io
	return strings.HasSuffix(url, ".k8s.io") || strings.HasSuffix(url, ".openshift.io")
}
