package postgres

import (
	"math"
	"strconv"

	"github.com/jackc/pgtype"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/readable"
)

// dataTypeQueryMetadata includes metadata for queries on different data types.
type dataTypeQueryMetadata struct {
	// alloc allocates a value that can be passed to the postgres rows.Scan function,
	// in order to scan a value of this data type.
	alloc func() interface{}
	// printer converts the scanned value (allocated by alloc, populated by rows.Scan, and potentially transformed
	// by a post-transform func)
	// into a list of human-readable strings.
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
				// All the work of conversion is done by the post transform func, so
				// we don't need to do much here.
				return val.([]string)
			},
		},
		walker.DateTime: {
			alloc: func() interface{} {
				return &pgtype.Timestamp{}
			},
			printer: func(val interface{}) []string {
				ts, _ := val.(*pgtype.Timestamp)
				if ts == nil {
					return nil
				}
				return []string{readable.Time(ts.Time)}
			},
		},
		walker.Enum: {
			alloc: func() interface{} {
				return pointers.Int(0)
			},
			printer: func(val interface{}) []string {
				// The post transform func converts the enum to its string representation,
				// so it will be a string, not the (*int) allocated above.
				return []string{val.(string)}
			},
		},
		walker.Integer: {
			alloc: func() interface{} {
				return pointers.Int(0)
			},
			printer: func(val interface{}) []string {
				return []string{strconv.Itoa(*val.(*int))}
			},
		},
		walker.Numeric: {
			alloc: func() interface{} {
				return &pgtype.Numeric{}
			},
			printer: func(val interface{}) []string {
				asNumeric := val.(*pgtype.Numeric)
				if asNumeric.Status != pgtype.Present {
					return nil
				}
				switch asNumeric.InfinityModifier {
				case pgtype.Infinity:
					return []string{"inf"}
				case pgtype.NegativeInfinity:
					return []string{"-inf"}
				}
				if asNumeric.NaN {
					return []string{"NaN"}
				}
				asFloat := float64(asNumeric.Int.Int64()) * math.Pow(10, float64(asNumeric.Exp))
				return []string{readable.Float(asFloat, 3)}
			},
		},
		walker.IntArray: {
			alloc: func() interface{} {
				out := make([]int, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		walker.EnumArray: {
			alloc: func() interface{} {
				out := make([]int, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		walker.Map: {
			alloc: func() interface{} {
				out := make([]byte, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
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
