package generator

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
)

type schemaEntry struct {
	Data           typeData
	ListData       map[string]reflect.Type
	ExtraResolvers []string
}

func isListType(p reflect.Type) bool {
	if p == nil {
		return false
	}
	name := p.Name()
	if p.Kind() == reflect.Ptr {
		name = p.Elem().Name()
	}
	return isProto(p) && len(name) > 4 && name[0:4] == "List"
}

func makeSchemaEntries(data []typeData, extraResolvers map[string][]string) []schemaEntry {
	output := make([]schemaEntry, 0)
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
		if (td.Type == nil || isProto(td.Type) || isEnum(td.Type)) && !isListType(td.Type) {
			listRef := listRef[td.Name]
			se := schemaEntry{
				Data:           td,
				ListData:       listRef,
				ExtraResolvers: extraResolvers[td.Name],
			}
			output = append(output, se)
		}
	}
	return output
}

func schemaType(fd fieldData) string {
	if strings.HasPrefix(fd.Name, "XXX_") {
		return ""
	}
	if fd.Name == "Id" && fd.Type.Kind() == reflect.String {
		return "ID!"
	}
	return schemaExpand(fd.Type)
}

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
		if p.Elem().Kind() == reflect.String &&
			p.Elem().Kind() == reflect.String {
			return "[Label!]!"
		}
	case reflect.Ptr:
		if p == timestampType {
			return "Time"
		}
		if isProto(p) {
			return p.Elem().Name()
		}
		inner := schemaExpand(p.Elem())
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

// RegisterProtoEnum is a utility method used by the generated code to output enums
func RegisterProtoEnum(builder SchemaBuilder, typ reflect.Type) {
	m := proto.EnumValueMap(importedName(typ))
	values := make([]string, 0, len(m))
	for k := range m {
		values = append(values, k)
	}
	sort.Slice(values, func(i, j int) bool {
		return m[values[i]] < m[values[j]]
	})
	err := builder.AddEnumType(typ.Name(), values)
	if err != nil {
		panic(err)
	}
}
