package walker

//go:generate stringer -type=DataType
type DataType int

const (
	BOOL         DataType = 0
	NUMERIC      DataType = 1
	STRING       DataType = 2
	DATETIME     DataType = 3
	MAP          DataType = 4
	ENUM         DataType = 5
	ARRAY        DataType = 6
	STRING_ARRAY DataType = 7
	INT_ARRAY    DataType = 8
	INTEGER      DataType = 9
)

func DataTypeToSQLType(dataType DataType) string {
	var sqlType string
	switch dataType {
	case BOOL:
		sqlType = "bool"
	case NUMERIC:
		sqlType = "numeric"
	case STRING:
		sqlType = "varchar"
	case DATETIME:
		sqlType = "timestamp"
	case MAP:
		sqlType = "jsonb"
	case ENUM, INTEGER:
		sqlType = "integer"
	case STRING_ARRAY:
		sqlType = "text[]"
	case INT_ARRAY:
		sqlType = "int[]"
	default:
		panic(dataType.String())
	}
	return sqlType
}
