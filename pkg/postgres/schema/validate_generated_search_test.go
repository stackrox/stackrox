package schema

import (
	"reflect"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema/internal"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateGeneratedSearchFields compares the generated search fields
// with the original walker.Walk output to ensure no functionality is lost
func TestValidateGeneratedSearchFields(t *testing.T) {
	testCases := []struct {
		name             string
		storageType      reflect.Type
		tableName        string
		generatedFields  map[search.FieldLabel]*search.Field
		generatedSchema  *walker.Schema
		searchCategory   v1.SearchCategory
	}{
		{
			name:             "alerts",
			storageType:      reflect.TypeOf((*storage.Alert)(nil)),
			tableName:        "alerts",
			generatedFields:  internal.AlertSearchFields,
			generatedSchema:  internal.AlertSchema,
			searchCategory:   v1.SearchCategory_ALERTS,
		},
		{
			name:             "policies",
			storageType:      reflect.TypeOf((*storage.Policy)(nil)),
			tableName:        "policies",
			generatedFields:  internal.PolicySearchFields,
			generatedSchema:  internal.PolicySchema,
			searchCategory:   v1.SearchCategory_POLICIES,
		},
		{
			name:             "deployments",
			storageType:      reflect.TypeOf((*storage.Deployment)(nil)),
			tableName:        "deployments",
			generatedFields:  internal.DeploymentSearchFields,
			generatedSchema:  internal.DeploymentSchema,
			searchCategory:   v1.SearchCategory_DEPLOYMENTS,
		},
		{
			name:             "nodes",
			storageType:      reflect.TypeOf((*storage.Node)(nil)),
			tableName:        "nodes",
			generatedFields:  internal.NodeSearchFields,
			generatedSchema:  internal.NodeSchema,
			searchCategory:   v1.SearchCategory_NODES,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate original schema using walker.Walk
			originalSchema := walker.Walk(tc.storageType, tc.tableName)

			// Create OptionsMap from storage type using search.Walk
			originalSearchOptionsMap := search.Walk(tc.searchCategory, "", reflect.Zero(tc.storageType).Interface())
			originalSearchFields := originalSearchOptionsMap.Original()

			// Compare the number of search fields
			assert.Equal(t, len(originalSearchFields), len(tc.generatedFields),
				"Number of search fields should match between original and generated")

			// Compare each search field
			for fieldLabel, originalField := range originalSearchFields {
				generatedField, exists := tc.generatedFields[fieldLabel]
				require.True(t, exists, "Generated fields missing field: %s", fieldLabel)

				// Compare field properties
				assert.Equal(t, originalField.FieldPath, generatedField.FieldPath,
					"FieldPath mismatch for field: %s", fieldLabel)
				assert.Equal(t, originalField.Store, generatedField.Store,
					"Store mismatch for field: %s", fieldLabel)
				assert.Equal(t, originalField.Hidden, generatedField.Hidden,
					"Hidden mismatch for field: %s", fieldLabel)
				assert.Equal(t, originalField.Category, generatedField.Category,
					"Category mismatch for field: %s", fieldLabel)

				// Compare analyzer if present
				if originalField.Analyzer != "" {
					assert.Equal(t, originalField.Analyzer, generatedField.Analyzer,
						"Analyzer mismatch for field: %s", fieldLabel)
				}
			}

			// Check for extra fields in generated that weren't in original
			for fieldLabel := range tc.generatedFields {
				_, exists := originalSearchFields[fieldLabel]
				assert.True(t, exists, "Generated fields has extra field not in original: %s", fieldLabel)
			}

			// Compare schema structure
			compareSchemaStructure(t, originalSchema, tc.generatedSchema)
		})
	}
}

// compareSchemaStructure compares the structure of two schemas
func compareSchemaStructure(t *testing.T, original, generated *walker.Schema) {
	assert.Equal(t, original.Table, generated.Table, "Table name should match")
	assert.Equal(t, original.Type, generated.Type, "Type should match")
	assert.Equal(t, original.TypeName, generated.TypeName, "TypeName should match")

	// Compare fields count
	assert.Equal(t, len(original.Fields), len(generated.Fields),
		"Number of fields should match")

	// Create a map for easier field comparison
	originalFieldsMap := make(map[string]walker.Field)
	for _, field := range original.Fields {
		originalFieldsMap[field.ColumnName] = field
	}

	generatedFieldsMap := make(map[string]walker.Field)
	for _, field := range generated.Fields {
		generatedFieldsMap[field.ColumnName] = field
	}

	// Compare each field
	for columnName, originalField := range originalFieldsMap {
		generatedField, exists := generatedFieldsMap[columnName]
		require.True(t, exists, "Generated schema missing field: %s", columnName)

		assert.Equal(t, originalField.Name, generatedField.Name,
			"Field name mismatch for column: %s", columnName)
		assert.Equal(t, originalField.Type, generatedField.Type,
			"Field type mismatch for column: %s", columnName)
		assert.Equal(t, originalField.SQLType, generatedField.SQLType,
			"SQL type mismatch for column: %s", columnName)
		assert.Equal(t, originalField.DataType, generatedField.DataType,
			"Data type mismatch for column: %s", columnName)
	}

	// Check for extra fields in generated that weren't in original
	for columnName := range generatedFieldsMap {
		_, exists := originalFieldsMap[columnName]
		assert.True(t, exists, "Generated schema has extra field not in original: %s", columnName)
	}

	// Compare child schemas if they exist
	compareChildSchemas(t, original.Children, generated.Children)
}

