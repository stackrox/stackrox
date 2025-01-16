package phonehome

import (
	"strings"
)

// APICallCampaignCriterion defines a criterion for an API interception of a
// telemetry campaign. Requests parameters need to match all fields for the
// request to be tracked. Any request matches empty criterion.
type APICallCampaignCriterion struct {
	Paths   []string          `json:"paths,omitempty"`
	Methods []string          `json:"methods,omitempty"`
	Codes   []int32           `json:"codes,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// APICallCampaign defines an API interception telemetry campaign as a list of
// criteria to fulfil for an API call to be intercepted.
// A request should fulfil at least one of the criterion to be tracked.
type APICallCampaign []APICallCampaignCriterion

func (c *APICallCampaignCriterion) IsFulfilled(rp *RequestParams) bool {
	codeMatches := len(c.Codes) == 0
	for _, code := range c.Codes {
		if rp.Code == int(code) {
			codeMatches = true
			break
		}
	}

	methodMatches := len(c.Methods) == 0
	for _, method := range c.Methods {
		if strings.EqualFold(rp.Method, method) {
			methodMatches = true
			break
		}
	}

	return codeMatches && methodMatches &&
		(c.Paths == nil || rp.HasPathIn(c.Paths)) &&
		(c.Headers == nil || rp.HasHeader(c.Headers))
}

func (c APICallCampaign) IsFulfilled(rp *RequestParams) bool {
	for _, cc := range c {
		if cc.IsFulfilled(rp) {
			return true
		}
	}
	return false
}
