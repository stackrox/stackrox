package walker

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStorageType struct {
	Id string `sql:"pk,id,type(uuid)"`
}

// One can specify a custom SQL type for the structure field
func TestClusterGetter(t *testing.T) {
	IdField := Field{SQLType: ""}
	schema := Walk(reflect.TypeOf(&TestStorageType{}), "test_table")

	for _, f := range schema.Fields {
		if f.Name == "Id" {
			IdField = f
		}
	}

	assert.Equal(t, IdField.SQLType, "uuid")
}
