package dnrintegration

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const servicesEndpoint = "v1/services"

//////////////////////////////////////////////////////////////////////
// The following are D&R alert types that have been copy-pasted here,
// for the purposes of JSON unmarshaling.
// Only the fields that Prevent cares about are included here.
/////////////////////////////////////////////////////////////////////

type serviceList struct {
	Results []service `json:"results"`
}

type service struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Labels    []string `json:"labels"`
}

func (d *dnrIntegrationImpl) Services(params url.Values) ([]service, error) {
	bytes, err := d.makeAuthenticatedRequest("GET", servicesEndpoint, params)
	if err != nil {
		return nil, fmt.Errorf("making services request: %s", err)
	}
	var serviceList serviceList
	err = json.Unmarshal(bytes, &serviceList)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling services struct: %s", err)
	}

	return serviceList.Results, nil
}
