package querybuilders

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ForCompound returns a custom query builder for a compound field that contains <count> values, separated by =.
func ForCompound(field string, count int) QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		vals := make([]string, 0, len(group.GetValues()))
		for _, v := range group.GetValues() {
			fields := strings.SplitN(v.GetValue(), "=", count)
			if len(fields) != count {
				return nil
			}

			for idx := range fields {
				fields[idx] = fmt.Sprintf("(%s)", stringutils.OrDefault(fields[idx], ".*"))
			}

			// Eg: Compound fields are augmented and stored as "firstValue\tsecondValue"
			// To match this, we create the regex "(firstRegex)\t(secondRegex)",
			// replacing empty component by a ".*"
			vals = append(vals, fmt.Sprintf("%s%s",
				search.RegexPrefix,
				strings.Join(fields, augmentedobjs.CompositeFieldCharSep)))
		}

		return []*query.FieldQuery{
			{
				Field:    field,
				Operator: operatorProtoMap[group.GetBooleanOperator()],
				Values:   vals,
			},
		}
	})
}
