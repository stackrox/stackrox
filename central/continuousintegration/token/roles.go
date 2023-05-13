package token

import (
	"regexp"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

type roleMapperImpl struct{}

func (r *roleMapperImpl) MapRoles(configsForType []*storage.ContinuousIntegrationConfig, idToken *oidc.IDToken) []string {
	rolesToAssign := set.NewStringSet()
	for _, cfg := range configsForType {
		for _, mapping := range cfg.GetMappings() {
			if valuesMatch(idToken.Subject, mapping.GetValue()) && !rolesToAssign.Contains(mapping.GetRole()) {
				rolesToAssign.Add(mapping.GetRole())
			}
		}
	}
	return rolesToAssign.AsSlice()
}

func checkIfRegexp(expr string) *regexp.Regexp {
	parsedExpr, err := regexp.Compile(expr)
	if err != nil {
		return nil
	}
	return parsedExpr
}

func valuesMatch(claimValue string, expr string) bool {
	// The expression is either a simple string value or a regular expression.
	if regExpr := checkIfRegexp(expr); regExpr != nil {
		return regExpr.MatchString(claimValue)
	}
	// Otherwise if it is not a regular expression, fall back to string comparison.
	return claimValue == expr
}
