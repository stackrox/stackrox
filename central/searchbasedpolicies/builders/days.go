package builders

import (
	"fmt"
	"strconv"
	"time"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/search"
)

type dayQueryBuilder struct {
	fieldLabel         search.FieldLabel
	fieldHumanName     string
	retrieveFieldValue func(*v1.PolicyFields) (int64, bool)
}

func (d *dayQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	days, exists := d.retrieveFieldValue(fields)
	if !exists {
		return
	}

	searchField, err := getSearchField(d.fieldLabel, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", d.Name(), err)
		return
	}

	q = search.NewQueryBuilder().AddDaysHighlighted(d.fieldLabel, days).ProtoQuery()

	v = violationPrinterForField(searchField.GetFieldPath(), func(match string) string {
		epochTime, err := strconv.ParseInt(match, 10, 64)
		if err != nil {
			logger.Errorf("Days query for %s: retrieved invalid epoch time in match: %s.", d.fieldHumanName, match)
			return ""
		}
		return fmt.Sprintf("%s '%s' was more than %d days ago", d.fieldHumanName, readable.Time(time.Unix(epochTime, 0)), days)
	})

	return
}

func (d *dayQueryBuilder) Name() string {
	return fmt.Sprintf("day-matching query builder for %s", d.fieldHumanName)
}

// NewDaysQueryBuilder returns a query builder for matching fields that check whether a field value is at least a
// certain number of days ago.
func NewDaysQueryBuilder(fieldLabel search.FieldLabel, fieldHumanName string,
	retrieveFieldValue func(*v1.PolicyFields) (int64, bool)) searchbasedpolicies.PolicyQueryBuilder {
	return &dayQueryBuilder{
		fieldLabel:         fieldLabel,
		fieldHumanName:     fieldHumanName,
		retrieveFieldValue: retrieveFieldValue,
	}
}
