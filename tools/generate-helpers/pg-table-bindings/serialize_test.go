package main

import (
	"go/parser"
	"go/token"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseGoCode(t *testing.T, code string) {
	t.Helper()
	src := "package p\nfunc f() {\n" + code + "\n}"
	_, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	require.NoError(t, err, "Generated code should be valid Go syntax")
}

func TestSerializeSchemaSimple(t *testing.T) {
	schema := &walker.Schema{
		Table:    "alerts",
		Type:     "*storage.Alert",
		TypeName: "Alert",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
				ID:         true,
			},
			Search: walker.SearchField{
				FieldName: "Alert ID",
				Enabled:   true,
			},
		},
		{
			Schema:       schema,
			Name:         "State",
			ProtoBufName: "state",
			ColumnName:   "state",
			Type:         "int32",
			DataType:     postgres.Integer,
			SQLType:      "integer",
			ModelType:    "int32",
			ObjectGetter: walker.MakeObjectGetter("GetState()", false),
			Options:      walker.PostgresOptions{},
			Search: walker.SearchField{
				FieldName: "State",
				Enabled:   true,
			},
		},
	}

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, `Table: "alerts"`)
	assert.Contains(t, output, `Type: "*storage.Alert"`)
	assert.Contains(t, output, `TypeName: "Alert"`)
	assert.Contains(t, output, "schema.Fields = []walker.Field{")
	assert.Contains(t, output, `Name: "Id"`)
	assert.Contains(t, output, `ColumnName: "id"`)
	assert.Contains(t, output, "walker.MakeObjectGetter")
	assert.Contains(t, output, `DataType: postgres.String`)
	assert.Contains(t, output, "PrimaryKey: true")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithChildren(t *testing.T) {
	parent := &walker.Schema{
		Table:    "deployments",
		Type:     "*storage.Deployment",
		TypeName: "Deployment",
	}
	parent.Fields = []walker.Field{
		{
			Schema:       parent,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
				ID:         true,
			},
		},
	}

	child := &walker.Schema{
		Parent:       parent,
		Table:        "deployments_labels",
		Type:         "*storage.Deployment_Label",
		TypeName:     "Label",
		ObjectGetter: "GetLabels()",
	}
	child.Fields = []walker.Field{
		{
			Schema:       child,
			Name:         "Key",
			ProtoBufName: "key",
			ColumnName:   "key",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "varchar",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetKey()", false),
		},
	}

	parent.Children = []*walker.Schema{child}

	output := SerializeSchema(parent, "schema")

	assert.Contains(t, output, `Table: "deployments"`)
	assert.Contains(t, output, "child0 := &walker.Schema{")
	assert.Contains(t, output, "Parent: schema")
	assert.Contains(t, output, `Table: "deployments_labels"`)
	assert.Contains(t, output, `ObjectGetter: "GetLabels()"`)
	assert.Contains(t, output, "child0.Fields = []walker.Field{")
	assert.Contains(t, output, "schema.Children = []*walker.Schema{child0}")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithReference(t *testing.T) {
	referencedSchema := &walker.Schema{
		Table:    "clusters",
		Type:     "*storage.Cluster",
		TypeName: "Cluster",
	}
	referencedSchema.Fields = []walker.Field{
		{
			Schema:       referencedSchema,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
				ID:         true,
			},
		},
	}

	schema := &walker.Schema{
		Table:    "deployments",
		Type:     "*storage.Deployment",
		TypeName: "Deployment",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "ClusterId",
			ProtoBufName: "cluster_id",
			ColumnName:   "cluster_id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetClusterId()", false),
		},
	}

	schema.Fields[0].SetReference("storage.Cluster", "id", false, false, false, false)
	schema.Fields[0].Options.Reference.OtherSchema = referencedSchema
	schema.Fields[0].Options.Reference.ColumnName = "id"

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, "schema.Fields[0].SetReference")
	assert.Contains(t, output, `"storage.Cluster"`)
	assert.Contains(t, output, `"id"`)
	assert.Contains(t, output, "false, false, false, false")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithParentReference(t *testing.T) {
	parent := &walker.Schema{
		Table:    "deployments",
		Type:     "*storage.Deployment",
		TypeName: "Deployment",
	}
	parent.Fields = []walker.Field{
		{
			Schema:       parent,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
				ID:         true,
			},
		},
	}

	child := &walker.Schema{
		Parent:       parent,
		Table:        "deployments_labels",
		Type:         "*storage.Deployment_Label",
		TypeName:     "Label",
		ObjectGetter: "GetLabels()",
	}
	child.Fields = []walker.Field{
		{
			Schema:       child,
			Name:         "DeploymentId",
			ProtoBufName: "deployment_id",
			ColumnName:   "deployment_id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("deploymentId", true),
		},
	}

	child.Fields[0].SetParentReference(parent, "id")

	parent.Children = []*walker.Schema{child}

	output := SerializeSchema(parent, "schema")

	assert.Contains(t, output, "child0.Fields[0].SetParentReference(schema")
	assert.Contains(t, output, `"id"`)

	parseGoCode(t, output)
}

