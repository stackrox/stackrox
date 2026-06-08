package walker

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStorageType struct {
	ID string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk,id,type(uuid)"`
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

func TestSchemaRoot(t *testing.T) {
	grandparent := &Schema{Table: "grandparent"}
	parent := &Schema{Table: "parent", Parent: grandparent}
	child := &Schema{Table: "child", Parent: parent}

	assert.Equal(t, grandparent, grandparent.Root())
	assert.Equal(t, grandparent, parent.Root())
	assert.Equal(t, grandparent, child.Root())
}

func TestFieldIncludeNoSerialized(t *testing.T) {
	noSerSchema := &Schema{NoSerialized: true}
	normalSchema := &Schema{NoSerialized: false}

	cases := map[string]struct {
		field    Field
		expected bool
	}{
		"no-serialized: serialized column excluded": {
			field:    Field{Schema: noSerSchema, ColumnName: "serialized"},
			expected: false,
		},
		"no-serialized: regular field included": {
			field:    Field{Schema: noSerSchema, ColumnName: "name", Name: "name"},
			expected: true,
		},
		"no-serialized: PK included": {
			field:    Field{Schema: noSerSchema, ColumnName: "id", Options: PostgresOptions{PrimaryKey: true}},
			expected: true,
		},
		"normal: serialized column included": {
			field:    Field{Schema: normalSchema, ColumnName: "serialized"},
			expected: true,
		},
		"normal: regular field excluded without PK/search/ref": {
			field:    Field{Schema: normalSchema, ColumnName: "name", Name: "name"},
			expected: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.field.Include())
		})
	}
}

func TestWalkWithNoSerialized(t *testing.T) {
	mt := reflect.TypeOf(&TestStorageType{})
	require.NotNil(t, mt)

	t.Run("default walk includes serialized", func(t *testing.T) {
		schema := Walk(mt, "test_table")
		var hasSerialized bool
		for _, f := range schema.Fields {
			if f.ColumnName == "serialized" {
				hasSerialized = true
			}
		}
		assert.True(t, hasSerialized, "default walk should include serialized field")
		assert.False(t, schema.NoSerialized)
	})

	t.Run("WithNoSerialized excludes serialized from DBColumnFields", func(t *testing.T) {
		schema := Walk(mt, "test_table", WithNoSerialized())
		assert.True(t, schema.NoSerialized)
		for _, f := range schema.DBColumnFields() {
			assert.NotEqual(t, "serialized", f.ColumnName,
				"no-serialized schema should not have serialized in DBColumnFields")
		}
	})
}
