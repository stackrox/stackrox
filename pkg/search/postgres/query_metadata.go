package postgres

import (
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/readable"
)

// dataTypeQueryMetadata includes metadata for queries on different data types.
type dataTypeQueryMetadata struct {
	// alloc allocates a value that can be passed to the postgres rows.Scan function,
	// in order to scan a value of this data type.
	alloc func() any
	// printer converts the scanned value (allocated by alloc, populated by rows.Scan, and potentially transformed
	// by a post-transform func)
	// into a list of human-readable strings.
	printer func(any) []string
}

var (
	dataTypesToMetadata = map[postgres.DataType]dataTypeQueryMetadata{
		postgres.String: {
			alloc: func() any {
				return new("")
			},
			printer: func(val any) []string {
				return []string{*(val.(*string))}
			},
		},
		postgres.Bool: {
			alloc: func() any {
				return new(false)
			},
			printer: func(val any) []string {
				return []string{strconv.FormatBool(*(val.(*bool)))}
			},
		},
		postgres.StringArray: {
			alloc: func() any {
				out := make([]string, 0)
				return &out
			},
			printer: func(val any) []string {
				// All the work of conversion is done by the post transform func, so
				// we don't need to do much here.
				return val.([]string)
			},
		},
		postgres.DateTime: {
			alloc: func() any {
				return &pgtype.Timestamp{}
			},
			printer: func(val any) []string {
				ts, _ := val.(*pgtype.Timestamp)
				if ts == nil {
					return nil
				}
				return []string{readable.Time(ts.Time)}
			},
		},
		postgres.DateTimeTZ: {
			alloc: func() any {
				return &pgtype.Timestamptz{}
			},
			printer: func(val any) []string {
				ts, _ := val.(*pgtype.Timestamptz)
				if ts == nil {
					return nil
				}
				return []string{readable.Time(ts.Time)}
			},
		},
		postgres.Enum: {
			alloc: func() any {
				return new(0)
			},
			printer: func(val any) []string {
				// The post transform func converts the enum to its string representation,
				// so it will be a string, not the (*int) allocated above.
				return []string{val.(string)}
			},
		},
		postgres.Integer: {
			alloc: func() any {
				return new(0)
			},
			printer: func(val any) []string {
				return []string{strconv.Itoa(*val.(*int))}
			},
		},
		postgres.BigInteger: {
			alloc: func() any {
				return new(int64(0))
			},
			printer: func(val any) []string {
				return []string{strconv.FormatInt(*val.(*int64), 10)}
			},
		},
		postgres.Numeric: {
			alloc: func() any {
				return &pgtype.Numeric{}
			},
			printer: func(val any) []string {
				asNumeric := val.(*pgtype.Numeric)
				if !asNumeric.Valid {
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
			alloc: func() any {
				out := make([]int, 0)
				return &out
			},
			printer: func(val any) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.EnumArray: {
			alloc: func() any {
				out := make([]int, 0)
				return &out
			},
			printer: func(val any) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.Map: {
			alloc: func() any {
				out := make([]byte, 0)
				return &out
			},
			printer: func(val any) []string {
				// The post-transform function does the work of conversion.
				return val.([]string)
			},
		},
		postgres.UUID: {
			alloc: func() any {
				return new("")
			},
			printer: func(val any) []string {
				return []string{*(val.(*string))}
			},
		},
	}
)

func mustAllocForDataType(typ postgres.DataType) any {
	return dataTypesToMetadata[typ].alloc()
}

func mustPrintForDataType(typ postgres.DataType, val any) []string {
	return dataTypesToMetadata[typ].printer(val)
}
