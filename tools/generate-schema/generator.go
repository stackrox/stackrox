package main

import (
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

// SchemaGenerator generates PostgreSQL schema files
type SchemaGenerator struct {
	ProjectRoot  string
	OutputDir    string
	Verbose      bool
	EntityFilter string
	analyzer     *TypeAnalyzer
}

// SchemaData represents the data needed to generate a schema file
type SchemaData struct {
	PackageName     string
	TableName       string
	TypeName        string
	StoragePackage  string
	Fields          []SchemaField
	SearchFields    []SearchFieldData
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

// SearchFieldData represents a search field for compile-time generation
type SearchFieldData struct {
	FieldLabel   string
	FieldPath    string
	DataType     string
	Store        bool
	Hidden       bool
	Analyzer     string
}

// RunDiscovery runs only the discovery phase for testing
func (sg *SchemaGenerator) RunDiscovery() error {
	configs, err := sg.discoverSchemas()
	if err != nil {
		return fmt.Errorf("discovering schemas: %w", err)
	}

	fmt.Printf("Discovered %d schema configurations:\n", len(configs))
	for i, config := range configs {
		fmt.Printf("%d. %s -> %s (category: %s, resource: %s)\n",
			i+1, config.TypeName, config.TableName, config.SearchCategory, config.ScopingResource)
	}

	return nil
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
	return sg.discoverSchemasFromFiles()
}

// discoverSchemasFromFiles scans schema files to find walker.Walk usages
func (sg *SchemaGenerator) discoverSchemasFromFiles() ([]SchemaData, error) {
	var configs []SchemaData

	// Scan schema directory for walker.Walk usages
	schemaDir := filepath.Join(sg.ProjectRoot, "pkg/postgres/schema")

	files, err := ioutil.ReadDir(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".go") || strings.HasPrefix(file.Name(), "generated_") {
			continue
		}

		filePath := filepath.Join(schemaDir, file.Name())
		config, err := sg.extractSchemaFromFile(filePath)
		if err != nil {
			if sg.Verbose {
				log.Printf("Skipping %s: %v", file.Name(), err)
			}
			continue
		}

		if config != nil {
			// Apply entity filter if specified
			if sg.EntityFilter == "" || config.TypeName == sg.EntityFilter {
				configs = append(configs, *config)
			}
		}
	}

	return configs, nil
}

// extractSchemaFromFile extracts schema configuration from a Go file
func (sg *SchemaGenerator) extractSchemaFromFile(filePath string) (*SchemaData, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	// Look for walker.Walk pattern: walker.Walk(reflect.TypeOf((*storage.TypeName)(nil)), "table_name")
	walkerPattern := `walker\.Walk\(reflect\.TypeOf\(\(\*storage\.([^)]+)\)\(nil\)\),\s*"([^"]+)"\)`
	re := regexp.MustCompile(walkerPattern)

	matches := re.FindStringSubmatch(contentStr)
	if len(matches) != 3 {
		return nil, fmt.Errorf("no walker.Walk pattern found")
	}

	typeName := matches[1]
	tableName := matches[2]

	// Extract search category from search.Walk pattern
	searchCategory := sg.extractSearchCategory(contentStr, typeName)

	// Extract scoping resource
	scopingResource := sg.extractScopingResource(contentStr, typeName)

	return &SchemaData{
		PackageName:     "schema",
		TableName:       tableName,
		TypeName:        typeName,
		StoragePackage:  "github.com/stackrox/rox/generated/storage",
		SearchCategory:  searchCategory,
		ScopingResource: scopingResource,
	}, nil
}

// extractSearchCategory extracts search category from file content
func (sg *SchemaGenerator) extractSearchCategory(content, typeName string) string {
	// Look for v1.SearchCategory_CATEGORY pattern
	searchPattern := `v1\.SearchCategory_([A-Z_]+)`
	re := regexp.MustCompile(searchPattern)

	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 2 {
			category := match[1]
			// Skip internal categories like SEARCH
			if !strings.Contains(category, "SEARCH") {
				return category
			}
		}
	}

	// Default mapping based on type name
	return sg.defaultSearchCategory(typeName)
}

// extractScopingResource extracts scoping resource from file content
func (sg *SchemaGenerator) extractScopingResource(content, typeName string) string {
	// Look for resources.ResourceName pattern
	resourcePattern := `resources\.([A-Za-z]+)`
	re := regexp.MustCompile(resourcePattern)

	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}

	// Default mapping based on type name
	return sg.defaultScopingResource(typeName)
}

