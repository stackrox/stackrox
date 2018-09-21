package builders

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// EnvQueryBuilder builds queries for environment policiies.
type EnvQueryBuilder struct {
}

func regexOrWildcard(valueInPolicy string) string {
	if valueInPolicy == "" {
		return search.WildcardString
	}
	return search.RegexQueryString(valueInPolicy)
}

func printEnvPolicy(env *v1.KeyValuePolicy) string {
	sb := strings.Builder{}
	if env.GetKey() != "" {
		sb.WriteString(fmt.Sprintf("key = '%s'", env.GetKey()))
		if env.GetValue() != "" {
			sb.WriteString(", ")
		}
	}
	if env.GetValue() != "" {
		sb.WriteString(fmt.Sprintf("value = '%s'", env.GetValue()))
	}
	return sb.String()
}

// Query implements the PolicyQueryBuilder interface.
func (e EnvQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	if fields.GetEnv().GetKey() == "" && fields.GetEnv().GetValue() == "" {
		return
	}

	keySearchField, err := getSearchField(search.EnvironmentKey, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", e.Name(), err)
		return
	}
	valueSearchField, err := getSearchField(search.EnvironmentValue, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", e.Name(), err)
		return
	}

	keyQuery := regexOrWildcard(fields.GetEnv().GetKey())
	valueQuery := regexOrWildcard(fields.GetEnv().GetValue())
	q = search.NewQueryBuilder().AddLinkedFieldsHighlighted(
		[]search.FieldLabel{search.EnvironmentKey, search.EnvironmentValue},
		[]string{keyQuery, valueQuery}).ProtoQuery()

	v = func(result search.Result) []*v1.Alert_Violation {
		keyMatches := result.Matches[keySearchField.GetFieldPath()]
		valueMatches := result.Matches[valueSearchField.GetFieldPath()]
		if len(keyMatches) == 0 || len(valueMatches) == 0 {
			return nil
		}
		violations := make([]*v1.Alert_Violation, 0, len(keyMatches))
		for i, keyMatch := range keyMatches {
			if i >= len(valueMatches) {
				logger.Errorf("Mismatched number of key and value matches: %+v; %+v", keyMatches, valueMatches)
				return violations
			}
			violations = append(violations, &v1.Alert_Violation{
				Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched environment policy (%s)",
					keyMatch, valueMatches[i], printEnvPolicy(fields.GetEnv())),
			})
		}
		return violations
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (EnvQueryBuilder) Name() string {
	return "query builder for env variables"
}
