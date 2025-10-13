package pgsearch

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func newCIDRQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	whereClause, err := newCIDRQueryWhereClause(ctx.qualifiedColumnName, ctx.value, ctx.queryModifiers...)
	if err != nil {
		return nil, err
	}
	return qeWithSelectFieldIfNeeded(ctx, &whereClause, nil), nil
}

func newCIDRQueryWhereClause(columnName string, value string, queryModifiers ...pkgSearch.QueryModifier) (WhereClause, error) {
	if len(value) == 0 {
		return WhereClause{}, errors.New("value in search query cannot be empty")
	}

	if len(queryModifiers) == 0 {
		_, cidrVal, err := net.ParseCIDR(value)
		if err != nil {
			return WhereClause{}, errors.Wrapf(err, "value %q in search query must be valid CIDR", value)
		}
		return WhereClause{
			Query:  fmt.Sprintf("%s <<= $$", columnName),
			Values: []interface{}{cidrVal},
			equivalentGoFunc: func(foundValue interface{}) bool {
				foundValueStr, ok := foundValue.(string)
				if !ok {
					return false
				}
				return IPNetContainsSubnet(cidrVal, foundValueStr)
			},
		}, nil
	}
	err := fmt.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return WhereClause{}, err
}

func IPNetContainsSubnet(cidr *net.IPNet, sub string) bool {
	if cidr == nil {
		return false
	}

	subIP, subCIDR, err := net.ParseCIDR(sub)
	if err != nil {
		return false
	}

	cidrOnes, _ := cidr.Mask.Size()
	subOnes, _ := subCIDR.Mask.Size()
	if subOnes < cidrOnes {
		return false
	}

	return cidr.Contains(subIP)
}
