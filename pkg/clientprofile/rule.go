package clientprofile

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/grpc/common/requestinterceptor"
)

// Rule defines a set of conditions for matching an API call.
// Request parameters must match all fields to satisfy the rule. An empty rule
// matches any request.
type Rule struct {
	Path    *glob.Pattern `json:"path,omitempty"`
	Method  *glob.Pattern `json:"method,omitempty"`
	Codes   []int32       `json:"codes,omitempty"`
	Headers GlobMap       `json:"headers,omitempty"`
}

// RuleSet is a list of matchers where at least one must match for an API call
// to be selected.
type RuleSet []*Rule

// Compile compiles and caches all glob patterns of the rule.
func (r *Rule) Compile() error {
	if r == nil {
		return nil
	}
	for name, value := range r.Headers {
		if err := name.Compile(); err != nil {
			return errors.WithMessage(err, "error parsing header name pattern")
		}
		if err := value.Compile(); err != nil {
			return errors.WithMessage(err, "error parsing header value pattern")
		}
	}
	if err := r.Path.Compile(); err != nil {
		return errors.WithMessage(err, "error parsing path pattern")
	}
	if err := r.Method.Compile(); err != nil {
		return errors.WithMessage(err, "error parsing methods pattern")
	}
	return nil
}

// Match reports whether the request satisfies all conditions of the rule.
// A nil rule never matches. An empty rule matches any request.
func (r *Rule) Match(rp *requestinterceptor.RequestParams) (bool, Headers) {
	if r != nil &&
		(len(r.Codes) == 0 || slices.Contains(r.Codes, int32(rp.Code))) &&
		(r.Path == nil || (*r.Path).Match(rp.Path)) &&
		(r.Method == nil || (*r.Method).Match(rp.Method)) {
		return Headers(rp.Headers).Match(r.Headers)
	}
	return false, nil
}

// Compile compiles and caches all glob patterns of every rule.
func (rs RuleSet) Compile() error {
	for _, cc := range rs {
		if err := cc.Compile(); err != nil {
			return err
		}
	}
	return nil
}

// CountMatched calls f on each satisfied rule and returns their count.
func (rs RuleSet) CountMatched(rp *requestinterceptor.RequestParams, f func(cc *Rule, h Headers)) int {
	matched := 0
	for _, rule := range rs {
		if ok, h := rule.Match(rp); ok {
			f(rule, h)
			matched++
		}
	}
	return matched
}

// Codes creates a rule that matches requests with any of the provided response
// codes.
// If called with no arguments, the resulting rule does not restrict by response
// code.
func Codes(codes ...int32) *Rule {
	return &Rule{Codes: codes}
}

// MethodPattern builds a rule that matches request methods using the provided
// glob pattern.
func MethodPattern(pattern glob.Pattern) *Rule {
	return &Rule{Method: pattern.Ptr()}
}

// PathPattern builds a rule that matches request paths using the provided glob
// pattern.
func PathPattern(pattern glob.Pattern) *Rule {
	return &Rule{Path: pattern.Ptr()}
}

// HeaderPattern builds a rule that matches request headers against the provided
// glob patterns. The returned rule's Headers map contains exactly one entry:
// the header name pattern mapped to a value pattern.
func HeaderPattern(header glob.Pattern, pattern glob.Pattern) *Rule {
	return &Rule{
		Headers: GlobMap{
			header: pattern,
		},
	}
}
