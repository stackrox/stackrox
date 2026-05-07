package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"text/template"
	"unicode"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/stringutils"
)

func parseReferencesAndInjectPeerSchemas(schema *walker.Schema, refs []string) (parsedRefs []parsedReference) {
	schemasByObjType := make(map[string]*walker.Schema, len(refs))
	parsedRefs = make([]parsedReference, 0, len(refs))
	for _, ref := range refs {
		var refTable, refObjType string
		if strings.Contains(ref, ":") {
			refTable, refObjType = stringutils.Split2(ref, ":")
		} else {
			refObjType = ref
			refTable = pgutils.NamingStrategy.TableName(stringutils.GetAfter(refObjType, "."))
		}

		refMsgType := protoutils.MessageType(refObjType)
		if refMsgType == nil {
			log.Fatalf("could not find message for type: %s", refObjType)
		}
		refSchema := walker.Walk(refMsgType, refTable)
		schemasByObjType[refObjType] = refSchema
		parsedRefs = append(parsedRefs, parsedReference{
			TypeName: refObjType,
			Table:    refTable,
		})
	}
	schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
		return schemasByObjType[fmt.Sprintf("storage.%s", messageTypeName)]
	})
	return parsedRefs
}

func splitWords(s string) []string {
	var words []string
	var currWord strings.Builder

	for _, r := range s {
		newWord, skip := false, false
		if unicode.IsUpper(r) && currWord.Len() > 0 {
			newWord = true
		} else if !unicode.IsOneOf([]*unicode.RangeTable{unicode.Letter, unicode.Digit}, r) {
			newWord, skip = true, true
		}

		if newWord && currWord.Len() > 0 {
			words = append(words, currWord.String())
			currWord.Reset()
		}
		if !skip {
			currWord.WriteRune(r)
		}
	}

	if currWord.Len() > 0 {
		words = append(words, currWord.String())
	}

	return words
}

func applyPointwise(ss []string, f func(string) string) {
	for i, s := range ss {
		ss[i] = f(s)
	}
}

func lowerCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	words[0] = strings.ToLower(words[0])
	if len(words) == 1 && words[0] == "id" {
		return "id"
	}
	applyPointwise(words[1:], strings.Title)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}

func upperCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	applyPointwise(words, strings.Title)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}

func valueExpansion(size int) string {
	var all []string
	for i := 0; i < size; i++ {
		all = append(all, fmt.Sprintf("$%d", i+1))
	}
	return strings.Join(all, ", ")
}