// defaultSearchCategory provides default search category mapping
func (sg *SchemaGenerator) defaultSearchCategory(typeName string) string {
	categoryMap := map[string]string{
		"Alert":          "ALERTS",
		"Deployment":     "DEPLOYMENTS",
		"Image":          "IMAGES",
		"Policy":         "POLICIES",
		"Cluster":        "CLUSTERS",
		"AuthProvider":   "AUTH_PROVIDERS",
		"Role":           "ROLES",
		"Node":           "NODES",
		"Secret":         "SECRETS",
		"Namespace":      "NAMESPACES",
		"ServiceAccount": "SERVICE_ACCOUNTS",
	}

	if category, ok := categoryMap[typeName]; ok {
		return category
	}

	// Default: uppercase type name
	return strings.ToUpper(typeName) + "S"
}

// defaultScopingResource provides default scoping resource mapping
func (sg *SchemaGenerator) defaultScopingResource(typeName string) string {
	resourceMap := map[string]string{
		"Alert":                    "Alert",
		"Deployment":               "Deployment",
		"Image":                    "Image",
		"Policy":                   "WorkflowAdministration",
		"Cluster":                  "Cluster",
		"AuthProvider":             "Access",
		"Role":                     "Access",
		"K8SRole":                  "Access",
		"K8SRoleBinding":           "Access",
		"PermissionSet":            "Access",
		"Group":                    "Access",
		"ServiceAccount":           "Access",
		"ServiceIdentity":          "Access",
		"Secret":                   "Secret",
		"Node":                     "Node",
		"Namespace":                "Namespace",
		"ComplianceOperatorScan":   "Compliance",
		"ComplianceOperatorProfile": "Compliance",
	}

	if resource, ok := resourceMap[typeName]; ok {
		return resource
	}

	// Default: same as type name
	return typeName
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

	// Generate search fields
	config.SearchFields = sg.generateSearchFields(typeInfo.Fields, config.SearchCategory)

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

// generateSearchFields generates search field data from analyzed fields
func (sg *SchemaGenerator) generateSearchFields(fields []FieldInfo, searchCategory string) []SearchFieldData {
	var searchFields []SearchFieldData
	seenFields := make(map[string]bool)

	for _, field := range fields {
		if field.SearchTag == "" {
			continue
		}

		// Parse search tag
		searchFieldData := sg.parseSearchTag(field.SearchTag, field.Name, searchCategory)
		if searchFieldData != nil && !seenFields[searchFieldData.FieldLabel] {
			searchFields = append(searchFields, *searchFieldData)
			seenFields[searchFieldData.FieldLabel] = true
		}
	}

	return searchFields
}

// parseSearchTag parses a search struct tag and returns SearchFieldData
func (sg *SchemaGenerator) parseSearchTag(searchTag, fieldName, searchCategory string) *SearchFieldData {
	if searchTag == "" || searchTag == "-" {
		return nil
	}

	parts := strings.Split(searchTag, ",")
	if len(parts) == 0 {
		return nil
	}

	fieldLabel := parts[0]
	if fieldLabel == "" {
		return nil
	}

	searchField := &SearchFieldData{
		FieldLabel: fieldLabel,
		FieldPath:  fieldName,
		DataType:   sg.getSearchDataType(fieldName),
	}

	// Parse additional options
	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		switch part {
		case "hidden":
			searchField.Hidden = true
		case "store":
			searchField.Store = true
		default:
			if strings.HasPrefix(part, "analyzer=") {
				searchField.Analyzer = strings.TrimPrefix(part, "analyzer=")
			}
		}
	}

	return searchField
}

// getSearchDataType maps Go types to search data types
func (sg *SchemaGenerator) getSearchDataType(fieldName string) string {
	// This is a simplified mapping - in a real implementation we'd need
	// to analyze the actual field type from the TypeInfo
	switch {
	case strings.Contains(strings.ToLower(fieldName), "time"):
		return "DATETIME"
	case strings.Contains(strings.ToLower(fieldName), "id"):
		return "STRING"
	case strings.Contains(strings.ToLower(fieldName), "name"):
		return "STRING"
	case strings.Contains(strings.ToLower(fieldName), "count"):
		return "NUMERIC"
	default:
		return "STRING"
	}
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
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// generated{{.TypeName}}SearchFields contains pre-computed search fields for {{.TableName}}
	generated{{.TypeName}}SearchFields = map[search.FieldLabel]*search.Field{
		{{range .SearchFields}}
		"{{.FieldLabel}}": {
			FieldPath: "{{.FieldPath}}",
			Type:      v1.SearchDataType_SEARCH_{{.DataType}},
			Store:     {{.Store}},
			Hidden:    {{.Hidden}},
			Category:  v1.SearchCategory_{{$.SearchCategory}},
			{{if .Analyzer}}Analyzer:  "{{.Analyzer}}",{{end}}
		},
		{{end}}
	}

	// generated{{.TypeName}}Schema is the pre-computed schema for {{.TableName}} table
	generated{{.TypeName}}Schema = &walker.Schema{
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
	if generated{{.TypeName}}Schema.OptionsMap == nil {
		generated{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_{{.SearchCategory}}, generated{{.TypeName}}SearchFields))
	}
	return generated{{.TypeName}}Schema
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