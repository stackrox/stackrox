package schema

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var manuallyMaintainedTables = map[string]bool{
	"network_flows_v2":              true,
	"background_migration_versions": true,
}

func TestGeneratedSchemasMatchWalker(t *testing.T) {
	for tableName, rt := range registeredTables {
		t.Run(tableName, func(t *testing.T) {
			if manuallyMaintainedTables[tableName] {
				t.Skip("manually maintained schema")
				return
			}
			if rt.Schema.Type == "" {
				t.Skip("schema has no Type field")
				return
			}

			// Extract the proto type name (strip leading "*" if present)
			typeName := strings.TrimPrefix(rt.Schema.Type, "*")

			// Get the reflect.Type for this proto message
			protoType := protoutils.MessageType(typeName)
			if protoType == nil {
				t.Skipf("cannot resolve proto type %q", typeName)
				return
			}

			var opts []walker.WalkOption
			if rt.Schema.NoSerialized {
				opts = append(opts, walker.WithNoSerialized())
			}
			groundTruth := walker.Walk(protoType, tableName, opts...)

			compareSchemas(t, tableName, rt.Schema, groundTruth)
		})
	}
}

// TestGeneratedSearchOptionsMatch validates that registered OptionsMaps match
// the output of search.Walk. This ensures static search options will match reflection-based ones.
func TestGeneratedSearchOptionsMatch(t *testing.T) {
	for tableName, rt := range registeredTables {
		t.Run(tableName, func(t *testing.T) {
			if manuallyMaintainedTables[tableName] {
				t.Skip("manually maintained schema")
				return
			}
			if rt.Schema.OptionsMap == nil {
				t.Skip("schema has no OptionsMap")
				return
			}

			// Skip schemas without a valid proto Type
			if rt.Schema.Type == "" {
				t.Skip("schema has no Type field")
				return
			}

			// Extract the proto type name (strip leading "*" if present)
			typeName := strings.TrimPrefix(rt.Schema.Type, "*")

			// Get the reflect.Type for this proto message
			protoType := protoutils.MessageType(typeName)
			if protoType == nil {
				t.Skipf("cannot resolve proto type %q", typeName)
				return
			}

			// Create a nil pointer instance for search.Walk
			protoInstance := reflect.New(protoType.Elem()).Interface()

			// Get the prefix (message name lowercased, e.g., "storage.Alert" -> "alert")
			parts := strings.Split(typeName, ".")
			messageName := parts[len(parts)-1]
			prefix := strings.ToLower(messageName)

			// Generate ground truth OptionsMap using search.Walk
			category := rt.Schema.OptionsMap.PrimaryCategory()
			groundTruth := search.Walk(category, prefix, protoInstance)

			// Compare the OptionsMaps
			compareOptionsMaps(t, tableName, rt.Schema.OptionsMap, groundTruth)
		})
	}
}

// TestGeneratedEnumRegistryMatch validates that static AddValues calls in generated
// schema files produce the same enum registry entries as search.Walk would.
func TestGeneratedEnumRegistryMatch(t *testing.T) {
	for tableName, rt := range registeredTables {
		t.Run(tableName, func(t *testing.T) {
			if manuallyMaintainedTables[tableName] {
				t.Skip("manually maintained schema")
				return
			}
			if rt.Schema.OptionsMap == nil {
				t.Skip("schema has no OptionsMap")
				return
			}
			if rt.Schema.Type == "" {
				t.Skip("schema has no Type field")
				return
			}

			typeName := strings.TrimPrefix(rt.Schema.Type, "*")
			protoType := protoutils.MessageType(typeName)
			if protoType == nil {
				t.Skipf("cannot resolve proto type %q", typeName)
				return
			}

			before := enumregistry.Snapshot()

			protoInstance := reflect.New(protoType.Elem()).Interface()
			parts := strings.Split(typeName, ".")
			prefix := strings.ToLower(parts[len(parts)-1])
			category := rt.Schema.OptionsMap.PrimaryCategory()
			search.Walk(category, prefix, protoInstance)

			after := enumregistry.Snapshot()

			for path, afterValues := range after {
				beforeValues, existed := before[path]
				if !existed {
					t.Errorf("enum path %q was added by search.Walk but not by static AddValues", path)
					continue
				}
				for name, num := range afterValues {
					beforeNum, ok := beforeValues[name]
					if !ok {
						t.Errorf("enum path %q: value %q=%d added by search.Walk but missing from static AddValues", path, name, num)
						continue
					}
					assert.Equal(t, num, beforeNum, "enum path %q: value %q number mismatch", path, name)
				}
			}
		})
	}
}

// compareSchemas compares two walker.Schema objects field by field.
// It does not compare Reference pointers since those are runtime cross-references.
func compareSchemas(t *testing.T, path string, registered, groundTruth *walker.Schema) {
	// Basic fields
	assert.Equal(t, groundTruth.Table, registered.Table, "%s: Table mismatch", path)
	assert.Equal(t, groundTruth.Type, registered.Type, "%s: Type mismatch", path)
	assert.Equal(t, groundTruth.TypeName, registered.TypeName, "%s: TypeName mismatch", path)
	assert.Equal(t, groundTruth.ObjectGetter, registered.ObjectGetter, "%s: ObjectGetter mismatch", path)
	assert.Equal(t, groundTruth.NoSerialized, registered.NoSerialized, "%s: NoSerialized mismatch", path)

	// Compare Fields
	require.Equal(t, len(groundTruth.Fields), len(registered.Fields), "%s: Field count mismatch", path)
	for i := range groundTruth.Fields {
		compareFields(t, path, &registered.Fields[i], &groundTruth.Fields[i])
	}

	// Compare Children recursively
	require.Equal(t, len(groundTruth.Children), len(registered.Children), "%s: Children count mismatch", path)
	for i := range groundTruth.Children {
		childPath := path + "." + groundTruth.Children[i].Table
		compareSchemas(t, childPath, registered.Children[i], groundTruth.Children[i])
	}

	// Compare SubMessages map
	if groundTruth.SubMessages != nil || registered.SubMessages != nil {
		assert.Equal(t, groundTruth.SubMessages, registered.SubMessages, "%s: SubMessages mismatch", path)
	}
}

