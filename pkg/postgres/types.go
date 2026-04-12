package postgres

import "github.com/stackrox/rox/pkg/postgres/datatypes"

// DataType is re-exported from datatypes for backward compatibility.
// New code should import pkg/postgres/datatypes directly to avoid
// pulling in the pgx database driver.
type DataType = datatypes.DataType

// Re-export all DataType constants from the lightweight datatypes package.
const (
	Bytes       = datatypes.Bytes
	Bool        = datatypes.Bool
	Numeric     = datatypes.Numeric
	String      = datatypes.String
	DateTime    = datatypes.DateTime
	Map         = datatypes.Map
	Enum        = datatypes.Enum
	StringArray = datatypes.StringArray
	EnumArray   = datatypes.EnumArray
	Integer     = datatypes.Integer
	IntArray    = datatypes.IntArray
	BigInteger  = datatypes.BigInteger
	UUID        = datatypes.UUID
	CIDR        = datatypes.CIDR
	DateTimeTZ  = datatypes.DateTimeTZ
)

// UnsupportedDerivedFieldDataTypes is re-exported from datatypes.
var UnsupportedDerivedFieldDataTypes = datatypes.UnsupportedDerivedFieldDataTypes

// DataTypeToSQLType converts the internal representation to SQL
func DataTypeToSQLType(dataType DataType) string {
	var sqlType string
	switch dataType {
	case Bool:
		sqlType = "bool"
	case Numeric:
		sqlType = "numeric"
	case String:
		sqlType = "varchar"
	case DateTime:
		sqlType = "timestamp"
	case DateTimeTZ:
		sqlType = "timestamptz"
	case Map:
		sqlType = "jsonb"
	case Enum, Integer:
		sqlType = "integer"
	case BigInteger:
		sqlType = "bigint"
	case StringArray:
		sqlType = "text[]"
	case EnumArray, IntArray:
		sqlType = "int[]"
	case Bytes:
		sqlType = "bytea"
	case CIDR:
		sqlType = "cidr"
	default:
		panic(dataType)
	}
	return sqlType
}

// GetToGormModelType converts the internal representation to Gorm Model type
func GetToGormModelType(typ string, dataType DataType) string {
	var modelType string
	switch dataType {
	case DateTime, DateTimeTZ:
		modelType = "*time.Time"
	case StringArray:
		modelType = "*pq.StringArray"
	case EnumArray, IntArray:
		modelType = "*pq.Int32Array"
	default:
		modelType = typ
	}
	return modelType
}
