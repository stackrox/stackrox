package phonehome

import (
	"slices"

	"github.com/pkg/errors"
)

// APICallCampaignCriterion defines a criterion for an API interception of a
// telemetry campaign. Requests parameters need to match all fields for the
// request to be tracked. Any request matches empty criterion.
type APICallCampaignCriterion struct {
	Path    *Pattern           `json:"path,omitempty"`
	Method  *Pattern           `json:"method,omitempty"`
	Codes   []int32            `json:"codes,omitempty"`
	Headers map[string]Pattern `json:"headers,omitempty"`
}

// APICallCampaign defines an API interception telemetry campaign as a list of
// criteria to fulfil for an API call to be intercepted.
// A request should fulfil at least one of the criterion to be tracked.
type APICallCampaign []*APICallCampaignCriterion

// Compile compiles and caches all glob patterns of the criterion.
func (c *APICallCampaignCriterion) Compile() error {
	if c == nil {
		return nil
	}
	for _, pattern := range c.Headers {
		if err := pattern.compile(); err != nil {
			return errors.WithMessage(err, "error parsing header pattern")
		}
	}
	if err := c.Path.compile(); err != nil {
		return errors.WithMessage(err, "error parsing path pattern")
	}
	if err := c.Method.compile(); err != nil {
		return errors.WithMessage(err, "error parsing methods pattern")
	}
	return nil
}

func (c *APICallCampaignCriterion) isFulfilled(rp *RequestParams) bool {
	return c != nil &&
		(len(c.Codes) == 0 || slices.Contains(c.Codes, int32(rp.Code))) &&
		(c.Path == nil || (*c.Path).Match(rp.Path)) &&
		(c.Method == nil || (*c.Method).Match(rp.Method)) &&
		(c.Headers == nil || rp.HasHeader(c.Headers))
}

// Compile compiles and caches all glob patterns of the campaign.
func (c APICallCampaign) Compile() error {
	for _, cc := range c {
		if err := cc.Compile(); err != nil {
			return err
		}
	}
	return nil
}

// CountFulfilled calls f on each fulfilled criterion and returns their number.
func (c APICallCampaign) CountFulfilled(rp *RequestParams, f func(cc *APICallCampaignCriterion)) int {
	fulfilled := 0
	for _, cc := range c {
		if cc.isFulfilled(rp) {
			f(cc)
			fulfilled++
		}
	}
	return fulfilled
}
