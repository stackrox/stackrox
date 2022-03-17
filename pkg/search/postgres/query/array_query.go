package pgsearch

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgtype"
)

func getStringArrayPostTransformFunc(filter func(interface{}) bool) func(val interface{}) interface{} {
	return func(val interface{}) interface{} {
		textArray, _ := val.(*pgtype.TextArray)
		if textArray == nil {
			return (*pgtype.TextArray)(nil)
		}

		var out []string
		for _, elem := range textArray.Elements {
			if elem.Status == pgtype.Present && filter(elem.String) {
				out = append(out, elem.String)
			}
		}
		return out
	}
}

func getIntArrayPostTransformFunc(filter func(interface{}) bool) func(val interface{}) interface{} {
	return func(val interface{}) interface{} {
		asIntArray := *(val.(*[]int))
		var out []string
		for _, elem := range asIntArray {
			if filter(elem) {
				out = append(out, strconv.Itoa(elem))
			}
		}
		return out
	}
}

func queryOnArray(baseQueryFunc queryFunction, postTransformFuncGetter func(func(interface{}) bool) func(interface{}) interface{}) queryFunction {
	return func(ctx *queryAndFieldContext) (*QueryEntry, error) {
		clonedCtx := *ctx
		clonedCtx.highlight = false
		clonedCtx.qualifiedColumnName = "elem"
		entry, err := baseQueryFunc(&clonedCtx)
		if err != nil {
			return nil, err
		}
		entry.Where.Query = fmt.Sprintf("exists (select * from unnest(%s) as elem where %s)", ctx.qualifiedColumnName, entry.Where.Query)
		if ctx.highlight {
			if entry.Where.equivalentGoFunc == nil {
				return nil, fmt.Errorf("highlights not supported for query %s on column %v", ctx.value, ctx.qualifiedColumnName)
			}
			entry.SelectedFields = []SelectQueryField{{
				SelectPath: ctx.qualifiedColumnName,
				FieldType:  ctx.dbField.DataType,
				FieldPath:  ctx.field.FieldPath,
				PostTransform: func(val interface{}) interface{} {
					return postTransformFuncGetter(entry.Where.equivalentGoFunc)(val)
				},
			}}
		}
		return entry, nil
	}
}
