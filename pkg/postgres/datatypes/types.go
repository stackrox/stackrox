// Package datatypes defines PostgreSQL data type constants used by the search
// and schema packages. It is intentionally free of pgx/driver dependencies so
// that non-database binaries (sensor, AC) can reference column types without
// pulling in the database driver.
package datatypes

import "github.com/stackrox/rox/pkg/set"

// DataType is the internal enum representation of the type.
type DataType string

// Defines all the internal types derived from the struct fields.
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
	UUID        DataType = "uuid"
	CIDR        DataType = "cidr"
	DateTimeTZ  DataType = "datetimetz"
)

// UnsupportedDerivedFieldDataTypes are the data types to which aggregation functions cannot be applied.
var UnsupportedDerivedFieldDataTypes = set.NewFrozenSet(StringArray, EnumArray, IntArray, Map)
