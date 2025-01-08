package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// CampaignCriterium defines a criterium for an API interception of a telemetry
// campaign. Requests parameters need to match all fields for the request to
// be tracked.
type CampaignCriterium struct {
	UserAgents   []string `json:"userAgents,omitempty"`
	PathPatterns []string `json:"pathPatterns,omitempty"`
	Methods      []string `json:"methods,omitempty"`
	Codes        []int    `json:"codes,omitempty"`
}

// Campaign defines an API interception telemetry campaign as a list of
// criterium to fulfil for an API call to be intercepted.
// A request should fulfil at least one of the criterium to be tracked.
type Campaign []CampaignCriterium

func (c *CampaignCriterium) IsFulfilled(rp *phonehome.RequestParams) bool {
	codeMatches := len(c.Codes) == 0
	for _, code := range c.Codes {
		if rp.Code == code {
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
		(len(c.PathPatterns) == 0 || rp.HasPathIn(c.PathPatterns) && !rp.HasPathIn(ignoredPaths)) &&
		(len(c.UserAgents) == 0 || rp.HasUserAgentWith(c.UserAgents))
}

func (c Campaign) IsFulfilled(rp *phonehome.RequestParams) bool {
	for _, cc := range c {
		if cc.IsFulfilled(rp) {
			return true
		}
	}
	return false
}
