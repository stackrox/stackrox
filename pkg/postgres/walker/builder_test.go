package walker

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestMakeObjectGetter(t *testing.T) {
	cases := map[string]struct {
		value    string
		variable bool
	}{
		"local variable": {
			value:    "parentId",
			variable: true,
		},
		"field getter": {
			value:    "GetId()",
			variable: false,
		},
		"nested getter": {
			value:    "GetSignal().GetName()",
			variable: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			og := MakeObjectGetter(tc.value, tc.variable)
			assert.Equal(t, tc.value, og.GetValue())
			assert.Equal(t, tc.variable, og.GetVariable())
		})
	}
}

func TestSetReference(t *testing.T) {
	cases := map[string]struct {
		typeName       string
		protoBufField  string
		noConstraint   bool
		restrictDelete bool
		directional    bool
		nullable       bool
	}{
		"basic reference": {
			typeName:      "storage.Alert",
			protoBufField: "id",
		},
		"with no constraint": {
			typeName:      "storage.Alert",
			protoBufField: "id",
			noConstraint:  true,
		},
		"with restrict delete": {
			typeName:       "storage.Alert",
			protoBufField:  "id",
			restrictDelete: true,
		},
		"directional reference": {
			typeName:      "storage.Alert",
			protoBufField: "id",
			directional:   true,
		},
		"nullable reference": {
			typeName:      "storage.Alert",
			protoBufField: "id",
			nullable:      true,
		},
		"all flags": {
			typeName:       "storage.Alert",
			protoBufField:  "id",
			noConstraint:   true,
			restrictDelete: true,
			directional:    true,
			nullable:       true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := Field{}
			f.SetReference(tc.typeName, tc.protoBufField, tc.noConstraint, tc.restrictDelete, tc.directional, tc.nullable)

			assert.True(t, f.HasReference())
			assert.Equal(t, tc.typeName, f.RefTypeName())
			assert.Equal(t, tc.protoBufField, f.RefProtoBufField())
			assert.Equal(t, tc.noConstraint, f.RefNoConstraint())
			assert.Equal(t, tc.restrictDelete, f.RefRestrictDelete())
			assert.Equal(t, tc.directional, f.RefDirectional())
			assert.Equal(t, tc.nullable, f.RefNullable())
		})
	}
}

func TestSetParentReference(t *testing.T) {
	parentSchema := &Schema{Table: "alerts"}
	f := Field{}
	f.SetParentReference(parentSchema, "alert_id")

	assert.True(t, f.HasReference())
	assert.Equal(t, parentSchema, f.Options.Reference.OtherSchema)
	assert.Equal(t, "alert_id", f.Options.Reference.ColumnName)
}

func TestHasReference(t *testing.T) {
	cases := map[string]struct {
		field    Field
		expected bool
	}{
		"no reference": {
			field:    Field{},
			expected: false,
		},
		"with reference": {
			field: Field{
				Options: PostgresOptions{
					Reference: &foreignKeyRef{
						TypeName:      "storage.Alert",
						ProtoBufField: "id",
					},
				},
			},
			expected: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.field.HasReference())
		})
	}
}

func TestReferenceAccessors(t *testing.T) {
	t.Run("nil reference returns zero values", func(t *testing.T) {
		f := Field{}
		assert.False(t, f.HasReference())
		assert.Equal(t, "", f.RefTypeName())
		assert.Equal(t, "", f.RefProtoBufField())
		assert.False(t, f.RefNoConstraint())
		assert.False(t, f.RefRestrictDelete())
		assert.False(t, f.RefDirectional())
		assert.False(t, f.RefNullable())
	})

	t.Run("reference with all fields set", func(t *testing.T) {
		f := Field{
			Options: PostgresOptions{
				Reference: &foreignKeyRef{
					TypeName:       "storage.Alert",
					ProtoBufField:  "id",
					NoConstraint:   true,
					RestrictDelete: true,
					Directional:    true,
					Nullable:       true,
				},
			},
		}
		assert.True(t, f.HasReference())
		assert.Equal(t, "storage.Alert", f.RefTypeName())
		assert.Equal(t, "id", f.RefProtoBufField())
		assert.True(t, f.RefNoConstraint())
		assert.True(t, f.RefRestrictDelete())
		assert.True(t, f.RefDirectional())
		assert.True(t, f.RefNullable())
	})
}

func TestRoundTrip(t *testing.T) {
	walkedSchema := Walk(reflect.TypeFor[*storage.TestSingleKeyStruct](), "test_single_key_structs")

	manualSchema := &Schema{
		Table:    "test_single_key_structs",
		Type:     walkedSchema.Type,
		TypeName: walkedSchema.TypeName,
		Fields:   []Field{},
	}

	for _, wf := range walkedSchema.Fields {
		mf := Field{
			Schema:       manualSchema,
			Name:         wf.Name,
			ProtoBufName: wf.ProtoBufName,
			ObjectGetter: MakeObjectGetter(wf.ObjectGetter.value, wf.ObjectGetter.variable),
			ColumnName:   wf.ColumnName,
			Type:         wf.Type,
			DataType:     wf.DataType,
			SQLType:      wf.SQLType,
			ModelType:    wf.ModelType,
		}

		if wf.Options.Reference != nil {
			if wf.Options.Reference.OtherSchema != nil && wf.Options.Reference.OtherSchema == wf.Schema.Parent {
				mf.SetParentReference(wf.Options.Reference.OtherSchema, wf.Options.Reference.ColumnName)
			} else {
				mf.SetReference(
					wf.Options.Reference.TypeName,
					wf.Options.Reference.ProtoBufField,
					wf.Options.Reference.NoConstraint,
					wf.Options.Reference.RestrictDelete,
					wf.Options.Reference.Directional,
					wf.Options.Reference.Nullable,
				)
			}
		}

		manualSchema.Fields = append(manualSchema.Fields, mf)
	}

	assert.Equal(t, len(walkedSchema.Fields), len(manualSchema.Fields), "field count should match")

	for i := range walkedSchema.Fields {
		wf := walkedSchema.Fields[i]
		mf := manualSchema.Fields[i]

		assert.Equal(t, wf.Name, mf.Name, "field %d: Name mismatch", i)
		assert.Equal(t, wf.ColumnName, mf.ColumnName, "field %d: ColumnName mismatch", i)
		assert.Equal(t, wf.ObjectGetter.value, mf.ObjectGetter.value, "field %d: ObjectGetter.value mismatch", i)
		assert.Equal(t, wf.ObjectGetter.variable, mf.ObjectGetter.variable, "field %d: ObjectGetter.variable mismatch", i)

		if wf.HasReference() {
			assert.True(t, mf.HasReference(), "field %d: should have reference", i)
			assert.Equal(t, wf.RefTypeName(), mf.RefTypeName(), "field %d: RefTypeName mismatch", i)
			assert.Equal(t, wf.RefProtoBufField(), mf.RefProtoBufField(), "field %d: RefProtoBufField mismatch", i)
			assert.Equal(t, wf.RefNoConstraint(), mf.RefNoConstraint(), "field %d: RefNoConstraint mismatch", i)
			assert.Equal(t, wf.RefRestrictDelete(), mf.RefRestrictDelete(), "field %d: RefRestrictDelete mismatch", i)
			assert.Equal(t, wf.RefDirectional(), mf.RefDirectional(), "field %d: RefDirectional mismatch", i)
			assert.Equal(t, wf.RefNullable(), mf.RefNullable(), "field %d: RefNullable mismatch", i)
		} else {
			assert.False(t, mf.HasReference(), "field %d: should not have reference", i)
		}
	}
}
