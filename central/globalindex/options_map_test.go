package globalindex

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertKindsEqual(t *testing.T, expected, actual reflect.Kind) {
	assert.Equal(t, expected, actual, "Expected kind to be %s, but found %s", expected, actual)
}

func assertElementSatisfiesSearchDataType(t *testing.T, obj interface{}, searchDataType v1.SearchDataType) {
	typ := reflect.TypeOf(obj)
	// Sometimes we index a slice of primitive types.
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	kind := typ.Kind()
	typ.Name()

	switch searchDataType {
	case v1.SearchDataType_SEARCH_BOOL:
		assertKindsEqual(t, reflect.Bool, kind)
	case v1.SearchDataType_SEARCH_NUMERIC:
		// All numeric fields we index happen to be float32s right now.
		// Update this check to cover more types if that changes.
		assertKindsEqual(t, reflect.Float32, kind)
	case v1.SearchDataType_SEARCH_STRING:
		assertKindsEqual(t, reflect.String, kind)
	case v1.SearchDataType_SEARCH_DATETIME:
		assertKindsEqual(t, reflect.Int64, kind)
	// These two are enum types which marshal to int32s.
	// We also explicitly expect them to be the enum types.
	case v1.SearchDataType_SEARCH_ENFORCEMENT:
		assertKindsEqual(t, reflect.Int32, kind)
		assert.Equal(t, "EnforcementAction", typ.Name())
	case v1.SearchDataType_SEARCH_SEVERITY:
		assertKindsEqual(t, reflect.Int32, kind)
		assert.Equal(t, "Severity", typ.Name())
	case v1.SearchDataType_SEARCH_MAP:
		assertKindsEqual(t, reflect.Map, kind)
	default:
		t.Fatalf("Search type %s unknown, please update this test to handle it", searchDataType)
	}
}

func assertElementAtJSONPathExistsAndIsOfType(t *testing.T, obj interface{}, path []string, searchDataType v1.SearchDataType) {
	require.True(t, len(path) > 0)

	typ := reflect.TypeOf(obj)

	// We keep dereferencing pointers and slices until we get to a struct.
	// Examples:
	// *v1.Image -> v1.Image
	// []*v1.Container -> *v1.Container -> v1.Container
	for typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}

	require.Equal(t, typ.Kind(), reflect.Struct, "Object %#v (kind: %s) is not a struct (looking for path %#v)",
		obj, typ.Kind(), path)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := field.Tag.Get("json")
		// All proto objects have the json tag set to "<tag>,omitempty"
		id := strings.TrimSuffix(jsonTag, ",omitempty")
		if id == "" {
			id = field.Name
		}

		if id == path[0] {
			zeroElem := reflect.Zero(typ).Field(i).Interface()
			if zeroElem == nil {
				return // This helps with interfaces until we can support them more fully
			}

			if len(path) > 1 {
				assertElementAtJSONPathExistsAndIsOfType(t, zeroElem, path[1:], searchDataType)
			} else {
				assertElementSatisfiesSearchDataType(t, zeroElem, searchDataType)
			}
			return
		}
	}
	t.Fatalf("Couldn't find object at path '%s' of %#v", strings.Join(path, "."), obj)
}

// This tests all the options map to make sure that all the search fields are valid for the object being indexed.
// This involves verifying that the JSON path exists, and that the element at that JSON path has a type compatible
// with the type specified in the search field.
func TestCategoryToOptionsMap(t *testing.T) {
	categoryToMetadata := map[v1.SearchCategory]struct {
		// objName represents the name of the object used in the wrap objects we index.
		// It is always the first element of the field path.
		objName string
		// This represents the proto object that's being indexed.
		protoObj interface{}
	}{
		v1.SearchCategory_ALERTS:             {"alert", v1.Alert{}},
		v1.SearchCategory_DEPLOYMENTS:        {"deployment", v1.Deployment{}},
		v1.SearchCategory_IMAGES:             {"image", v1.Image{}},
		v1.SearchCategory_POLICIES:           {"policy", v1.Policy{}},
		v1.SearchCategory_SECRETS:            {"secret", v1.Secret{}},
		v1.SearchCategory_PROCESS_INDICATORS: {"process_indicator", v1.ProcessIndicator{}},
	}

	for category, optionsMap := range CategoryToOptionsMap {
		t.Run(category.String(), func(t *testing.T) {

			for option, searchField := range optionsMap {
				t.Run(option.String(), func(t *testing.T) {
					// Basic checks
					require.NotEqual(t, searchField.GetCategory(), v1.SearchCategory_SEARCH_UNSET)
					require.NotEmpty(t, searchField.GetFieldPath())

					splitFieldPath := strings.Split(searchField.GetFieldPath(), ".")
					require.NotEmpty(t, splitFieldPath)

					metadata, ok := categoryToMetadata[searchField.GetCategory()]
					require.True(t, ok, "Please update the metadata in this test to support category: %s", category)

					assert.Equal(t, metadata.objName, splitFieldPath[0], "The field path for category %s must start with %s", category, metadata.objName)
					assertElementAtJSONPathExistsAndIsOfType(t, metadata.protoObj, splitFieldPath[1:], searchField.GetType())
				})
			}
		})
	}
}
