package postgres

import (
	"math"
	"strconv"

	"github.com/jackc/pgtype"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres"
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
	dataTypesToMetadata = map[postgres.DataType]dataTypeQueryMetadata{
		postgres.String: {
			alloc: func() interface{} {
				return pointers.String("")
			},
			printer: func(val interface{}) []string {
				return []string{*(val.(*string))}
			},
		},
		postgres.Bool: {
			alloc: func() interface{} {
				return pointers.Bool(false)
			},
			printer: func(val interface{}) []string {
				return []string{strconv.FormatBool(*(val.(*bool)))}
			},
		},
		postgres.StringArray: {
			alloc: func() interface{} {
				out := make([]string, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// All the work of conversion is done by the post transform func, so
				// we don't need to do much here.
				return val.([]string)
			},
		},
		postgres.DateTime: {
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
		postgres.Enum: {
			alloc: func() interface{} {
				return pointers.Int(0)
			},
			printer: func(val interface{}) []string {
				// The post transform func converts the enum to its string representation,
				// so it will be a string, not the (*int) allocated above.
				return []string{val.(string)}
			},
		},
		postgres.Integer: {
			alloc: func() interface{} {
				return pointers.Int(0)
			},
			printer: func(val interface{}) []string {
				return []string{strconv.Itoa(*val.(*int))}
			},
		},
		postgres.BigInteger: {
			alloc: func() interface{} {
				return pointers.Int64(0)
			},
			printer: func(val interface{}) []string {
				return []string{strconv.FormatInt(*val.(*int64), 10)}
			},
		},
		postgres.Numeric: {
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
		postgres.IntArray: {
			alloc: func() interface{} {
				out := make([]int, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.EnumArray: {
			alloc: func() interface{} {
				out := make([]int, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.Map: {
			alloc: func() interface{} {
				out := make([]byte, 0)
				return &out
			},
			printer: func(val interface{}) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.UUID: {
			alloc: func() interface{} {
				return pointers.String("")
			},
			printer: func(val interface{}) []string {
				return []string{*(val.(*string))}
			},
		},
	}
)

func mustAllocForDataType(typ postgres.DataType) interface{} {
	return dataTypesToMetadata[typ].alloc()
}

func mustPrintForDataType(typ postgres.DataType, val interface{}) []string {
	return dataTypesToMetadata[typ].printer(val)
}
