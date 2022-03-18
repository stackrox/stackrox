package pgsearch

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgtype"
)

// See the documentation on the PostTransform field of the SelectQueryField struct
// for more clarity on the purpose of these post transform funcs.

func getStringArrayPostTransformFunc(entry *QueryEntry) (func(val interface{}) interface{}, error) {
	filterFunc := entry.Where.equivalentGoFunc
	if filterFunc == nil {
		return nil, errors.New("no filter func found")
	}
	return func(val interface{}) interface{} {
		textArray, _ := val.(*pgtype.TextArray)
		if textArray == nil {
			return (*pgtype.TextArray)(nil)
		}

		var out []string
		for _, elem := range textArray.Elements {
			if elem.Status == pgtype.Present && filterFunc(elem.String) {
				out = append(out, elem.String)
			}
		}
		return out
	}, nil
}

func getIntArrayPostTransformFunc(entry *QueryEntry) (func(val interface{}) interface{}, error) {
	filterFunc := entry.Where.equivalentGoFunc
	if filterFunc == nil {
		return nil, errors.New("no filter func found")
	}

	return func(val interface{}) interface{} {
		asIntArray := *(val.(*[]int))
		var out []string
		for _, elem := range asIntArray {
			if filterFunc(elem) {
				out = append(out, strconv.Itoa(elem))
			}
		}
		return out
	}, nil
}

func getEnumArrayPostTransformFunc(entry *QueryEntry) (func(val interface{}) interface{}, error) {
	filterFunc := entry.Where.equivalentGoFunc
	if filterFunc == nil {
		return nil, errors.New("no filter func found")
	}
	if entry.enumStringifyFunc == nil {
		return nil, errors.New("no enum stringify func found")
	}

	return func(val interface{}) interface{} {
		asIntArray := *(val.(*[]int))
		var out []string
		for _, elem := range asIntArray {
			if filterFunc(elem) {
				out = append(out, entry.enumStringifyFunc(int32(elem)))
			}
		}
		return out
	}, nil
}

func queryOnArray(baseQueryFunc queryFunction, postTransformFuncGetter func(entry *QueryEntry) (func(interface{}) interface{}, error)) queryFunction {
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
			postTransformFunc, err := postTransformFuncGetter(entry)
			if err != nil {
				return nil, fmt.Errorf("highlights not supported for query %s on column %v: %w", ctx.value, ctx.qualifiedColumnName, err)
			}

			entry.SelectedFields = []SelectQueryField{{
				SelectPath:    ctx.qualifiedColumnName,
				FieldType:     ctx.dbField.DataType,
				FieldPath:     ctx.field.FieldPath,
				PostTransform: postTransformFunc,
			}}
		}
		return entry, nil
	}
}
