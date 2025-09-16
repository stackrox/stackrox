package scannerv4

import (
	"net/url"

	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultVersion represents the default version string that will be used
	// when setting the Scanner V4 version if no version was provided by the
	// gRPC metadata.
	DefaultVersion = "v4"
)

// Version contains version information about a particular instance or flow of
// communication with a Scanner V4 service.
type Version struct {
	Indexer string `mapstructure:"indexer"`
	Matcher string `mapstructure:"matcher"`
}

// Encode converts a Version into a URI query string.
// The error case is unlikely and can mostly be used as a program assertion.
func (v Version) Encode() (string, error) {
	var uv url.Values
	if err := mapstructure.Decode(v, &uv); err != nil {
		return "", err
	}
	return uv.Encode(), nil
}

// DecodeVersion converts a URI query of the Scanner V4 version information
// into a Version. Returns an error if the string isn't a URI query or if the
// query string values couldn't be decoded into the Version object. The latter
// is unlikely and can mostly be used as a program assertion.
func DecodeVersion(version string) (Version, error) {
	uv, err := url.ParseQuery(version)
	if err != nil {
		return Version{}, err
	}

	// The `Values` type provided by net/url is a map under the hood:
	// `type Values map[string][]string`. Therefore, we can use mapstructure to
	// decode the values in a *Version.
	var v Version
	if err := mapstructure.Decode(uv, &v); err != nil {
		return Version{}, err
	}

	return v, nil
}
