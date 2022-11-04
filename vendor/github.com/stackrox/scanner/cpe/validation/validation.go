package validation

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stackrox/scanner/cpe/match"
	"github.com/stackrox/scanner/pkg/component"
)

var (
	// Validators holds each registered Validator.
	Validators = make(map[component.SourceType]Validator)
)

// Validator is the common interface for validating NVD results.
type Validator interface {
	ValidateResult(result match.Result) bool
}

// Register registers the given validator.
func Register(src component.SourceType, validator Validator) {
	if _, ok := Validators[src]; ok {
		panic(fmt.Sprintf("%q has already been registered", src))
	}
	Validators[src] = validator
}

// TargetSWMatches checks if the TargetSW from the resulting CPE matches the language
// or matches any language
func TargetSWMatches(res match.Result, lang string) bool {
	for _, a := range res.CVE.Config() {
		if a.TargetSW == wfn.Any || a.TargetSW == lang {
			return true
		}
	}
	return false
}
