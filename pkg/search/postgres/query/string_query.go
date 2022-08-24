package pgsearch

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func newStringQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	whereClause, err := newStringQueryWhereClause(ctx.qualifiedColumnName, ctx.value, ctx.queryModifiers...)
	if err != nil {
		return nil, err
	}
	return qeWithSelectFieldIfNeeded(ctx, &whereClause, nil), nil
}

func newStringQueryWhereClause(columnName string, value string, queryModifiers ...pkgSearch.QueryModifier) (WhereClause, error) {
	if len(value) == 0 {
		return WhereClause{}, errors.New("value in search query cannot be empty")
	}

	if len(queryModifiers) == 0 {
		return WhereClause{
			Query:  fmt.Sprintf("%s ilike $$", columnName),
			Values: []interface{}{"%" + value + "%"},
			equivalentGoFunc: func(foundValue interface{}) bool {
				return strings.HasPrefix(foundValue.(string), value)
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
			return WhereClause{
				Query:  fmt.Sprintf("NOT (%s ilike $$)", columnName),
				Values: []interface{}{"%" + value + "%"},
				equivalentGoFunc: func(foundValue interface{}) bool {
					return !strings.HasPrefix(foundValue.(string), value)
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
			Query:  fmt.Sprintf("%s %s~* $$", columnName, negationString),
			Values: []interface{}{value},
			equivalentGoFunc: func(foundValue interface{}) bool {
				return re.MatchString(foundValue.(string)) != negated
			},
		}, nil
	case pkgSearch.Equality:
		return WhereClause{
			Query:  fmt.Sprintf("%s %s= $$", columnName, negationString),
			Values: []interface{}{value},
			equivalentGoFunc: func(foundValue interface{}) bool {
				return (foundValue.(string) == value) != negated
			},
		}, nil
	}
	err := fmt.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return WhereClause{}, err
}
