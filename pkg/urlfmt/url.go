package urlfmt

import (
	"net/url"
	"strings"
)

// FormatURL takes in an endpoint, whether to prepend https if no scheme is specified and if the url should end in a slash
func FormatURL(endpoint string, httpsDefault, endingSlash bool) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		if httpsDefault {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
		return FormatURL(endpoint, httpsDefault, endingSlash)
	}

	if endingSlash && !strings.HasSuffix(endpoint, "/") {
		return endpoint + "/", nil
	} else if !endingSlash {
		return strings.TrimRight(endpoint, "/"), nil
	}
	return endpoint, nil
}
