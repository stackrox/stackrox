package urlfmt

import (
	"net/url"
	"strings"
)

// FormatURL takes in an endpoint, whether to prepend https if no scheme is specified and if the url should end in a slash
func FormatURL(endpoint string, httpsDefault, endingSlash bool) (string, error) {
	if !strings.HasPrefix(endpoint, "http") {
		if httpsDefault {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	if endingSlash && !strings.HasSuffix(endpoint, "/") {
		return endpoint + "/", nil
	} else if !endingSlash {
		return strings.TrimRight(endpoint, "/"), nil
	}
	return endpoint, nil
}

// FullyQualifiedURL returns a URL in the proper format or returns an error if the format is invalid
func FullyQualifiedURL(endpoint string, values url.Values, args ...string) (string, error) {
	endpoint = strings.TrimRight(endpoint, "/")
	for i, s := range args {
		s = strings.TrimLeft(s, "/")
		s = strings.TrimRight(s, "/")
		args[i] = s
	}
	fullPath := strings.Join(append([]string{endpoint}, args...), "/")
	url, err := url.Parse(fullPath)
	if err != nil {
		return "", err
	}
	url.RawQuery = values.Encode()
	return url.String(), nil
}
