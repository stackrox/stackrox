package search

import (
	"fmt"
	"reflect"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stackrox/rox/pkg/stringutils"
)

type searchWalker struct {
	category v1.SearchCategory
	fields   map[FieldLabel]*Field
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(category v1.SearchCategory, prefix string, obj interface{}) OptionsMap {
	sw := searchWalker{
		category: category,
		fields:   make(map[FieldLabel]*Field),
	}
	sw.walkRecursive(prefix, reflect.TypeOf(obj))
	return OptionsMapFromMap(category, sw.fields)
}

func (s *searchWalker) getSearchField(path, tag string) (string, *Field) {
	if tag == "" {
		return "", nil
	}
	fields := strings.Split(tag, ",")
	if !IsValidFieldLabel(fields[0]) {
		log.Panicf("Field %q is not a valid FieldLabel. You may need to add it to pkg/search/options.go", fields[0])
	}

	fieldName := fields[0]
	var hidden, store bool
	var analyzer string
	if len(fields) > 1 {
		for _, f := range fields[1:] {
			switch f {
			case "hidden":
				hidden = true
			case "store":
				store = true
			default:
				if strings.HasPrefix(f, "analyzer=") {
					spl := strings.Split(f, "=")
					if len(spl) != 2 {
						log.Fatalf("Invalid analyzer struct annotation %q on field %q", f, fieldName)
					}
					analyzer = spl[1]
				} else {
					log.Fatalf("Field %s in search annotation is invalid", f)
				}
			}
		}
	}

	return fieldName, &Field{
		FieldPath: path,
		Store:     store,
		Hidden:    hidden,
		Category:  s.category,
		Analyzer:  analyzer,
	}
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *searchWalker) handleStruct(prefix string, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)

		// Currently only proto structs are supported. We need to skip
		// internal fields, because they contain recursive references.
		if protoreflect.IsInternalGeneratorField(field) {
			continue
		}

		jsonTag := strings.TrimSuffix(field.Tag.Get("json"), ",omitempty")
		if jsonTag == "-" {
			continue
		} else if jsonTag == "" { // If no JSON tag, then Bleve takes the field name
			jsonTag = field.Name
		}
		fullPath := fmt.Sprintf("%s.%s", prefix, jsonTag)
		searchTag := field.Tag.Get("search")
		if searchTag == "-" {
			continue
		}
		if strings.HasPrefix(searchTag, "flag=") {
			flag := stringutils.GetAfter(searchTag, "=")
			ff, ok := features.Flags[flag]
			if !ok {
				log.Fatalf("flag %s is not a valid feature flag", flag)
			}
			if !ff.Enabled() {
				continue
			}
			searchTag = ""
		}

		// Special case proto timestamp because we actually want to index seconds
		if field.Type == protocompat.TimestampPtrType {
			fieldName, searchField := s.getSearchField(fullPath+".seconds", searchTag)
			if searchField == nil {
				continue
			}
			searchField.Type = v1.SearchDataType_SEARCH_DATETIME
			s.fields[FieldLabel(fieldName)] = searchField
			continue
		}
		// For oneof fields, get the types and return values as a slice of
		// interfaces that are nil type pointers.
		if field.Tag.Get("protobuf_oneof") != "" {
			oneOfFieldTypes := protocompat.GetOneOfTypesByFieldIndex(original, i)
			for _, oneOfFieldType := range oneOfFieldTypes {
				s.walkRecursive(fullPath, oneOfFieldType)
			}
			continue
		}

		searchDataType := s.walkRecursive(fullPath, field.Type)
		fieldName, searchField := s.getSearchField(fullPath, searchTag)
		if searchField == nil {
			continue
		}
		if searchDataType < 0 {
			panic(fmt.Sprintf("SearchDataType for field %s is invalid", fieldName))
		}
		searchField.Type = searchDataType
		s.fields[FieldLabel(fieldName)] = searchField
	}
}

func (s *searchWalker) walkRecursive(prefix string, original reflect.Type) v1.SearchDataType {
	switch original.Kind() {
	case reflect.Ptr, reflect.Slice:
		return s.walkRecursive(prefix, original.Elem())
	case reflect.Struct:
		s.handleStruct(prefix, original)
	case reflect.Map:
		return v1.SearchDataType_SEARCH_MAP
	case reflect.String:
		return v1.SearchDataType_SEARCH_STRING
	case reflect.Bool:
		return v1.SearchDataType_SEARCH_BOOL
	case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		enum, ok := reflect.Zero(original).Interface().(protoreflect.ProtoEnum)
		if !ok {
			return v1.SearchDataType_SEARCH_NUMERIC
		}
		enumDesc, err := protoreflect.GetEnumDescriptor(enum)
		if err != nil {
			panic(err)
		}
		enumregistry.Add(prefix, enumDesc)
		return v1.SearchDataType_SEARCH_ENUM
	case reflect.Interface:
		return v1.SearchDataType_SEARCH_STRING
	}
	// TODO(ROX-9291): Add unknown SearchDataType to the enum, move enum definition to go struct.
	return v1.SearchDataType(-1)
}
