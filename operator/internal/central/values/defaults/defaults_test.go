package defaults

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestCentralStaticDefaults(t *testing.T) {
	tests := map[string]struct {
		defaults   *platform.CentralSpec
		errorCheck require.ErrorAssertionFunc
	}{
		"empty defaults": {
			defaults:   &platform.CentralSpec{},
			errorCheck: require.NoError,
		},
		"non-empty defaults": {
			defaults: &platform.CentralSpec{Egress: &platform.Egress{}},
			errorCheck: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorContains(t, err, "is not empty")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tt.errorCheck(t, CentralStaticDefaults.DefaultingFunc(logr.Discard(), nil, nil, nil, tt.defaults))
		})
	}
}

const defaultFormat = "The default is: %s."

func TestCentralStaticDefaultsMatchesCRD(t *testing.T) {
	centralSpecSchema := loadCentralSpecSchema(t)

	t.Run("Defaults", func(t *testing.T) {
		checkStruct(t, staticDefaults, centralSpecSchema)
	})
}

func loadCentralSpecSchema(t *testing.T) chartutil.Values {
	data, err := os.ReadFile("../../../../bundle/manifests/platform.stackrox.io_centrals.yaml")
	require.NoError(t, err)

	var crd unstructured.Unstructured
	require.NoError(t, yaml.Unmarshal(data, &crd))

	onlyVersion := crd.Object["spec"].(map[string]any)["versions"].([]any)[0].(map[string]any)
	centralSpecSchema, err := chartutil.Values(onlyVersion).Table("schema.openAPIV3Schema.properties.spec")
	require.NoError(t, err)

	return centralSpecSchema
}

func checkStruct(t *testing.T, s any, schema chartutil.Values) {
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
				checkStruct(t, field.Interface(), table)
			case reflect.Ptr:
				if field.IsNil() {
					t.Skip("nil") // TODO(ROX-29467): check this case too
				}
				switch field.Type().Elem().Kind() {
				case reflect.Struct:
					table, err := schema.Table("properties." + jsonName)
					require.NoError(t, err)
					checkStruct(t, field.Elem().Interface(), table)
				case reflect.String:
					desc, err := schema.PathValue(fmt.Sprintf("properties.%s.description", jsonName))
					require.NoError(t, err)
					require.IsType(t, "string", desc, jsonName)
					checkLeafField(t, field, desc.(string))
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

func checkLeafField(t *testing.T, field reflect.Value, crdDescription string) {
	inCodeDefault := field.Elem() // TODO(ROX-29467): support more leaf field types
	inCodeDefaultDescription := fmt.Sprintf(defaultFormat, inCodeDefault)
	inCRDDefaultDescription := lastLine(crdDescription)
	require.Equal(t, inCodeDefaultDescription, inCRDDefaultDescription)
}

func lastLine(multiline string) string {
	lines := strings.Split(multiline, "\n")
	return lines[len(lines)-1]
}
