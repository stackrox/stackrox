package walker

import (
	"reflect"
	"strings"
)

// goTypeToArraySQLType converts a Go type to its PostgreSQL array type.
func goTypeToArraySQLType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "text[]"
	case reflect.Int32, reflect.Uint32:
		return "int4[]"
	case reflect.Int64, reflect.Uint64:
		return "int8[]"
	case reflect.Bool:
		return "bool[]"
	default:
		return ""
	}
}

// goTypeToArrayGoType converts a Go type to its array Go type string.
func goTypeToArrayGoType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "[]string"
	case reflect.Int32:
		return "[]int32"
	case reflect.Uint32:
		return "[]uint32"
	case reflect.Int64:
		return "[]int64"
	case reflect.Uint64:
		return "[]uint64"
	case reflect.Bool:
		return "[]bool"
	default:
		return ""
	}
}

// buildProtoFieldPath constructs a dot-separated proto field path from the walker context.
// For example, if ctx.column is "Signal", it returns "signal.lineageinfo".
func buildProtoFieldPath(ctx walkerContext, protoFieldName string) string {
	// The column path from ctx.column is in CamelCase (e.g., "Signal" or "Signal_Metadata")
	// We need to convert it to lowercase dot-separated (e.g., "signal" or "signal.metadata")
	var parts []string
	if ctx.column != "" {
		// Split on underscore to handle nested fields
		columnParts := strings.Split(ctx.column, "_")
		for _, part := range columnParts {
			parts = append(parts, strings.ToLower(part))
		}
	}
	// Add the current field name
	parts = append(parts, strings.ToLower(protoFieldName))
	return strings.Join(parts, ".")
}
