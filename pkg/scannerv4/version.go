package scannerv4

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
)

const (
	// ServiceVersionHeader is the key for the gRPC metadata (usually a header)
	// that contains the Scanner version the caller is communicating with.
	ServiceVersionHeader = "X-Scanner-Version"

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
	// mapstructure doesn't have a convenient way to decode struct string
	// fields into a map []string value. Instead, decode Version into a
	// map[string]string first, copy the values over to a url.Values object,
	// and encode that. This also has the added benefit of ignoring empty
	// string fields in Version during the map[string]string -> url.Values
	// copy.
	var decodeMap map[string]string
	if err := mapstructure.Decode(v, &decodeMap); err != nil {
		return "", fmt.Errorf("decoding version to string map: %w", err)
	}

	uv := url.Values{}
	for k, v := range decodeMap {
		if v != "" {
			uv.Set(k, v)
		}
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
		return Version{}, fmt.Errorf("parsing url query string: %w", err)
	}

	// mapstructure doesn't have a convenient way to decode map []string values
	// to struct string fields. Instead, copy all values over to a
	// map[string]string while joining all string slices to a single string,
	// then decode that map[string]string to a Version.
	encodeMap := make(map[string]string)
	for k, v := range uv {
		encodeMap[k] = strings.Join(v, ",")
	}

	var v Version
	if err := mapstructure.Decode(encodeMap, &v); err != nil {
		return Version{}, fmt.Errorf("decoding version from string map: %w", err)
	}

	return v, nil
}
