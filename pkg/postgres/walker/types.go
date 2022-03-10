package walker

// DataType is the internal enum representation of the type
type DataType string

// Defines all the internal types derived from the struct fields
const (
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
	case StringArray:
		sqlType = "text[]"
	case EnumArray, IntArray:
		sqlType = "int[]"
	default:
		panic(dataType)
	}
	return sqlType
}
