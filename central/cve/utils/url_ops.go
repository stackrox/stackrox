package utils

import (
	"net/url"
)

// QueryParam represents query params in a UR
type QueryParam struct {
	Key   string
	Value string
}

// GetURLWithQueryParams returns URL with query params
func GetURLWithQueryParams(baseURL string, queryParams []QueryParam) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	for _, queryParam := range queryParams {
		q.Add(queryParam.Key, queryParam.Value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
