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

type Version struct {
	Indexer string `mapstructure:"indexer"`
	Matcher string `mapstructure:"matcher"`
}

func (v *Version) Encode() (string, error) {
	var uv url.Values
	if err := mapstructure.Decode(v, &uv); err != nil {
		return "", err
	}
	return uv.Encode(), nil
}

func DecodeVersion(version string) (*Version, error) {
	uv, err := url.ParseQuery(version)
	if err != nil {
		return nil, err
	}

	// The `Values` type provided by net/url is a map under the hood:
	// `type Values map[string][]string`. Therefore, we can use mapstructure to
	// decode the values in a *Version.
	var v Version
	if err := mapstructure.Decode(uv, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
