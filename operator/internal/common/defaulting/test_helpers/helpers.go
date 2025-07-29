package test_helpers

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const defaultFormat = "The default is: %s."

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
	for i := 0; i < structValue.NumField(); i++ {
		structField := structValue.Type().Field(i)

		t.Run(structField.Name, func(t *testing.T) {
			field := structValue.Field(i)
			jsonName := getJSONName(t, structField)
			switch field.Type().Kind() {
			case reflect.Struct:
				table, err := schema.Table("properties." + jsonName)
				require.NoError(t, err)
				CheckStruct(t, field.Interface(), table)
			case reflect.Ptr:
				if field.IsNil() {
					t.Skip("nil") // TODO(ROX-29467): check this case too
				}
				switch field.Type().Elem().Kind() {
				case reflect.Struct:
					table, err := schema.Table("properties." + jsonName)
					require.NoError(t, err)
					CheckStruct(t, field.Elem().Interface(), table)
				case reflect.String:
					desc, err := schema.PathValue(fmt.Sprintf("properties.%s.description", jsonName))
					require.NoError(t, err)
					require.IsType(t, "string", desc, jsonName)
					CheckLeafField(t, field, desc.(string))
				default:
					t.SkipNow() // TODO(ROX-29467): support more leaf field types
				}
			default:
				t.SkipNow() // TODO(ROX-29467): support more leaf field types
			}
		})
	}
}

func getJSONName(t *testing.T, structField reflect.StructField) string {
	jsonName, _, _ := strings.Cut(structField.Tag.Get("json"), ",")
	require.NotEmpty(t, jsonName, "field %s should have a 'json' tag", structField.Name)
	return jsonName
}

func CheckLeafField(t *testing.T, field reflect.Value, crdDescription string) {
	inCodeDefault := field.Elem() // TODO(ROX-29467): support more leaf field types
	inCodeDefaultDescription := fmt.Sprintf(defaultFormat, inCodeDefault)
	inCRDDefaultDescription := lastLine(crdDescription)
	require.Equal(t, inCodeDefaultDescription, inCRDDefaultDescription)
}

func lastLine(multiline string) string {
	lines := strings.Split(multiline, "\n")
	return lines[len(lines)-1]
}