func TestSerializeSearchFields(t *testing.T) {
	fields := map[search.FieldLabel]*search.Field{
		"Alert ID": {
			FieldPath: "alert.id",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Store:     true,
			Category:  v1.SearchCategory_ALERTS,
		},
		"Cluster": {
			FieldPath: "alert.clustername",
			Type:      v1.SearchDataType_SEARCH_STRING,
			Hidden:    true,
			Category:  v1.SearchCategory_ALERTS,
			Analyzer:  "keyword",
		},
	}

	optionsMap := search.OptionsMapFromMap(v1.SearchCategory_ALERTS, fields)
	output := SerializeSearchFields(optionsMap, "v1.SearchCategory_ALERTS")

	assert.Contains(t, output, "map[search.FieldLabel]*search.Field{")
	assert.Contains(t, output, `"Alert ID"`)
	assert.Contains(t, output, `FieldPath: "alert.id"`)
	assert.Contains(t, output, "Type: v1.SearchDataType_SEARCH_STRING")
	assert.Contains(t, output, "Category: v1.SearchCategory_ALERTS")
	assert.Contains(t, output, "Store: true")
	assert.Contains(t, output, "Hidden: true")
	assert.Contains(t, output, `Analyzer: "keyword"`)

	assert.NotContains(t, output, "Store: false")
	assert.NotContains(t, output, "Hidden: false")

	parseGoCode(t, output)
}

func TestSerializeEnumEntries(t *testing.T) {
	before := map[string]map[string]int32{
		"alert.state": {
			"ACTIVE":   1,
			"RESOLVED": 2,
		},
	}

	after := map[string]map[string]int32{
		"alert.state": {
			"ACTIVE":   1,
			"RESOLVED": 2,
		},
		"policy.severity": {
			"LOW":      1,
			"MEDIUM":   2,
			"HIGH":     3,
			"CRITICAL": 4,
		},
	}

	output := SerializeEnumEntries(before, after)

	assert.Contains(t, output, "enumregistry.AddValues")
	assert.Contains(t, output, `"policy.severity"`)
	assert.Contains(t, output, `"CRITICAL": 4`)
	assert.Contains(t, output, `"HIGH": 3`)
	assert.Contains(t, output, `"LOW": 1`)
	assert.Contains(t, output, `"MEDIUM": 2`)

	assert.NotContains(t, output, "alert.state")

	parseGoCode(t, output)
}

func TestSerializeEnumEntriesEmpty(t *testing.T) {
	before := map[string]map[string]int32{
		"alert.state": {
			"ACTIVE": 1,
		},
	}

	after := map[string]map[string]int32{
		"alert.state": {
			"ACTIVE": 1,
		},
	}

	output := SerializeEnumEntries(before, after)

	assert.Empty(t, output)
}

func TestSerializeSchemaWithDerivedSearchFields(t *testing.T) {
	schema := &walker.Schema{
		Table:    "alerts",
		Type:     "*storage.Alert",
		TypeName: "Alert",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
			},
			Search: walker.SearchField{
				FieldName: "Alert ID",
				Enabled:   true,
			},
			DerivedSearchFields: []walker.DerivedSearchField{
				{
					DerivedFrom:     "Alert Count",
					DerivationType:  search.CountDerivationType,
					DerivedDataType: postgres.Integer,
				},
			},
		},
	}

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, "DerivedSearchFields: []walker.DerivedSearchField{")
	assert.Contains(t, output, `DerivedFrom: "Alert Count"`)
	assert.Contains(t, output, "DerivationType: search.CountDerivationType")
	assert.Contains(t, output, "DerivedDataType: postgres.Integer")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithNoSerialized(t *testing.T) {
	schema := &walker.Schema{
		Table:        "test_noserialized",
		Type:         "*storage.TestNoSerialized",
		TypeName:     "TestNoSerialized",
		NoSerialized: true,
		SubMessages: map[string]string{
			"Metadata": "storage.TestNoSerialized_Metadata",
		},
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
			},
		},
	}

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, "NoSerialized: true")
	assert.Contains(t, output, "SubMessages: map[string]string{")
	assert.Contains(t, output, `"Metadata": "storage.TestNoSerialized_Metadata"`)

	parseGoCode(t, output)
}