func concatWith(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func arr(els ...any) []any {
	return els
}

func isArrayColumn(f walker.Field) bool {
	return f.DataType == postgres.ArrayColumn
}

func arrayExtractExpr(f walker.Field) string {
	return fmt.Sprintf(`func() %s {
		src := obj.%s
		result := make(%s, len(src))
		for i, elem := range src {
			result[i] = elem.%s
		}
		return result
	}()`, f.ModelType, f.ArraySourceGetter, f.ModelType, f.ArrayFieldName)
}

// scanVarType returns the Go type to use for scanning a field from the database.
func scanVarType(f walker.Field) string {
	if f.ObjectGetter.IsVariable() {
		// Variable fields (serialized, idx, parent FK) use their own type
		return f.Type
	}
	switch f.DataType {
	case "datetime", "datetimetz":
		return "*time.Time"
	case "enum":
		return "int32"
	case "integer":
		return "int32"
	case "biginteger":
		if f.Type == "uint32" || f.Type == "uint64" {
			return f.Type
		}
		return "int64"
	case "numeric":
		if f.Type == "uint64" {
			return "uint64"
		}
		if f.Type == "float32" {
			return "float32"
		}
		return "float64"
	case "bool":
		return "bool"
	case "string":
		return "string"
	case "stringarray":
		return "[]string"
	case "bytes", "messagebytes":
		return "[]byte"
	case "enumarray":
		return "[]int32"
	case "intarray":
		return "[]int32"
	case "map":
		return f.Type
	case postgres.ArrayColumn:
		return f.ModelType
	}
	return f.Type
}

// scanVarName returns a safe variable name for scanning a field.
func scanVarName(f walker.Field) string {
	return "col_" + strings.ReplaceAll(f.ColumnName, ".", "_")
}

// setterPath converts an ObjectGetter path like "GetSignal().GetName()" into a
// setter assignment path. Returns the field path parts for building nested struct assignments.
// For example: "GetSignal().GetName()" -> ["Signal", "Name"]
func setterPath(f walker.Field) []string {
	getter := f.ObjectGetter.Value()
	if f.ObjectGetter.IsVariable() {
		return nil
	}
	// Split "GetSignal().GetName()" -> ["GetSignal()", "GetName()"]
	parts := strings.Split(getter, ".")
	var result []string
	for _, part := range parts {
		// Remove "Get" prefix and "()" suffix
		part = strings.TrimPrefix(part, "Get")
		part = strings.TrimSuffix(part, "()")
		result = append(result, part)
	}
	return result
}

// fieldSetterExpr returns the Go expression to set the field on "obj", e.g., "obj.Signal.Name".
// For variable fields (serialized, idx), returns empty string.
func fieldSetterExpr(f walker.Field) string {
	if f.ObjectGetter.IsVariable() {
		return ""
	}
	path := setterPath(f)
	return "obj." + strings.Join(path, ".")
}

// needsTypeConversion returns whether a field needs type conversion when scanning from DB.
func needsTypeConversion(f walker.Field) bool {
	switch f.DataType {
	case "datetime", "datetimetz":
		return true
	case "enum":
		return true
	case "messagebytes":
		return true
	}
	if f.SQLType == "uuid" && f.Type == "string" {
		return true
	}
	return false
}

// typeConversionExpr returns the Go expression to convert a scanned value to the proto field type.
func typeConversionExpr(f walker.Field, varName string) string {
	switch f.DataType {
	case "datetime", "datetimetz":
		return fmt.Sprintf("protocompat.ConvertTimeToTimestampOrNil(%s)", varName)
	case "enum":
		return fmt.Sprintf("%s(%s)", f.Type, varName)
	case "messagebytes":
		return fmt.Sprintf("pgutils.MustUnmarshalRepeatedMessages(%s, func() *%s { return &%s{} })",
			varName, f.MessageBytesElemType, f.MessageBytesElemType)
	}
	return varName
}

// canScanDirect returns whether a field can be scanned directly into the proto
// struct field, without needing an intermediate variable and type conversion.
func canScanDirect(f walker.Field) bool {
	if f.ObjectGetter.IsVariable() {
		return false
	}
	return !needsTypeConversion(f)
}

// canUnnest returns whether a field type can be used in a multi-arg unnest().
// 2D arrays (stringarray, enumarray, intarray) and maps don't work with unnest's
// parallel-array iteration because unnest flattens them instead of producing subarrays.
// ArrayColumn fields can't be unnested either — they need per-row UPDATE fallback.
func canUnnest(f walker.Field) bool {
	switch f.DataType {
	case "stringarray", "enumarray", "intarray", "map", postgres.ArrayColumn:
		return false
	}
	return true
}

// pgArrayCast returns the Postgres array type cast for use in unnest(), e.g., "uuid[]", "text[]".
func pgArrayCast(f walker.Field) string {
	switch f.SQLType {
	case "uuid":
		return "uuid[]"
	case "cidr":
		return "cidr[]"
	case "bytea":
		return "bytea[]"
	}
	switch f.DataType {
	case "datetime":
		return "timestamp[]"
	case "datetimetz":
		return "timestamptz[]"
	case "bool":
		return "bool[]"
	case "integer", "enum":
		return "int[]"
	case "biginteger":
		return "bigint[]"
	case "numeric":
		return "numeric[]"
	case "stringarray":
		return "text[][]"
	case "enumarray":
		return "int[][]"
	case "intarray":
		return "int[][]"
	case "map":
		return "jsonb[]"
	}
	return "text[]"
}

// unnestArrayGoType returns the Go type for a slice of scan values, e.g., "[]string", "[]*time.Time".
func unnestArrayGoType(f walker.Field) string {
	return "[]" + scanVarType(f)
}

// unnestAppendExpr returns the Go expression to append a value from obj to the array.
// This mirrors the insertValues template but for array accumulation.
func unnestAppendExpr(f walker.Field) string {
	getter := f.Getter("obj")
	switch {
	case f.DataType == postgres.ArrayColumn:
		return arrayExtractExpr(f)
	case f.DataType == "messagebytes":
		return fmt.Sprintf("pgutils.MustMarshalRepeatedMessages(%s)", getter)
	case f.DataType == "datetime" || f.DataType == "datetimetz":
		return fmt.Sprintf("protocompat.NilOrTime(%s)", getter)
	case f.SQLType == "uuid":
		return getter // pgx handles string -> uuid cast via SQL
	case f.SQLType == "cidr":
		return getter
	case f.DataType == "map":
		return fmt.Sprintf("pgutils.EmptyOrMap(%s)", getter)
	case f.DataType == "string" && f.Options.Reference != nil && f.Options.Reference.Nullable:
		return getter
	default:
		return getter
	}
}

func isMessageBytes(f walker.Field) bool {
	return f.DataType == "messagebytes"
}

var funcMap = template.FuncMap{
	"arr":                          arr,
	"lowerCamelCase":               lowerCamelCase,
	"upperCamelCase":               upperCamelCase,
	"valueExpansion":               valueExpansion,
	"lowerCase":                    strings.ToLower,
	"concatWith":                   concatWith,
	"searchFieldNameInOtherSchema": searchFieldNameInOtherSchema,
	"isSacScoping":                 isSacScoping,
	"dict":                         dict,
	"scanVarType":                  scanVarType,
	"scanVarName":                  scanVarName,
	"setterPath":                   setterPath,
	"joinPath":                     func(parts []string) string { return strings.Join(parts, ".") },
	"stripPointer":                 func(s string) string { return strings.TrimPrefix(s, "*") },
	"getterToSetter": func(getter string) string {
		// Convert "GetSignal().GetLineageInfo()" to "Signal.LineageInfo"
		parts := strings.Split(getter, ".")
		var result []string
		for _, part := range parts {
			part = strings.TrimPrefix(part, "Get")
			part = strings.TrimSuffix(part, "()")
			result = append(result, part)
		}
		return strings.Join(result, ".")
	},
	"fieldSetterExpr":  fieldSetterExpr,
	"canUnnest":        canUnnest,
	"isArrayColumn":    isArrayColumn,
	"arrayExtractExpr": arrayExtractExpr,
	"unnestableFields": func(fields []walker.Field) []walker.Field {
		var out []walker.Field
		for _, f := range fields {
			if canUnnest(f) {
				out = append(out, f)
			}
		}
		return out
	},
	"nonUnnestableFields": func(fields []walker.Field) []walker.Field {
		var out []walker.Field
		for _, f := range fields {
			if !canUnnest(f) {
				out = append(out, f)
			}
		}
		return out
	},
	"isMessageBytes":      isMessageBytes,
	"canScanDirect":       canScanDirect,
	"needsTypeConversion": needsTypeConversion,
	"typeConversionExpr":  typeConversionExpr,
	"pgArrayCast":         pgArrayCast,
	"unnestArrayGoType":   unnestArrayGoType,
	"unnestAppendExpr":    unnestAppendExpr,
	"pluralType": func(s string) string {
		if s[len(s)-1] == 'y' {
			return fmt.Sprintf("%sies", strings.TrimSuffix(s, "y"))
		}
		return fmt.Sprintf("%ss", s)
	},
}
