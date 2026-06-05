package codegen

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/stackrox/rox/pkg/protocompat"
)

type schemaEntry struct {
	Data     typeData
	ListData map[string]reflect.Type
}

func isListType(p reflect.Type) bool {
	if p == nil {
		return false
	}
	name := p.Name()
	if p.Kind() == reflect.Pointer {
		name = p.Elem().Name()
	}
	return isProto(p) && len(name) > 4 && name[0:4] == "List"
}

func makeSchemaEntries(data []typeData) []schemaEntry {
	var output []schemaEntry
	listRef := make(map[string]map[string]reflect.Type)
	for _, td := range data {
		if isListType(td.Type) {
			listFields := make(map[string]reflect.Type)
			for _, f := range td.FieldData {
				listFields[f.Name] = f.Type
			}
			listRef[td.Name[4:]] = listFields
		}
	}

	for _, td := range data {
		// Skip unexported types.
		if name := td.Type.Name(); len(name) > 0 && unicode.IsLower(rune(name[0])) {
			continue
		}

		// Skip scalars and maps -- except for enums.
		if !isEnum(td.Type) {
			if kind := td.Type.Kind(); kind <= reflect.Complex128 || kind == reflect.Map {
				continue
			}
		}

		// Skip list types.
		if isListType(td.Type) {
			continue
		}

		output = append(output, schemaEntry{
			Data:     td,
			ListData: listRef[td.Name],
		})
	}
	return output
}

func schemaType(fd fieldData) string {
	if strings.ToLower(fd.Name) == "id" && fd.Type.Kind() == reflect.String {
		return "ID!"
	}
	return schemaExpand(fd.Type)
}

// hasSchemaFields reports whether the entry will produce at least one GraphQL field.
// GraphQL spec requires every object type to define one or more fields.
func hasSchemaFields(entry schemaEntry) bool {
	for _, fd := range entry.Data.FieldData {
		if schemaType(fd) != "" {
			return true
		}
	}
	return len(entry.Data.UnionData) > 0
}

// TODO(ROX-34963): Add support for []byte fields in GraphQL schema generation.
// Proto fields of type `bytes` (Go []byte) are silently dropped because uint8
// has no GraphQL scalar mapping, causing types like CosignSignature to appear
// empty. A proper fix would map []byte to a String scalar (base64-encoded) and
// update the resolver codegen to emit encoding/decoding wrappers.

func schemaExpand(p reflect.Type) string {
	switch p.Kind() {
	case reflect.String:
		return "String!"
	case reflect.Int32:
		if isEnum(p) {
			return p.Name() + "!"
		}
		return "Int!"
	case reflect.Int64:
		return "Int!"
	case reflect.Uint32:
		return "Int!"
	case reflect.Float32:
		return "Float!"
	case reflect.Float64:
		return "Float!"
	case reflect.Bool:
		return "Boolean!"
	case reflect.Slice:
		inner := schemaExpand(p.Elem())
		if inner != "" {
			return fmt.Sprintf("[%s]!", inner)
		}
		return ""
	case reflect.Map:
		if p.Key().Kind() == reflect.String && p.Elem().Kind() == reflect.String {
			return "[Label!]!"
		}
	case reflect.Pointer:
		if p == protocompat.TimestampPtrType {
			return "Time"
		}
		elem := p.Elem()
		if elem.Kind() == reflect.Struct {
			return elem.Name()
		}
		inner := schemaExpand(elem)
		if inner == "" {
			return ""
		}
		if strings.HasSuffix(inner, "!") {
			return inner[:len(inner)-1]
		}
		return inner
	}
	return ""
}
