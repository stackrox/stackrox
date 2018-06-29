package dnrintegration

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const versionEndpoint = "v1/version"

type versionResponse struct {
	Version string `json:"version"`
}

func (d *dnrIntegrationImpl) version() (string, error) {
	bytes, err := d.makeAuthenticatedRequest("GET", versionEndpoint, url.Values{})
	if err != nil {
		return "", err
	}
	versionResponse := versionResponse{}
	err = json.Unmarshal(bytes, &versionResponse)
	if err != nil {
		return "", fmt.Errorf("unmarshalling version JSON: %s", err)
	}
	return versionResponse.Version, nil
}
