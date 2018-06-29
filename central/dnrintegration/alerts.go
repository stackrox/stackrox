package dnrintegration

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const alertsEndpoint = "v0.1/api/alerts/"

//////////////////////////////////////////////////////////////////////
// The following are D&R alert types that have been copy-pasted here,
// for the purposes of JSON unmarshaling.
// Only the fields that Prevent cares about are included here.
/////////////////////////////////////////////////////////////////////

// AlertList is used to return a list of results along with pagination information for D&R Alert API consumers.
type AlertList struct {
	Results []PolicyAlert `json:"results"`
}

// A PolicyAlert is a violation of a Policy Definition.
type PolicyAlert struct {
	PolicyName string `json:"policy_name"`

	SeverityWord  string  `json:"severity_word"`
	SeverityScore float64 `json:"severity_score"`
}

func (d *dnrIntegrationImpl) Alerts(namespace, serviceName string) ([]PolicyAlert, error) {
	params := url.Values{}
	// This makes sure we don't show Acknowledged or Resolved alerts.
	params.Add("workflowState", "New")
	// In D&R alerts, the "default" namespace translates to unset.
	if namespace != "" && namespace != "default" {
		params.Set("namespace", namespace)
	}
	if serviceName != "" {
		params.Set("serviceName", serviceName)
	}
	bytes, err := d.makeAuthenticatedRequest("GET", alertsEndpoint, params)
	if err != nil {
		return nil, fmt.Errorf("making alerts request: %s", err)
	}
	var alertList AlertList
	err = json.Unmarshal(bytes, &alertList)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling alerts struct: %s", err)
	}
	return alertList.Results, nil
}