// compareChildSchemas compares child schemas recursively
func compareChildSchemas(t *testing.T, originalChildren, generatedChildren []*walker.Schema) {
	assert.Equal(t, len(originalChildren), len(generatedChildren),
		"Number of child schemas should match")

	// Create maps for easier comparison
	originalChildrenMap := make(map[string]*walker.Schema)
	for _, child := range originalChildren {
		originalChildrenMap[child.Table] = child
	}

	generatedChildrenMap := make(map[string]*walker.Schema)
	for _, child := range generatedChildren {
		generatedChildrenMap[child.Table] = child
	}

	// Compare each child schema
	for tableName, originalChild := range originalChildrenMap {
		generatedChild, exists := generatedChildrenMap[tableName]
		require.True(t, exists, "Generated schema missing child table: %s", tableName)

		// Recursively compare child schema structure
		compareSchemaStructure(t, originalChild, generatedChild)
	}

	// Check for extra child schemas in generated
	for tableName := range generatedChildrenMap {
		_, exists := originalChildrenMap[tableName]
		assert.True(t, exists, "Generated schema has extra child table not in original: %s", tableName)
	}
}

// TestSearchFieldGeneration tests that the search fields are properly set up
func TestSearchFieldGeneration(t *testing.T) {
	testCases := []struct {
		name            string
		getSchemaFunc   func() *walker.Schema
		expectedFields  map[search.FieldLabel]*search.Field
		searchCategory  v1.SearchCategory
	}{
		{
			name:           "alerts",
			getSchemaFunc:  internal.GetAlertSchema,
			expectedFields: internal.AlertSearchFields,
			searchCategory: v1.SearchCategory_ALERTS,
		},
		{
			name:           "policies",
			getSchemaFunc:  internal.GetPolicySchema,
			expectedFields: internal.PolicySearchFields,
			searchCategory: v1.SearchCategory_POLICIES,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schema := tc.getSchemaFunc()

			// Verify that OptionsMap is set
			require.NotNil(t, schema.OptionsMap, "OptionsMap should be set")

			// Verify that all expected fields are present
			for fieldLabel, expectedField := range tc.expectedFields {
				actualField, exists := schema.OptionsMap.Get(string(fieldLabel))
				require.True(t, exists, "Missing field in OptionsMap: %s", fieldLabel)

				assert.Equal(t, expectedField.FieldPath, actualField.FieldPath,
					"FieldPath mismatch for field: %s", fieldLabel)
				assert.Equal(t, expectedField.Store, actualField.Store,
					"Store mismatch for field: %s", fieldLabel)
				assert.Equal(t, expectedField.Hidden, actualField.Hidden,
					"Hidden mismatch for field: %s", fieldLabel)
				assert.Equal(t, expectedField.Category, actualField.Category,
					"Category mismatch for field: %s", fieldLabel)
			}
		})
	}
}

// TestDeploymentChildSchemas specifically tests that deployment child schemas are properly generated
func TestDeploymentChildSchemas(t *testing.T) {
	// Generate original deployment schema with walker.Walk
	originalSchema := walker.Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")

	// Get our generated deployment schema
	generatedSchema := internal.GetDeploymentSchema()

	// The generated schema should have the same child schemas as the original
	require.Equal(t, len(originalSchema.Children), len(generatedSchema.Children),
		"Deployment schema should have the same number of child schemas")

	// Expected child table names for deployments
	expectedChildTables := []string{
		"deployments_containers",
		"deployments_ports",
	}

	// Verify each expected child table exists in both schemas
	for _, expectedTable := range expectedChildTables {
		t.Run("child_table_"+expectedTable, func(t *testing.T) {
			// Find in original schema
			var originalChild *walker.Schema
			for _, child := range originalSchema.Children {
				if child.Table == expectedTable {
					originalChild = child
					break
				}
			}
			require.NotNil(t, originalChild, "Original schema should have child table: %s", expectedTable)

			// Find in generated schema
			var generatedChild *walker.Schema
			for _, child := range generatedSchema.Children {
				if child.Table == expectedTable {
					generatedChild = child
					break
				}
			}
			require.NotNil(t, generatedChild, "Generated schema should have child table: %s", expectedTable)

			// Compare the child schemas
			compareSchemaStructure(t, originalChild, generatedChild)
		})
	}
}

// BenchmarkOriginalWalkerWalk benchmarks the original walker.Walk approach
func BenchmarkOriginalWalkerWalk(b *testing.B) {
	storageType := reflect.TypeOf((*storage.Alert)(nil))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = walker.Walk(storageType, "alerts")
		_ = search.Walk(v1.SearchCategory_ALERTS, "", reflect.Zero(storageType).Interface())
	}
}

// BenchmarkGeneratedSchema benchmarks the generated schema approach
func BenchmarkGeneratedSchema(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = internal.GetAlertSchema()
	}
}
