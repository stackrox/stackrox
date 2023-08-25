package walker

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStorageType struct {
	ID string `sql:"pk,id,type(uuid)"`
}

// One can specify a custom SQL type for the structure field
func TestClusterGetter(t *testing.T) {
	IDField := Field{SQLType: ""}
	schema := Walk(reflect.TypeOf(&TestStorageType{}), "test_table")

	for _, f := range schema.Fields {
		if f.Name == "ID" {
			IDField = f
		}
	}

	assert.Equal(t, IDField.SQLType, "uuid")
}
