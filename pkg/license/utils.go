package license

import (
	"net/url"
)

// IDAsURLParam returns the license ID as a URL query parameter.
func IDAsURLParam(licenseID string) url.Values {
	return url.Values{
		"license_id": []string{licenseID},
	}
}
