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

func scanVarType(f walker.Field) string {
	if f.ObjectGetter.IsVariable() {
		return f.Type
	}
	switch f.DataType {
	case postgres.DateTime, postgres.DateTimeTZ:
		return "*time.Time"
	case postgres.Enum:
		return "int32"
	case postgres.Integer:
		return "int32"
	case postgres.BigInteger:
		if f.Type == "uint32" || f.Type == "uint64" {
			return f.Type
		}
		return "int64"
	case postgres.Numeric:
		if f.Type == "uint64" {
			return "uint64"
		}
		if f.Type == "float32" {
			return "float32"
		}
		return "float64"
	case postgres.Bool:
		return "bool"
	case postgres.String:
		return "string"
	case postgres.StringArray:
		return "[]string"
	case postgres.Bytes, postgres.MessageBytes:
		return "[]byte"
	case postgres.EnumArray, postgres.IntArray:
		return "[]int32"
	case postgres.Map:
		return f.Type
	}
	return f.Type
}

func scanVarName(f walker.Field) string {
	return "col_" + strings.ReplaceAll(f.ColumnName, ".", "_")
}

func setterPath(f walker.Field) []string {
	if f.ObjectGetter.IsVariable() {
		return nil
	}
	parts := strings.Split(f.ObjectGetter.Value(), ".")
	var result []string
	for _, part := range parts {
		part = strings.TrimPrefix(part, "Get")
		part = strings.TrimSuffix(part, "()")
		result = append(result, part)
	}
	return result
}

func fieldSetterExpr(f walker.Field) string {
	if f.ObjectGetter.IsVariable() {
		return ""
	}
	path := setterPath(f)
	return "obj." + strings.Join(path, ".")
}

func needsTypeConversion(f walker.Field) bool {
	switch f.DataType {
	case postgres.DateTime, postgres.DateTimeTZ:
		return true
	case postgres.Enum:
		return true
	case postgres.MessageBytes:
		return true
	}
	if f.SQLType == "uuid" && f.Type == "string" {
		return true
	}
	return false
}

func typeConversionExpr(f walker.Field, varName string) string {
	switch f.DataType {
	case postgres.DateTime, postgres.DateTimeTZ:
		return fmt.Sprintf("protocompat.ConvertTimeToTimestampOrNil(%s)", varName)
	case postgres.Enum:
		return fmt.Sprintf("%s(%s)", f.Type, varName)
	case postgres.MessageBytes:
		return fmt.Sprintf("pgutils.MustUnmarshalRepeatedMessages(%s, func() *%s { return &%s{} })",
			varName, f.MessageBytesElemType, f.MessageBytesElemType)
	}
	return varName
}

func canScanDirect(f walker.Field) bool {
	if f.ObjectGetter.IsVariable() {
		return false
	}
	return !needsTypeConversion(f)
}

func canUnnest(f walker.Field) bool {
	switch f.DataType {
	case postgres.StringArray, postgres.EnumArray, postgres.IntArray, postgres.Map:
		return false
	}
	return true
}

func isMessageBytes(f walker.Field) bool {
	return f.DataType == postgres.MessageBytes
}

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
	case postgres.DateTime:
		return "timestamp[]"
	case postgres.DateTimeTZ:
		return "timestamptz[]"
	case postgres.Bool:
		return "bool[]"
	case postgres.Integer, postgres.Enum:
		return "int[]"
	case postgres.BigInteger:
		return "bigint[]"
	case postgres.Numeric:
		return "numeric[]"
	case postgres.StringArray:
		return "text[][]"
	case postgres.EnumArray, postgres.IntArray:
		return "int[][]"
	case postgres.Map:
		return "jsonb[]"
	}
	return "text[]"
}

func unnestArrayGoType(f walker.Field) string {
	return "[]" + scanVarType(f)
}

func unnestAppendExpr(f walker.Field) string {
	getter := f.Getter("obj")
	switch {
	case f.DataType == postgres.MessageBytes:
		return fmt.Sprintf("pgutils.MustMarshalRepeatedMessages(%s)", getter)
	case f.DataType == postgres.DateTime || f.DataType == postgres.DateTimeTZ:
		return fmt.Sprintf("protocompat.NilOrTime(%s)", getter)
	case f.DataType == postgres.Enum:
		return fmt.Sprintf("int32(%s)", getter)
	case f.SQLType == "uuid":
		return getter
	case f.SQLType == "cidr":
		return getter
	case f.DataType == postgres.Map:
		return fmt.Sprintf("pgutils.EmptyOrMap(%s)", getter)
	case f.DataType == postgres.String && f.Options.Reference != nil && f.Options.Reference.Nullable:
		return getter
	default:
		return getter
	}
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
	"pluralType": func(s string) string {
		if s[len(s)-1] == 'y' {
			return fmt.Sprintf("%sies", strings.TrimSuffix(s, "y"))
		}
		return fmt.Sprintf("%ss", s)
	},
	"scanVarType":         scanVarType,
	"scanVarName":         scanVarName,
	"setterPath":          setterPath,
	"joinPath":            func(parts []string) string { return strings.Join(parts, ".") },
	"stripPointer":        func(s string) string { return strings.TrimPrefix(s, "*") },
	"fieldSetterExpr":     fieldSetterExpr,
	"canUnnest":           canUnnest,
	"isMessageBytes":      isMessageBytes,
	"canScanDirect":       canScanDirect,
	"needsTypeConversion": needsTypeConversion,
	"typeConversionExpr":  typeConversionExpr,
	"pgArrayCast":         pgArrayCast,
	"unnestArrayGoType":   unnestArrayGoType,
	"unnestAppendExpr":    unnestAppendExpr,
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
	"getterToSetter": func(getter string) string {
		parts := strings.Split(getter, ".")
		var result []string
		for _, part := range parts {
			part = strings.TrimPrefix(part, "Get")
			part = strings.TrimSuffix(part, "()")
			result = append(result, part)
		}
		return strings.Join(result, ".")
	},
}
