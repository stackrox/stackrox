package pgsearch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

func newUUIDQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	whereClause, err := newUUIDQueryWhereClause(ctx.qualifiedColumnName, ctx.value, ctx.queryModifiers...)
	if err != nil {
		return nil, err
	}
	return qeWithSelectFieldIfNeeded(ctx, &whereClause, nil), nil
}

func newUUIDQueryWhereClause(columnName string, value string, queryModifiers ...pkgSearch.QueryModifier) (WhereClause, error) {
	if len(value) == 0 {
		return WhereClause{}, errors.New("value in search query cannot be empty")
	}

	if value == pkgSearch.WildcardString {
		return WhereClause{
			Query:  fmt.Sprintf("%s is not null", columnName),
			Values: []interface{}{},
			equivalentGoFunc: func(foundValue interface{}) bool {
				foundVal := strings.ToLower(foundValue.(string))
				return foundVal != ""
			},
		}, nil
	}

	if len(queryModifiers) == 0 {
		uuidVal, err := uuid.FromString(value)
		if err != nil {
			return WhereClause{}, errors.Wrapf(err, "value %q in search query must be valid UUID", value)
		}
		return WhereClause{
			Query:  fmt.Sprintf("%s = $$", columnName),
			Values: []interface{}{uuidVal},
			equivalentGoFunc: func(foundValue interface{}) bool {
				return strings.EqualFold(foundValue.(string), value)
			},
		}, nil
	}
	if queryModifiers[0] == pkgSearch.AtLeastOne {
		panic("I dont think this is used")
	}
	var negationString string
	negated := queryModifiers[0] == pkgSearch.Negation
	if negated {
		negationString = "!"
		if len(queryModifiers) == 1 {
			uuidVal, err := uuid.FromString(value)
			if err != nil {
				return WhereClause{}, errors.Wrapf(err, "value %q in search query must be valid UUID", value)
			}
			return WhereClause{
				Query:  fmt.Sprintf("%s != $$", columnName),
				Values: []interface{}{uuidVal},
				equivalentGoFunc: func(foundValue interface{}) bool {
					return strings.EqualFold(foundValue.(string), value) != negated
				},
			}, nil
		}
		queryModifiers = queryModifiers[1:]
	}

	switch queryModifiers[0] {
	case pkgSearch.Regex:
		re, err := regexp.Compile(value)
		if err != nil {
			return WhereClause{}, fmt.Errorf("invalid regexp %s: %w", value, err)
		}
		return WhereClause{
			Query:  fmt.Sprintf("%s::text %s~* $$", columnName, negationString),
			Values: []interface{}{value},
			equivalentGoFunc: func(foundValue interface{}) bool {
				foundVal := strings.ToLower(foundValue.(string))
				return re.MatchString(foundVal) != negated
			},
		}, nil
	case pkgSearch.Equality:
		uuidVal, err := uuid.FromString(value)
		if err != nil {
			return WhereClause{}, errors.Wrapf(err, "value %q in search query must be valid UUID", value)
		}
		return WhereClause{
			Query:  fmt.Sprintf("%s %s= $$", columnName, negationString),
			Values: []interface{}{uuidVal},
			equivalentGoFunc: func(foundValue interface{}) bool {
				return strings.EqualFold(foundValue.(string), value) != negated
			},
		}, nil
	}
	err := fmt.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return WhereClause{}, err
}
