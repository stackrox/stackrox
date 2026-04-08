package phonehome

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/glob"
)

// APICallCampaignCriterion defines a criterion for an API interception of a
// telemetry campaign. Requests parameters need to match all fields for the
// request to be tracked. Any request matches empty criterion.
type APICallCampaignCriterion struct {
	Path    *glob.Pattern `json:"path,omitempty"`
	Method  *glob.Pattern `json:"method,omitempty"`
	Codes   []int32       `json:"codes,omitempty"`
	Headers GlobMap       `json:"headers,omitempty"`
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
		if err := pattern.Compile(); err != nil {
			return errors.WithMessage(err, "error parsing header pattern")
		}
	}
	if err := c.Path.Compile(); err != nil {
		return errors.WithMessage(err, "error parsing path pattern")
	}
	if err := c.Method.Compile(); err != nil {
		return errors.WithMessage(err, "error parsing methods pattern")
	}
	return nil
}

func (c *APICallCampaignCriterion) isFulfilled(rp *RequestParams) Headers {
	if c != nil &&
		(len(c.Codes) == 0 || slices.Contains(c.Codes, int32(rp.Code))) &&
		(c.Path == nil || (*c.Path).Match(rp.Path)) &&
		(c.Method == nil || (*c.Method).Match(rp.Method)) {
		return rp.MatchHeaders(c.Headers)
	}
	return nil
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
func (c APICallCampaign) CountFulfilled(rp *RequestParams, f func(cc *APICallCampaignCriterion, h Headers)) int {
	fulfilled := 0
	for _, cc := range c {
		if h := cc.isFulfilled(rp); h != nil {
			f(cc, h)
			fulfilled++
		}
	}
	return fulfilled
}

// Codes creates an APICallCampaignCriterion that matches requests with any of the provided response codes.
// If called with no arguments, the resulting criterion does not restrict by response code.
func Codes(codes ...int32) *APICallCampaignCriterion {
	return &APICallCampaignCriterion{Codes: codes}
}

// MethodPattern builds an APICallCampaignCriterion that matches request methods using the provided glob pattern.
func MethodPattern(pattern glob.Pattern) *APICallCampaignCriterion {
	return &APICallCampaignCriterion{Method: pattern.Ptr()}
}

// PathPattern builds an APICallCampaignCriterion that matches request paths
// using the provided glob pattern.
func PathPattern(pattern glob.Pattern) *APICallCampaignCriterion {
	return &APICallCampaignCriterion{Path: pattern.Ptr()}
}

// HeaderPattern builds an APICallCampaignCriterion that matches a single
// request header against the provided glob pattern. The returned criterion's
// Headers map contains exactly one entry: the header name mapped to pattern.
func HeaderPattern(header string, pattern glob.Pattern) *APICallCampaignCriterion {
	return &APICallCampaignCriterion{
		Headers: GlobMap{
			header: pattern,
		},
	}
}
