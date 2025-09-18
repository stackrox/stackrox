package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

// SchemaGenerator generates PostgreSQL schema files
type SchemaGenerator struct {
	ProjectRoot string
	OutputDir   string
	Verbose     bool
	analyzer    *TypeAnalyzer
}

// SchemaData represents the data needed to generate a schema file
type SchemaData struct {
	PackageName     string
	TableName       string
	TypeName        string
	StoragePackage  string
	Fields          []SchemaField
	HasChildren     bool
	SearchCategory  string
	ScopingResource string
}

// SchemaField represents a field in the schema
type SchemaField struct {
	Name         string
	ColumnName   string
	Type         string
	SQLType      string
	DataType     string
	IsPointer    bool
	IsSlice      bool
	IsPrimaryKey bool
	IsIndex      bool
	IndexType    string
	IsReference  bool
	IsSearchable bool
	SearchField  string
}

// Generate generates all schema files
func (sg *SchemaGenerator) Generate() error {
	sg.analyzer = NewTypeAnalyzer()

	// Load required packages
	if err := sg.loadPackages(); err != nil {
		return fmt.Errorf("loading packages: %w", err)
	}

	// Find all existing schema files to determine what to generate
	schemaConfigs, err := sg.discoverSchemas()
	if err != nil {
		return fmt.Errorf("discovering schemas: %w", err)
	}

	// Generate each schema
	for _, config := range schemaConfigs {
		if err := sg.generateSchema(config); err != nil {
			return fmt.Errorf("generating schema for %s: %w", config.TypeName, err)
		}
		if sg.Verbose {
			log.Printf("Generated schema for %s", config.TypeName)
		}
	}

	return nil
}

// loadPackages loads the required Go packages for analysis
func (sg *SchemaGenerator) loadPackages() error {
	packages := []string{
		"github.com/stackrox/rox/generated/storage",
		"github.com/stackrox/rox/pkg/postgres/walker",
	}

	for _, pkg := range packages {
		if err := sg.analyzer.LoadPackage(pkg); err != nil {
			return fmt.Errorf("loading package %s: %w", pkg, err)
		}
	}

	return nil
}

