package walker

// DataType is the internal enum representation of the type
type DataType string

// Defines all the internal types derived from the struct fields
const (
	Bytes       DataType = "bytes"
	Bool        DataType = "bool"
	Numeric     DataType = "numeric"
	String      DataType = "string"
	DateTime    DataType = "datetime"
	Map         DataType = "map"
	Enum        DataType = "enum"
	StringArray DataType = "stringarray"
	EnumArray   DataType = "enumarray"
	Integer     DataType = "integer"
	IntArray    DataType = "intarray"
	BigInteger  DataType = "biginteger"
)

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
	default:
		panic(dataType)
	}
	return sqlType
}

// GetToGormModelType converts the internal representation to Gorm Model type
func GetToGormModelType(typ string, dataType DataType) string {
	var modelType string
	switch dataType {
	case DateTime:
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
