package phonehome

import (
	"slices"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

type Pattern string

func (p *Pattern) compile() (glob.Glob, error) {
	g, err := glob.Compile(string(*p))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to compile %q", string(*p))
	}
	return g, nil
}

func (p Pattern) Pointer() *Pattern {
	return &p
}

var globCache = make(map[Pattern]glob.Glob)

// APICallCampaignCriterion defines a criterion for an API interception of a
// telemetry campaign. Requests parameters need to match all fields for the
// request to be tracked. Any request matches empty criterion.
type APICallCampaignCriterion struct {
	Paths   *Pattern           `json:"paths,omitempty"`
	Methods *Pattern           `json:"methods,omitempty"`
	Codes   []int32            `json:"codes,omitempty"`
	Headers map[string]Pattern `json:"headers,omitempty"`
}

// APICallCampaign defines an API interception telemetry campaign as a list of
// criteria to fulfil for an API call to be intercepted.
// A request should fulfil at least one of the criterion to be tracked.
type APICallCampaign []*APICallCampaignCriterion

func (c *APICallCampaignCriterion) Compile() error {
	if c == nil {
		return nil
	}
	var err error
	for _, pattern := range c.Headers {
		if globCache[pattern], err = pattern.compile(); err != nil {
			return errors.WithMessage(err, "error parsing header pattern")
		}
	}
	if c.Paths != nil {
		if globCache[*c.Paths], err = c.Paths.compile(); err != nil {
			return errors.WithMessage(err, "error parsing path pattern")
		}
	}
	if c.Methods != nil {
		if globCache[*c.Methods], err = c.Methods.compile(); err != nil {
			return errors.WithMessage(err, "error parsing methods pattern")
		}
	}
	return nil
}

func (c *APICallCampaignCriterion) isFulfilled(rp *RequestParams) bool {
	return c != nil &&
		(len(c.Codes) == 0 || slices.Contains(c.Codes, int32(rp.Code))) &&
		(c.Paths == nil || globCache[*c.Paths].Match(rp.Path)) &&
		(c.Methods == nil || globCache[*c.Methods].Match(rp.Method)) &&
		(c.Headers == nil || rp.HasHeader(c.Headers))
}

func (c APICallCampaign) Compile() error {
	for _, cc := range c {
		if err := cc.Compile(); err != nil {
			return err
		}
	}
	return nil
}

func (c APICallCampaign) IsFulfilled(rp *RequestParams) bool {
	return slices.ContainsFunc(c, func(cc *APICallCampaignCriterion) bool {
		return cc.isFulfilled(rp)
	})
}
