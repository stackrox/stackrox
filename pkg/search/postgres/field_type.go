package postgres

import (
	"strconv"

	"github.com/jackc/pgtype"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

type dataTypeQueryMetadata struct {
	alloc   func() interface{}
	printer func(interface{}) []string
}

var (
	dataTypesToMetadata = map[walker.DataType]dataTypeQueryMetadata{
		walker.String: {
			alloc: func() interface{} {
				return pointers.String("")
			},
			printer: func(val interface{}) []string {
				return []string{*(val.(*string))}
			},
		},
		walker.Bool: {
			alloc: func() interface{} {
				return pointers.Bool(false)
			},
			printer: func(val interface{}) []string {
				return []string{strconv.FormatBool(*(val.(*bool)))}
			},
		},
		walker.StringArray: {
			alloc: func() interface{} {
				return &pgtype.TextArray{}
			},
			printer: func(val interface{}) []string {
				textArray := val.(*pgtype.TextArray)
				var out []string
				for _, elem := range textArray.Elements {
					if elem.Status == pgtype.Present {
						out = append(out, elem.String)
					}
				}
				return out
			},
		},
	}
)

func mustAllocForDataType(typ walker.DataType) interface{} {
	return dataTypesToMetadata[typ].alloc()
}

func mustPrintForDataType(typ walker.DataType, val interface{}) []string {
	return dataTypesToMetadata[typ].printer(val)
}
