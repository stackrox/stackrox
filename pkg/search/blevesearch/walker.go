package blevesearch

import (
	"fmt"
	"reflect"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
)

const (
	searchXRefTag  = "search_xref"
	searchIndexTag = "search_index"
)

var (
	log = logging.LoggerForModule()

	typeToSearchCategory = map[string]v1.SearchCategory{
		"Image":            v1.SearchCategory_IMAGES,
		"Deployment":       v1.SearchCategory_DEPLOYMENTS,
		"ProcessIndicator": v1.SearchCategory_PROCESS_INDICATORS,
		"Secret":           v1.SearchCategory_SECRETS,
		"ServiceAccount":   v1.SearchCategory_SERVICE_ACCOUNTS,
		"Alert":            v1.SearchCategory_ALERTS,
		"Policy":           v1.SearchCategory_POLICIES,
		"Role":             v1.SearchCategory_ROLES,
		"Role Binding":     v1.SearchCategory_ROLEBINDINGS,
	}
)

type searchWalker struct {
	category v1.SearchCategory
	fields   map[search.FieldLabel]*v1.SearchField
}

// Walk iterates over the obj and creates a search.Map object from the found struct tags
func Walk(category v1.SearchCategory, prefix string, obj interface{}) search.OptionsMap {
	sw := searchWalker{
		category: category,
		fields:   make(map[search.FieldLabel]*v1.SearchField),
	}
	sw.walkRecursive(prefix, reflect.TypeOf(obj))
	return search.OptionsMapFromMap(sw.fields)
}

func (s *searchWalker) getSearchField(path, tag string) (string, *v1.SearchField) {
	if tag == "" {
		return "", nil
	}
	fields := strings.Split(tag, ",")
	if !search.FieldLabelSet.Contains(fields[0]) {
		log.Panicf("Field %q is not a valid FieldLabel. You may need to add it pkg/search/options.go", fields[0])
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

	return fieldName, &v1.SearchField{
		FieldPath: path,
		Store:     store,
		Hidden:    hidden,
		Category:  s.category,
		Analyzer:  analyzer,
	}
}

// handleXRef is used when a search_xref is found. This means that we want to show these as search options
// for the top level object, but do not want them to be prefixed with the top level prefix.
// e.g. showing "Image Tag", and having the search field path be image.name.tag instead of deployment.containers.image.name.tag
func (s *searchWalker) handleXRef(prefix string, t reflect.Type, tag string) {
	spl := strings.Split(tag, ",")
	if len(spl) == 0 {
		log.Fatalf("Found empty %s tag at %q", searchXRefTag, prefix)
	}

	searchCategory, ok := typeToSearchCategory[t.Elem().Name()]
	if !ok {
		log.Fatalf("Type %q is not in type to search category mapping", t.Elem().Name())
	}

	searchMap := Walk(searchCategory, prefix, reflect.New(t).Interface())
	if len(spl) == 1 && spl[0] == "all" {
		for k, v := range searchMap.Original() {
			s.fields[k] = v
		}
		return
	}
	for _, v := range spl {
		value, ok := searchMap.Get(v)
		if !ok {
			log.Fatalf("Could not find reference to field value %q in option map %q", v, prefix)
		}
		s.fields[search.FieldLabel(v)] = value
	}
}

// handleEmbeddedIndex is used when search_index is found. This means that we want to index specific fields
// from the sub object. e.g. alert.deployment.name should be indexed with the embedded path alert.deployment.name
// and should not refer to another external object
func (s *searchWalker) handleEmbeddedIndex(prefix string, t reflect.Type, tag string) {
	spl := strings.Split(tag, ",")
	if len(spl) == 0 {
		log.Fatalf("Found empty %s tag at %q", searchIndexTag, prefix)
	}
	searchMap := Walk(s.category, prefix, reflect.New(t).Interface())
	for _, v := range spl {
		searchField, ok := searchMap.Get(v)
		if !ok {
			log.Fatalf("Could not find reference to field value %q in option map %q", v, prefix)
		}
		s.fields[search.FieldLabel(v)] = searchField
	}
}

// handleStruct takes in a struct object and properly handles all of the fields
func (s *searchWalker) handleStruct(prefix string, original reflect.Type) {
	for i := 0; i < original.NumField(); i++ {
		field := original.Field(i)
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

		var hasExtraReference bool
		if searchXRefTag := field.Tag.Get(searchXRefTag); searchXRefTag != "" {
			hasExtraReference = true
			s.handleXRef(jsonTag, field.Type, searchXRefTag)
		}
		if searchIndexTag := field.Tag.Get(searchIndexTag); searchIndexTag != "" {
			hasExtraReference = true
			s.handleEmbeddedIndex(fullPath, field.Type, searchIndexTag)
		}
		if hasExtraReference {
			continue
		}

		// Special case proto timestamp because we actually want to index seconds
		if field.Type.String() == "*types.Timestamp" {
			fieldName, searchField := s.getSearchField(fullPath+".seconds", searchTag)
			if searchField == nil {
				continue
			}
			searchField.Type = v1.SearchDataType_SEARCH_DATETIME
			s.fields[search.FieldLabel(fieldName)] = searchField
			continue
		}
		// If it is a oneof then call XXX_OneofFuncs to get the types via the last returned value
		// The last returned value is a slice of interfaces that are nil type pointers
		if field.Tag.Get("protobuf_oneof") != "" {
			ptrToOriginal := reflect.PtrTo(original)
			method, ok := ptrToOriginal.MethodByName("XXX_OneofFuncs")
			if !ok {
				panic("XXX_OneofFuncs should exist for all protobuf oneofs")
			}
			out := method.Func.Call([]reflect.Value{reflect.New(original)})
			actualOneOfFields := out[3].Interface().([]interface{})
			for _, f := range actualOneOfFields {
				s.walkRecursive(fullPath, reflect.TypeOf(f))
			}
			continue
		}

		searchDataType := s.walkRecursive(fullPath, field.Type)
		fieldName, searchField := s.getSearchField(fullPath, searchTag)
		if searchField == nil {
			continue
		}
		searchField.Type = searchDataType
		s.fields[search.FieldLabel(fieldName)] = searchField
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
	default:
		panic(fmt.Sprintf("Type %s for field %s is not currently handled", original.Kind(), prefix))
	}
	return v1.SearchDataType_SEARCH_STRING
}
