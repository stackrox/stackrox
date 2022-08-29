package search

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/protowalk"
	"github.com/stackrox/rox/pkg/search/enumregistry"
)

type searchWalker struct {
	category v1.SearchCategory
	prefix   string
	fields   map[FieldLabel]*Field
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(category v1.SearchCategory, prefix string, obj interface{}) OptionsMap {
	sw := searchWalker{
		category: category,
		prefix:   prefix,
		fields:   make(map[FieldLabel]*Field),
	}
	protowalk.WalkProto(reflect.TypeOf(obj), sw.handleField)
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

var (
	timestampType = reflect.TypeOf((*types.Timestamp)(nil))
)

func (s *searchWalker) handleField(fp protowalk.FieldPath) bool {
	field := fp.Field()
	if field.JSONName() == "-" {
		return false
	}
	searchTag := field.Tag.Get("search")
	if searchTag == "-" {
		return false
	}

	fullPath := s.prefix + "." + fp.JSONPath()
	if field.ElemType() == timestampType {
		fieldName, searchField := s.getSearchField(fullPath+".seconds", searchTag)
		if searchField == nil {
			return false
		}
		searchField.Type = v1.SearchDataType_SEARCH_DATETIME
		s.fields[FieldLabel(fieldName)] = searchField
		return false
	}
	if !field.IsLeaf() {
		return true
	}

	fieldName, searchField := s.getSearchField(fullPath, searchTag)
	if searchField == nil {
		return false
	}

	searchField.Type = s.handleLeafField(fullPath, field.ElemType())
	s.fields[FieldLabel(fieldName)] = searchField
	return true
}

func (s *searchWalker) handleLeafField(path string, ty reflect.Type) v1.SearchDataType {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	switch ty.Kind() {
	case reflect.Map:
		return v1.SearchDataType_SEARCH_MAP
	case reflect.String:
		return v1.SearchDataType_SEARCH_STRING
	case reflect.Bool:
		return v1.SearchDataType_SEARCH_BOOL
	case reflect.Uint32, reflect.Uint64, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		enum, ok := reflect.Zero(ty).Interface().(protoreflect.ProtoEnum)
		if !ok {
			return v1.SearchDataType_SEARCH_NUMERIC
		}
		enumDesc, err := protoreflect.GetEnumDescriptor(enum)
		if err != nil {
			panic(err)
		}
		enumregistry.Add(path, enumDesc)
		return v1.SearchDataType_SEARCH_ENUM
	}
	panic(fmt.Sprintf("unknown leaf field type %s at %s", ty.Name(), path))
}