// discoverSchemas discovers existing schema configurations
func (sg *SchemaGenerator) discoverSchemas() ([]SchemaData, error) {
	var configs []SchemaData

	// For now, let's start with a few well-known schemas
	// This could be enhanced to auto-discover from existing schema files
	knownSchemas := []struct {
		typeName        string
		tableName       string
		searchCategory  string
		scopingResource string
	}{
		{"Alert", "alerts", "ALERTS", "Alert"},
		{"Deployment", "deployments", "DEPLOYMENTS", "Deployment"},
		{"Image", "images", "IMAGES", "Image"},
		{"Policy", "policies", "POLICIES", "WorkflowAdministration"},
		{"Cluster", "clusters", "CLUSTERS", "Cluster"},
	}

	for _, schema := range knownSchemas {
		config := SchemaData{
			PackageName:     "schema",
			TableName:       schema.tableName,
			TypeName:        schema.typeName,
			StoragePackage:  "github.com/stackrox/rox/generated/storage",
			SearchCategory:  schema.searchCategory,
			ScopingResource: schema.scopingResource,
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// generateSchema generates a single schema file
func (sg *SchemaGenerator) generateSchema(config SchemaData) error {
	// Analyze the storage type
	typeInfo, err := sg.analyzer.AnalyzeType(config.StoragePackage, config.TypeName)
	if err != nil {
		return fmt.Errorf("analyzing type %s: %w", config.TypeName, err)
	}

	// Convert type info to schema fields
	config.Fields = sg.convertFieldsToSchema(typeInfo.Fields)

	// Generate the Go code
	code, err := sg.generateCode(config)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Format the code
	formattedCode, err := format.Source([]byte(code))
	if err != nil {
		return fmt.Errorf("formatting code: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(sg.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", sg.OutputDir, err)
	}

	// Write to file
	filename := fmt.Sprintf("generated_%s.go", config.TableName)
	filepath := filepath.Join(sg.OutputDir, filename)

	if sg.Verbose {
		log.Printf("Writing file: %s (size: %d bytes)", filepath, len(formattedCode))
	}

	if err := ioutil.WriteFile(filepath, formattedCode, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", filepath, err)
	}

	return nil
}

// convertFieldsToSchema converts analyzed field info to schema fields
func (sg *SchemaGenerator) convertFieldsToSchema(fields []FieldInfo) []SchemaField {
	var schemaFields []SchemaField

	for _, field := range fields {
		schemaField := SchemaField{
			Name:       field.Name,
			ColumnName: sg.fieldNameToColumnName(field.Name),
			Type:       field.Type,
			IsPointer:  field.IsPointer,
			IsSlice:    field.IsSlice,
		}

		// Determine SQL type and data type based on Go type
		sg.determineSchemaFieldTypes(&schemaField, field)

		// Parse SQL tag for additional options
		sg.parseSqlTag(&schemaField, field.SqlTag)

		// Check if field is searchable
		schemaField.IsSearchable = field.SearchTag != ""
		schemaField.SearchField = field.SearchTag

		schemaFields = append(schemaFields, schemaField)
	}

	return schemaFields
}

// fieldNameToColumnName converts Go field name to database column name
func (sg *SchemaGenerator) fieldNameToColumnName(fieldName string) string {
	// Convert CamelCase to snake_case
	var result strings.Builder
	for i, r := range fieldName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// determineSchemaFieldTypes determines SQL and data types for a field
func (sg *SchemaGenerator) determineSchemaFieldTypes(schemaField *SchemaField, field FieldInfo) {
	switch field.Kind {
	case reflect.String:
		schemaField.SQLType = "varchar"
		schemaField.DataType = "postgres.String"
	case reflect.Bool:
		schemaField.SQLType = "bool"
		schemaField.DataType = "postgres.Bool"
	case reflect.Int32:
		if sg.isEnumType(field.Type) {
			schemaField.SQLType = "integer"
			schemaField.DataType = "postgres.Enum"
		} else {
			schemaField.SQLType = "integer"
			schemaField.DataType = "postgres.Integer"
		}
	case reflect.Int64, reflect.Uint64:
		schemaField.SQLType = "bigint"
		schemaField.DataType = "postgres.BigInteger"
	case reflect.Float32, reflect.Float64:
		schemaField.SQLType = "numeric"
		schemaField.DataType = "postgres.Numeric"
	case reflect.Slice:
		if field.ElementKind == reflect.String {
			schemaField.SQLType = "text[]"
			schemaField.DataType = "postgres.StringArray"
		} else if field.ElementKind == reflect.Uint8 {
			schemaField.SQLType = "bytea"
			schemaField.DataType = "postgres.Bytes"
		} else if field.ElementKind == reflect.Int32 && sg.isEnumType(field.ElementType) {
			schemaField.SQLType = "int[]"
			schemaField.DataType = "postgres.EnumArray"
		} else {
			schemaField.SQLType = "jsonb"
			schemaField.DataType = "postgres.Map"
		}
	case reflect.Struct:
		if field.Type == "*time.Time" || field.Type == "time.Time" {
			schemaField.SQLType = "timestamp"
			schemaField.DataType = "postgres.DateTime"
		} else {
			// Embedded struct - will be flattened
			schemaField.SQLType = ""
			schemaField.DataType = ""
		}
	default:
		schemaField.SQLType = "jsonb"
		schemaField.DataType = "postgres.Map"
	}

	// Handle special case for serialized field
	if schemaField.Name == "serialized" || strings.ToLower(schemaField.Name) == "serialized" {
		schemaField.SQLType = "bytea"
		schemaField.DataType = "postgres.Bytes"
	}
}

// isEnumType checks if a type is a protobuf enum
func (sg *SchemaGenerator) isEnumType(typeName string) bool {
	// Simple heuristic: if it contains a package path and ends with an uppercase identifier,
	// and the type starts with "storage.", it's likely an enum
	return strings.Contains(typeName, "storage.") &&
		   strings.Contains(typeName, ".") &&
		   len(typeName) > 0 &&
		   typeName[len(typeName)-1] >= 'A' && typeName[len(typeName)-1] <= 'Z'
}

// parseSqlTag parses SQL struct tag for additional field options
func (sg *SchemaGenerator) parseSqlTag(schemaField *SchemaField, sqlTag string) {
	if sqlTag == "" {
		return
	}

	parts := strings.Split(sqlTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "pk" || part == "primary_key" {
			schemaField.IsPrimaryKey = true
		} else if strings.HasPrefix(part, "index") {
			schemaField.IsIndex = true
			if strings.Contains(part, "=") {
				// Extract index type
				indexPart := strings.Split(part, "=")[1]
				if strings.Contains(indexPart, ":") {
					schemaField.IndexType = strings.Split(indexPart, ":")[1]
				}
			}
			if schemaField.IndexType == "" {
				schemaField.IndexType = "btree"
			}
		} else if part == "fk" || strings.HasPrefix(part, "references") {
			schemaField.IsReference = true
		}
	}
}

// generateCode generates the Go code for a schema
func (sg *SchemaGenerator) generateCode(config SchemaData) (string, error) {
	tmpl := `// Code generated by generate-schema tool. DO NOT EDIT.

package {{.PackageName}}

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// Generated{{.TypeName}}Schema is the pre-computed schema for {{.TableName}} table
	Generated{{.TypeName}}Schema = &walker.Schema{
		Table:    "{{.TableName}}",
		Type:     "*storage.{{.TypeName}}",
		TypeName: "{{.TypeName}}",
		Fields: []walker.Field{
			{{range .Fields}}{{if .SQLType}}
			{
				Name:       "{{.Name}}",
				ColumnName: "{{.ColumnName}}",
				Type:       "{{.Type}}",
				SQLType:    "{{.SQLType}}",
				DataType:   {{.DataType}},
				{{if .IsPrimaryKey}}
				Options: walker.PostgresOptions{
					PrimaryKey: true,
				},
				{{else if .IsIndex}}
				Options: walker.PostgresOptions{
					Index: []*walker.PostgresIndexOptions{
						{IndexType: "{{.IndexType}}"},
					},
				},
				{{end}}
				{{if .IsSearchable}}
				Search: walker.SearchField{
					Enabled:   true,
					FieldName: "{{.SearchField}}",
				},
				{{end}}
			},
			{{end}}{{end}}
		},
		{{if .ScopingResource}}
		ScopingResource: resources.{{.ScopingResource}},
		{{end}}
	}
)

// Get{{.TypeName}}Schema returns the generated schema for {{.TableName}}
func Get{{.TypeName}}Schema() *walker.Schema {
	// Set up search options if not already done
	if Generated{{.TypeName}}Schema.OptionsMap == nil {
		Generated{{.TypeName}}Schema.SetOptionsMap(search.Walk(v1.SearchCategory_{{.SearchCategory}}, "{{.TableName}}", (*storage.{{.TypeName}})(nil)))
	}
	return Generated{{.TypeName}}Schema
}
`

	t, err := template.New("schema").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, config); err != nil {
		return "", err
	}

	return buf.String(), nil
}