// compareFields compares two walker.Field objects.
func compareFields(t *testing.T, schemaPath string, registered, groundTruth *walker.Field) {
	fieldPath := schemaPath + "." + groundTruth.Name

	assert.Equal(t, groundTruth.Name, registered.Name, "%s: Name mismatch", fieldPath)
	assert.Equal(t, groundTruth.ColumnName, registered.ColumnName, "%s: ColumnName mismatch", fieldPath)
	assert.Equal(t, groundTruth.ProtoBufName, registered.ProtoBufName, "%s: ProtoBufName mismatch", fieldPath)
	assert.Equal(t, groundTruth.Type, registered.Type, "%s: Type mismatch", fieldPath)
	assert.Equal(t, groundTruth.DataType, registered.DataType, "%s: DataType mismatch", fieldPath)
	assert.Equal(t, groundTruth.SQLType, registered.SQLType, "%s: SQLType mismatch", fieldPath)
	assert.Equal(t, groundTruth.ModelType, registered.ModelType, "%s: ModelType mismatch", fieldPath)

	// Compare ObjectGetter (value only, not variable flag since both should match)
	assert.Equal(t, groundTruth.ObjectGetter, registered.ObjectGetter, "%s: ObjectGetter mismatch", fieldPath)

	// Compare Search settings
	assert.Equal(t, groundTruth.Search.FieldName, registered.Search.FieldName, "%s: Search.FieldName mismatch", fieldPath)
	assert.Equal(t, groundTruth.Search.Enabled, registered.Search.Enabled, "%s: Search.Enabled mismatch", fieldPath)
	assert.Equal(t, groundTruth.Search.Ignored, registered.Search.Ignored, "%s: Search.Ignored mismatch", fieldPath)

	// Compare Options (not Reference pointers, but basic flags)
	assert.Equal(t, groundTruth.Options.ID, registered.Options.ID, "%s: Options.ID mismatch", fieldPath)
	assert.Equal(t, groundTruth.Options.PrimaryKey, registered.Options.PrimaryKey, "%s: Options.PrimaryKey mismatch", fieldPath)
	assert.Equal(t, groundTruth.Options.Unique, registered.Options.Unique, "%s: Options.Unique mismatch", fieldPath)
	assert.Equal(t, groundTruth.Options.Ignored, registered.Options.Ignored, "%s: Options.Ignored mismatch", fieldPath)
	assert.Equal(t, groundTruth.Options.ColumnType, registered.Options.ColumnType, "%s: Options.ColumnType mismatch", fieldPath)
	assert.Equal(t, groundTruth.Options.RepeatedStrategy, registered.Options.RepeatedStrategy, "%s: Options.RepeatedStrategy mismatch", fieldPath)
	assert.Equal(t, groundTruth.Derived, registered.Derived, "%s: Derived mismatch", fieldPath)

	// Compare DerivedSearchFields (order-insensitive)
	if len(groundTruth.DerivedSearchFields) > 0 || len(registered.DerivedSearchFields) > 0 {
		assert.ElementsMatch(t, groundTruth.DerivedSearchFields, registered.DerivedSearchFields, "%s: DerivedSearchFields mismatch", fieldPath)
	}
}

// compareOptionsMaps compares two search.OptionsMap objects.
func compareOptionsMaps(t *testing.T, tableName string, registered, groundTruth search.OptionsMap) {
	assert.Equal(t, groundTruth.PrimaryCategory(), registered.PrimaryCategory(), "%s: PrimaryCategory mismatch", tableName)

	registeredFields := registered.Original()
	groundTruthFields := groundTruth.Original()

	// Compare field counts
	require.Equal(t, len(groundTruthFields), len(registeredFields), "%s: OptionsMap field count mismatch", tableName)

	// Compare each field
	for label, gtField := range groundTruthFields {
		regField, ok := registeredFields[label]
		require.True(t, ok, "%s: OptionsMap missing field %q", tableName, label)

		assert.Equal(t, gtField.FieldPath, regField.FieldPath, "%s: OptionsMap field %q: FieldPath mismatch", tableName, label)
		assert.Equal(t, gtField.Type, regField.Type, "%s: OptionsMap field %q: Type mismatch", tableName, label)
		assert.Equal(t, gtField.Store, regField.Store, "%s: OptionsMap field %q: Store mismatch", tableName, label)
		assert.Equal(t, gtField.Hidden, regField.Hidden, "%s: OptionsMap field %q: Hidden mismatch", tableName, label)
		assert.Equal(t, gtField.Category, regField.Category, "%s: OptionsMap field %q: Category mismatch", tableName, label)
		assert.Equal(t, gtField.Analyzer, regField.Analyzer, "%s: OptionsMap field %q: Analyzer mismatch", tableName, label)
	}
}
