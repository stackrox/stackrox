package pgsearch

import (
	"fmt"
	"strconv"

	"github.com/stackrox/stackrox/pkg/parse"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

func newBoolQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	if len(ctx.queryModifiers) > 0 {
		return nil, fmt.Errorf("modifiers for bool query not allowed: %+v", ctx.queryModifiers)
	}
	res, err := parse.FriendlyParseBool(ctx.value)
	if err != nil {
		return nil, err
	}
	// explicitly apply equality check
	ctx.value = strconv.FormatBool(res)
	ctx.queryModifiers = []pkgSearch.QueryModifier{pkgSearch.Equality}
	return newStringQuery(ctx)
}
