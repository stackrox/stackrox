package phonehome

import (
	"strings"
)

// APICallCampaignCriterium defines a criterium for an API interception of a telemetry
// campaign. Requests parameters need to match all fields for the request to
// be tracked. Any request matches empty criterium.
type APICallCampaignCriterium struct {
	UserAgents   []string `json:"user_agents,omitempty"`
	PathPatterns []string `json:"path_patterns,omitempty"`
	Methods      []string `json:"methods,omitempty"`
	Codes        []int32  `json:"codes,omitempty"`
}

// APICallCampaign defines an API interception telemetry campaign as a list of
// criterium to fulfil for an API call to be intercepted.
// A request should fulfil at least one of the criterium to be tracked.
type APICallCampaign []APICallCampaignCriterium

func (c *APICallCampaignCriterium) IsFulfilled(rp *RequestParams) bool {
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
		(len(c.PathPatterns) == 0 || rp.HasPathIn(c.PathPatterns)) &&
		(len(c.UserAgents) == 0 || rp.HasUserAgentWith(c.UserAgents))
}

func (c APICallCampaign) IsFulfilled(rp *RequestParams) bool {
	for _, cc := range c {
		if cc.IsFulfilled(rp) {
			return true
		}
	}
	return false
}
