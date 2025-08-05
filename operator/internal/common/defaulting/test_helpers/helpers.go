package test_helpers

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const defaultPrefix = "The default is: "
const defaultFormat = defaultPrefix + "%s."

func LoadSpecSchema(t *testing.T, resource string) chartutil.Values {
	data, err := os.ReadFile("../../../../bundle/manifests/platform.stackrox.io_" + resource + ".yaml")
	require.NoError(t, err)

	var crd unstructured.Unstructured
	require.NoError(t, yaml.Unmarshal(data, &crd))

	versionSpecs := crd.Object["spec"].(map[string]any)["versions"].([]any)
	require.Len(t, versionSpecs, 1, "expected a single API version spec")
	onlyVersion := versionSpecs[0].(map[string]any)
	centralSpecSchema, err := chartutil.Values(onlyVersion).Table("schema.openAPIV3Schema.properties.spec")
	require.NoError(t, err)

	return centralSpecSchema
}

func CheckStruct(t *testing.T, s any, schema chartutil.Values) {
	structValue := reflect.ValueOf(s)
	requireNoDefaultProperty(t, schema)
	for i := 0; i < structValue.NumField(); i++ {
		structField := structValue.Type().Field(i)

		t.Run(structField.Name, func(t *testing.T) {
			field := structValue.Field(i)
			jsonName, embedded := getJSONName(t, structField)
			if embedded {
				CheckStruct(t, field.Interface(), schema)
				return
			}
			fieldSchema, err := schema.Table("properties." + jsonName)
			require.NoError(t, err)
			switch field.Type().Kind() {
			case reflect.Ptr:
				switch field.Type().Elem().Kind() {
				case reflect.Struct:
					if field.IsNil() {
						// Operator code provides no defaults for this subtree.
						// Make sure the schema does not mention defaults either.
						checkObjectNoDefaults(t, fieldSchema)
					} else {
						CheckStruct(t, field.Elem().Interface(), fieldSchema)
					}
				case reflect.String, reflect.Bool, reflect.Int32:
					checkPtrLeafField(t, field, fieldSchema)
				default:
					t.Fatalf("unsupported type %q", field.Type().Elem().Kind())
				}
			case reflect.Map, reflect.Slice:
				// Currently we keep maps and slices empty by default.
				requireNoDefaultProperty(t, fieldSchema)
			default:
				t.Fatalf("unsupported field type %q, expected pointer, map or slice", field.Type().Kind())
			}
		})
	}
}

func requireNoDefaultProperty(t *testing.T, schema chartutil.Values) {
	require.NotContainsf(t, schema, "default", "expected schema to not define a default value, but got %v", schema["default"])
}

func checkObjectNoDefaults(t *testing.T, schema chartutil.Values) {
	requireNoDefaultProperty(t, schema)
	typeStr, err := schema.PathValue("type")
	require.NoError(t, err)
	require.Equal(t, "object", typeStr)
	checkNoDefaultsInSchema(t, schema)
	if _, hasAdditionalProps := schema["additionalProperties"]; hasAdditionalProps {
		// Some objects are in fact just fancy leaf fields. In that case checking the description (which we did above)
		// is enough.
		additionalProps, err := schema.Table("additionalProperties")
		require.NoError(t, err)
		if t, hasType := additionalProps["type"]; hasType && t.(string) == "string" {
			return
		}
		if t, hasIntOrString := additionalProps["x-kubernetes-int-or-string"]; hasIntOrString && t.(bool) {
			return
		}
		t.Fatalf("unrecognized object with additional properties: %v", additionalProps)
	}
	properties, err := schema.Table("properties")
	require.NoErrorf(t, err, "%+v", schema)
	for k, v := range properties {
		prop := chartutil.Values(v.(map[string]interface{}))
		t.Run(k, func(t *testing.T) {
			propType, err := prop.PathValue("type")
			require.NoError(t, err)
			switch propType.(string) {
			case "object":
				checkObjectNoDefaults(t, prop)
			case "string", "boolean", "integer":
				checkNoDefaultsInSchema(t, prop)
			case "array":
				if mapType, found := prop["x-kubernetes-list-type"]; found && mapType.(string) == "map" && k == "claims" {
					// We have no influence over built-in types defaults.
					// Nothing to do here.
					t.SkipNow()
				} else {
					// Currently we keep maps and slices empty by default.
					checkNoDefaultsInSchema(t, prop)
				}
			default:
				t.Fatalf("unrecognized type %q: %v", propType, v)
			}
		})
	}
}

func checkNoDefaultsInSchema(t *testing.T, schema chartutil.Values) {
	requireNoDefaultProperty(t, schema)
	desc, err := schema.PathValue("description")
	require.NoError(t, err)
	require.False(t, strings.HasPrefix(desc.(string), defaultPrefix))
}

func getJSONName(t *testing.T, structField reflect.StructField) (field string, embedded bool) {
	jsonName, rest, found := strings.Cut(structField.Tag.Get("json"), ",")
	if found && jsonName == "" && rest == "inline" {
		return "", true
	}
	require.NotEmpty(t, jsonName, "field %s should have a 'json' tag or be inline", structField.Name)
	return jsonName, false
}

func checkPtrLeafField(t *testing.T, field reflect.Value, schema chartutil.Values) {
	if field.IsNil() {
		// Operator code specifies no default for this field.
		// Make sure the schema does not mention one either.
		require.Falsef(t, strings.HasPrefix(lastDescriptionLine(t, schema), defaultPrefix), "unexpected default in schema %v", schema)
		return
	}
	checkLeafField(t, field.Elem(), schema)
}

func checkLeafField(t *testing.T, inCodeDefault reflect.Value, schema chartutil.Values) {
	requireNoDefaultProperty(t, schema)
	var inCodeDefaultString string
	switch inCodeDefault.Kind() {
	case reflect.String:
		inCodeDefaultString = inCodeDefault.String()
	case reflect.Bool:
		inCodeDefaultString = strconv.FormatBool(inCodeDefault.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		inCodeDefaultString = strconv.FormatInt(inCodeDefault.Int(), 10)
	default:
		t.Fatalf("unsupported type %v", inCodeDefault.Kind())
	}
	inCodeDefaultDescription := fmt.Sprintf(defaultFormat, inCodeDefaultString)
	inCRDDefaultDescription := lastDescriptionLine(t, schema)
	require.Equal(t, inCodeDefaultDescription, inCRDDefaultDescription)
}

func lastDescriptionLine(t *testing.T, schema chartutil.Values) string {
	crdDescription, err := schema.PathValue("description")
	require.NoError(t, err)
	require.IsType(t, "", crdDescription)
	return lastLine(crdDescription.(string))
}

func lastLine(multiline string) string {
	lines := strings.Split(multiline, "\n")
	return lines[len(lines)-1]
}