func TestSerializeSchemaWithGrandchildren(t *testing.T) {
	root := &walker.Schema{
		Table:    "root",
		Type:     "*storage.Root",
		TypeName: "Root",
	}
	root.Fields = []walker.Field{
		{
			Schema:       root,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
			},
		},
	}

	child := &walker.Schema{
		Parent:       root,
		Table:        "root_child",
		Type:         "*storage.Root_Child",
		TypeName:     "Child",
		ObjectGetter: "GetChildren()",
	}
	child.Fields = []walker.Field{
		{
			Schema:       child,
			Name:         "Name",
			ProtoBufName: "name",
			ColumnName:   "name",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "varchar",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetName()", false),
		},
	}

	grandchild := &walker.Schema{
		Parent:       child,
		Table:        "root_child_grandchild",
		Type:         "*storage.Root_Child_Grandchild",
		TypeName:     "Grandchild",
		ObjectGetter: "GetGrandchildren()",
	}
	grandchild.Fields = []walker.Field{
		{
			Schema:       grandchild,
			Name:         "Value",
			ProtoBufName: "value",
			ColumnName:   "value",
			Type:         "int32",
			DataType:     postgres.Integer,
			SQLType:      "integer",
			ModelType:    "int32",
			ObjectGetter: walker.MakeObjectGetter("GetValue()", false),
		},
	}

	child.Children = []*walker.Schema{grandchild}
	root.Children = []*walker.Schema{child}

	output := SerializeSchema(root, "schema")

	assert.Contains(t, output, "child0 := &walker.Schema{")
	assert.Contains(t, output, "child0_child0 := &walker.Schema{")
	assert.Contains(t, output, "Parent: child0")
	assert.Contains(t, output, "child0.Children = []*walker.Schema{child0_child0}")
	assert.Contains(t, output, "schema.Children = []*walker.Schema{child0}")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithComplexReferences(t *testing.T) {
	schema := &walker.Schema{
		Table:    "deployments",
		Type:     "*storage.Deployment",
		TypeName: "Deployment",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "ClusterId",
			ProtoBufName: "cluster_id",
			ColumnName:   "cluster_id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetClusterId()", false),
		},
	}

	schema.Fields[0].SetReference("storage.Cluster", "id", true, true, true, true)

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, "schema.Fields[0].SetReference")
	assert.Contains(t, output, "true, true, true, true")

	parseGoCode(t, output)
}

func TestSerializeSchemaWithColumnType(t *testing.T) {
	schema := &walker.Schema{
		Table:    "test",
		Type:     "*storage.Test",
		TypeName: "Test",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "CustomField",
			ProtoBufName: "custom_field",
			ColumnName:   "custom_field",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "text",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetCustomField()", false),
			Options: walker.PostgresOptions{
				ColumnType: "text",
			},
		},
	}

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, `ColumnType: "text"`)

	parseGoCode(t, output)
}

func TestSerializeSchemaWithUniqueConstraint(t *testing.T) {
	schema := &walker.Schema{
		Table:    "test",
		Type:     "*storage.Test",
		TypeName: "Test",
	}
	schema.Fields = []walker.Field{
		{
			Schema:       schema,
			Name:         "UniqueField",
			ProtoBufName: "unique_field",
			ColumnName:   "unique_field",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "varchar",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetUniqueField()", false),
			Options: walker.PostgresOptions{
				Unique: true,
			},
		},
	}

	output := SerializeSchema(schema, "schema")

	assert.Contains(t, output, "Unique: true")

	parseGoCode(t, output)
}

func TestSerializeSchemaMultipleChildren(t *testing.T) {
	parent := &walker.Schema{
		Table:    "parent",
		Type:     "*storage.Parent",
		TypeName: "Parent",
	}
	parent.Fields = []walker.Field{
		{
			Schema:       parent,
			Name:         "Id",
			ProtoBufName: "id",
			ColumnName:   "id",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "uuid",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetId()", false),
			Options: walker.PostgresOptions{
				PrimaryKey: true,
			},
		},
	}

	child1 := &walker.Schema{
		Parent:       parent,
		Table:        "parent_child1",
		Type:         "*storage.Parent_Child1",
		TypeName:     "Child1",
		ObjectGetter: "GetChild1()",
	}
	child1.Fields = []walker.Field{
		{
			Schema:       child1,
			Name:         "Name",
			ProtoBufName: "name",
			ColumnName:   "name",
			Type:         "string",
			DataType:     postgres.String,
			SQLType:      "varchar",
			ModelType:    "string",
			ObjectGetter: walker.MakeObjectGetter("GetName()", false),
		},
	}

	child2 := &walker.Schema{
		Parent:       parent,
		Table:        "parent_child2",
		Type:         "*storage.Parent_Child2",
		TypeName:     "Child2",
		ObjectGetter: "GetChild2()",
	}
	child2.Fields = []walker.Field{
		{
			Schema:       child2,
			Name:         "Value",
			ProtoBufName: "value",
			ColumnName:   "value",
			Type:         "int32",
			DataType:     postgres.Integer,
			SQLType:      "integer",
			ModelType:    "int32",
			ObjectGetter: walker.MakeObjectGetter("GetValue()", false),
		},
	}

	parent.Children = []*walker.Schema{child1, child2}

	output := SerializeSchema(parent, "schema")

	assert.Contains(t, output, "child0 := &walker.Schema{")
	assert.Contains(t, output, "child1 := &walker.Schema{")
	assert.Contains(t, output, "schema.Children = []*walker.Schema{child0, child1}")

	parseGoCode(t, output)
}
