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

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
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

	// Generate missing core entities directly from storage types
	missingEntities := []struct {
		TypeName       string
		TableName      string
		SearchCategory string
	}{
		{"Alert", "alerts", "ALERTS"},
		{"Policy", "policies", "POLICIES"},
		{"Node", "nodes", "NODES"},
	}

	for _, entity := range missingEntities {
		// Skip if entity filter is specified and doesn't match
		if sg.EntityFilter != "" && sg.EntityFilter != entity.TypeName {
			continue
		}

		// Check if internal file already exists
		internalPath := filepath.Join(sg.OutputDir, "internal", entity.TableName+".go")
		if _, err := os.Stat(internalPath); err == nil {
			if sg.Verbose {
				log.Printf("Skipping %s: internal file already exists", entity.TypeName)
			}
			continue
		}

		if err := sg.generateSchemaFromType(entity.TypeName, entity.TableName, entity.SearchCategory); err != nil {
			return fmt.Errorf("generating schema for %s: %w", entity.TypeName, err)
		}
		if sg.Verbose {
			log.Printf("Generated schema for %s", entity.TypeName)
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
		PackageName:     "internal",
		TableName:       tableName,
		TypeName:        typeName,
		StoragePackage:  "github.com/stackrox/rox/generated/storage",
		SearchCategory:  searchCategory,
		ScopingResource: scopingResource,
	}, nil
}

// extractSearchCategory extracts search category from file content
func (sg *SchemaGenerator) extractSearchCategory(content, typeName string) string {
	// First, look for RegisterCategoryToTable calls (these are authoritative)
	registerPattern := `RegisterCategoryToTable\(v1\.SearchCategory_([A-Z_0-9]+)`
	re := regexp.MustCompile(registerPattern)
	matches := re.FindStringSubmatch(content)
	if len(matches) == 2 {
		return matches[1]
	}

	// Second, look for schema.SetOptionsMap(search.Walk(...)) calls
	setOptionsMapPattern := `schema\.SetOptionsMap\(search\.Walk\(v1\.SearchCategory_([A-Z_0-9]+)`
	re2 := regexp.MustCompile(setOptionsMapPattern)
	matches2 := re2.FindStringSubmatch(content)
	if len(matches2) == 2 {
		return matches2[1]
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
		// Entities from our analysis that DO have SearchCategory
		"ImageComponentV2": "IMAGE_COMPONENTS_V2",
		"ImageCVEV2":       "IMAGE_VULNERABILITIES_V2",
		"ImageV2":          "IMAGES_V2",
		"K8SRole":          "ROLES",
		"K8SRoleBinding":   "ROLEBINDINGS",

		// Previously working entities
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

		// All remaining entities that don't use SearchCategory (based on analysis)
		"AuthMachineToMachineConfig":                    "",
		"ClusterInitBundle":                             "",
		"ComplianceConfig":                              "",
		"ComplianceOperatorCheckResult":                 "",
		"ComplianceOperatorProfile":                     "",
		"ComplianceOperatorRule":                        "",
		"ComplianceOperatorScanSettingBinding":          "",
		"ComplianceOperatorScan":                        "",
		"ComplianceStrings":                             "",
		"Config":                                        "",
		"DeclarativeConfigHealth":                       "",
		"DelegatedRegistryConfig":                       "",
		"ExternalBackup":                                "",
		"Group":                                         "",
		"Hash":                                          "",
		"InstallationInfo":                              "",
		"IntegrationHealth":                             "",
		"LogImbue":                                      "",
		"NetworkFlowV2":                                 "",
		"NetworkGraphConfig":                            "",
		"NetworkPolicyApplicationUndoDeploymentRecord": "",
		"NetworkPolicyApplicationUndoRecord":           "",
		"NotificationSchedule":                          "",
		"NotifierEncConfig":                             "",
		"Notifier":                                      "",
		"PermissionSet":                                 "",
		"SensorUpgradeConfig":                           "",
		"ServiceIdentity":                               "",
		"SignatureIntegration":                          "",
		"SimpleAccessScope":                             "",
		"SystemInfo":                                    "",
		"Version":                                       "",
		"WatchedImage":                                  "",

		// Test entities - all use SEARCH_UNSET
		"TestStruct":               "SEARCH_UNSET",
		"TestParent3":              "SEARCH_UNSET",
		"TestSingleUUIDKeyStruct":  "SEARCH_UNSET",
		"TestGGrandChild1":         "SEARCH_UNSET",
		"TestGrandChild1":          "SEARCH_UNSET",
		"TestParent4":              "SEARCH_UNSET",
		"TestSingleKeyStruct":      "SEARCH_UNSET",
		"TestChild1":               "SEARCH_UNSET",
		"TestG3GrandChild1":        "SEARCH_UNSET",
		"TestChild2":               "SEARCH_UNSET",
		"TestChild1P4":             "SEARCH_UNSET",
		"TestGrandparent":          "SEARCH_UNSET",
		"TestG2GrandChild1":        "SEARCH_UNSET",
		"TestParent1":              "SEARCH_UNSET",
		"TestShortCircuit":         "SEARCH_UNSET",
		"TestParent2":              "SEARCH_UNSET",
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
	internalDir := filepath.Join(sg.OutputDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", internalDir, err)
	}

	// Write to file
	filename := fmt.Sprintf("%s.go", config.TableName)
	filepath := filepath.Join(sg.OutputDir, "internal", filename)

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

// generateSearchFields generates search field data using search.Walk directly
func (sg *SchemaGenerator) generateSearchFields(fields []FieldInfo, searchCategory string) []SearchFieldData {
	return sg.generateSearchFieldsFromWalk(searchCategory)
}

// generateSearchFieldsFromWalk uses search.Walk to generate exact search fields
func (sg *SchemaGenerator) generateSearchFieldsFromWalk(searchCategory string) []SearchFieldData {
	// Map search category to storage type, v1.SearchCategory, and entity prefix
	typeMap := map[string]struct {
		storageType reflect.Type
		category    v1.SearchCategory
		prefix      string
	}{
		"ALERTS":      {reflect.TypeOf((*storage.Alert)(nil)), v1.SearchCategory_ALERTS, ""},
		"POLICIES":    {reflect.TypeOf((*storage.Policy)(nil)), v1.SearchCategory_POLICIES, ""},
		"DEPLOYMENTS": {reflect.TypeOf((*storage.Deployment)(nil)), v1.SearchCategory_DEPLOYMENTS, ""},
		"NODES":       {reflect.TypeOf((*storage.Node)(nil)), v1.SearchCategory_NODES, ""},
		"ROLEBINDINGS": {reflect.TypeOf((*storage.K8SRoleBinding)(nil)), v1.SearchCategory_ROLEBINDINGS, "k8srolebinding"},
		"ROLES":       {reflect.TypeOf((*storage.Role)(nil)), v1.SearchCategory_ROLES, ""},
	}

	typeInfo, exists := typeMap[searchCategory]
	if !exists {
		if sg.Verbose {
			log.Printf("No type mapping for search category: %s", searchCategory)
		}
		return []SearchFieldData{}
	}

	// Use search.Walk to get the exact search fields with the correct entity prefix
	searchOptionsMap := search.Walk(typeInfo.category, typeInfo.prefix, reflect.Zero(typeInfo.storageType).Interface())
	originalSearchFields := searchOptionsMap.Original()

	var searchFields []SearchFieldData
	for fieldLabel, field := range originalSearchFields {
		searchField := SearchFieldData{
			FieldLabel: string(fieldLabel),
			FieldPath:  field.FieldPath,
			Store:      field.Store,
			Hidden:     field.Hidden,
			Analyzer:   field.Analyzer,
		}

		// Map v1.SearchDataType to string
		switch field.Type {
		case v1.SearchDataType_SEARCH_STRING:
			searchField.DataType = "STRING"
		case v1.SearchDataType_SEARCH_BOOL:
			searchField.DataType = "BOOL"
		case v1.SearchDataType_SEARCH_NUMERIC:
			searchField.DataType = "NUMERIC"
		case v1.SearchDataType_SEARCH_ENUM:
			searchField.DataType = "ENUM"
		case v1.SearchDataType_SEARCH_DATETIME:
			searchField.DataType = "DATETIME"
		case v1.SearchDataType_SEARCH_MAP:
			searchField.DataType = "MAP"
		default:
			searchField.DataType = "STRING"
		}

		searchFields = append(searchFields, searchField)
	}

	return searchFields
}


// parseSearchTagWithField parses a search struct tag and returns SearchFieldData using FieldInfo
func (sg *SchemaGenerator) parseSearchTagWithField(searchTag string, field FieldInfo, searchCategory string) *SearchFieldData {
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

	// Convert field path to start with dot instead of entity name
	fieldPath := field.Name
	// If path contains entity prefix, remove it and add dot
	if strings.Contains(fieldPath, ".") {
		// Find first dot and convert to standard format
		parts := strings.SplitN(fieldPath, ".", 2)
		if len(parts) == 2 {
			fieldPath = "." + parts[1]
		}
	} else {
		// Simple field name, add dot prefix
		fieldPath = "." + fieldPath
	}

	// Debug output
	if sg.Verbose {
		fmt.Printf("DEBUG: Converting fieldName '%s' to fieldPath '%s'\n", field.Name, fieldPath)
	}

	searchField := &SearchFieldData{
		FieldLabel: fieldLabel,
		FieldPath:  fieldPath,
		DataType:   sg.getSearchDataType(field),
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

// parseSearchTag parses a search struct tag and returns SearchFieldData (legacy method)
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

	// Convert field path to start with dot instead of entity name
	fieldPath := fieldName
	// If path contains entity prefix, remove it and add dot
	if strings.Contains(fieldPath, ".") {
		// Find first dot and convert to standard format
		parts := strings.SplitN(fieldPath, ".", 2)
		if len(parts) == 2 {
			fieldPath = "." + parts[1]
		}
	} else {
		// Simple field name, add dot prefix
		fieldPath = "." + fieldPath
	}

	// Debug output
	if sg.Verbose {
		fmt.Printf("DEBUG: Converting fieldName '%s' to fieldPath '%s'\n", fieldName, fieldPath)
	}

	searchField := &SearchFieldData{
		FieldLabel: fieldLabel,
		FieldPath:  fieldPath,
		DataType:   sg.getSearchDataType(FieldInfo{Name: fieldName}),
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
func (sg *SchemaGenerator) getSearchDataType(field FieldInfo) string {
	// Handle time fields - look for .seconds suffix which search.Walk adds for timestamps
	if strings.HasSuffix(field.Name, ".seconds") {
		return "DATETIME"
	}

	// Map based on Go type
	switch field.Kind {
	case reflect.String:
		return "STRING"
	case reflect.Bool:
		return "BOOL"
	case reflect.Int32:
		if sg.isEnumType(field.Type) {
			return "ENUM"
		}
		return "NUMERIC"
	case reflect.Int64, reflect.Uint64, reflect.Float32, reflect.Float64:
		return "NUMERIC"
	case reflect.Slice:
		if field.ElementKind == reflect.String {
			return "STRING"
		}
		return "STRING"
	default:
		// Fallback based on field name patterns
		switch {
		case strings.Contains(strings.ToLower(field.Name), "time"):
			return "DATETIME"
		case strings.Contains(strings.ToLower(field.Name), "id"):
			return "STRING"
		case strings.Contains(strings.ToLower(field.Name), "name"):
			return "STRING"
		case strings.Contains(strings.ToLower(field.Name), "count"):
			return "NUMERIC"
		default:
			return "STRING"
		}
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
	"github.com/stackrox/rox/pkg/search"
)

var (
	// {{.TypeName}}SearchFields contains pre-computed search fields for {{.TableName}}
	{{.TypeName}}SearchFields = map[search.FieldLabel]*search.Field{
		{{range .SearchFields}}
		"{{.FieldLabel}}": {
			FieldPath: "{{.FieldPath}}",
			Type:      v1.SearchDataType_SEARCH_{{.DataType}},
			Store:     {{.Store}},
			Hidden:    {{.Hidden}},
			{{if $.SearchCategory}}Category:  v1.SearchCategory_{{$.SearchCategory}},{{end}}
			{{if .Analyzer}}Analyzer:  "{{.Analyzer}}",{{end}}
		},
		{{end}}
	}

	// {{.TypeName}}Schema is the pre-computed schema for {{.TableName}} table
	{{.TypeName}}Schema = &walker.Schema{
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
	}
)

// Get{{.TypeName}}Schema returns the generated schema for {{.TableName}}
func Get{{.TypeName}}Schema() *walker.Schema {
	// Set up search options if not already done
	if {{.TypeName}}Schema.OptionsMap == nil {
		{{if .SearchCategory}}{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_{{.SearchCategory}}, {{.TypeName}}SearchFields)){{else}}{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_SEARCH_UNSET, {{.TypeName}}SearchFields)){{end}}
	}
	return {{.TypeName}}Schema
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

// generateSchemaFromType generates a schema directly from a storage type using walker.Walk
func (sg *SchemaGenerator) generateSchemaFromType(typeName, tableName, searchCategory string) error {
	// Map type name to reflection type
	typeMap := map[string]reflect.Type{
		"Alert":         reflect.TypeOf((*storage.Alert)(nil)),
		"Policy":        reflect.TypeOf((*storage.Policy)(nil)),
		"Deployment":    reflect.TypeOf((*storage.Deployment)(nil)),
		"Node":          reflect.TypeOf((*storage.Node)(nil)),
		"K8SRoleBinding": reflect.TypeOf((*storage.K8SRoleBinding)(nil)),
		"Role":          reflect.TypeOf((*storage.Role)(nil)),
	}

	storageType, exists := typeMap[typeName]
	if !exists {
		return fmt.Errorf("no type mapping for %s", typeName)
	}

	// Use walker.Walk to get the schema structure
	walkerSchema := walker.Walk(storageType, tableName)

	// Use search.Walk to get search fields
	searchFields := sg.generateSearchFieldsFromWalk(searchCategory)

	// Generate code directly using the walker schema
	return sg.generateSchemaFromWalkerSchema(typeName, walkerSchema, searchFields, searchCategory)
}

// generateSchemaFromWalkerSchema generates schema code directly from walker.Schema
func (sg *SchemaGenerator) generateSchemaFromWalkerSchema(typeName string, walkerSchema *walker.Schema, searchFields []SearchFieldData, searchCategory string) error {
	// Generate the Go code using the walker schema directly
	code, err := sg.generateCodeFromWalkerSchema(typeName, walkerSchema, searchFields, searchCategory)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	// Format the code
	formattedCode, err := format.Source([]byte(code))
	if err != nil {
		return fmt.Errorf("formatting code: %w", err)
	}

	// Ensure output directory exists
	internalDir := filepath.Join(sg.OutputDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", internalDir, err)
	}

	// Write to file
	filename := fmt.Sprintf("%s.go", walkerSchema.Table)
	filepath := filepath.Join(sg.OutputDir, "internal", filename)

	if sg.Verbose {
		log.Printf("Writing file: %s (size: %d bytes)", filepath, len(formattedCode))
	}

	if err := ioutil.WriteFile(filepath, formattedCode, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", filepath, err)
	}

	return nil
}

// generateCodeFromWalkerSchema generates Go code that returns the walker schema directly
func (sg *SchemaGenerator) generateCodeFromWalkerSchema(typeName string, walkerSchema *walker.Schema, searchFields []SearchFieldData, searchCategory string) (string, error) {
	// Map type name to storage import path
	typeMap := map[string]string{
		"Alert":         "(*storage.Alert)(nil)",
		"Policy":        "(*storage.Policy)(nil)",
		"Deployment":    "(*storage.Deployment)(nil)",
		"Node":          "(*storage.Node)(nil)",
		"K8SRoleBinding": "(*storage.K8SRoleBinding)(nil)",
		"Role":          "(*storage.Role)(nil)",
	}

	storageType, exists := typeMap[typeName]
	if !exists {
		return "", fmt.Errorf("no storage type mapping for %s", typeName)
	}

	tmpl := `// Code generated by generate-schema tool. DO NOT EDIT.

package internal

import (
	"reflect"
	"sync"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// {{.TypeName}}SearchFields contains pre-computed search fields for {{.TableName}}
	{{.TypeName}}SearchFields = map[search.FieldLabel]*search.Field{
		{{range .SearchFields}}
		"{{.FieldLabel}}": {
			FieldPath: "{{.FieldPath}}",
			Type:      v1.SearchDataType_SEARCH_{{.DataType}},
			Store:     {{.Store}},
			Hidden:    {{.Hidden}},
			{{if $.SearchCategory}}Category:  v1.SearchCategory_{{$.SearchCategory}},{{end}}
			{{if .Analyzer}}Analyzer:  "{{.Analyzer}}",{{end}}
		},
		{{end}}
	}

	{{.TypeName}}SchemaOnce sync.Once
	{{.TypeName}}Schema *walker.Schema
)

// Get{{.TypeName}}Schema returns the walker.Walk generated schema for {{.TableName}}
func Get{{.TypeName}}Schema() *walker.Schema {
	{{.TypeName}}SchemaOnce.Do(func() {
		{{.TypeName}}Schema = walker.Walk(reflect.TypeOf({{.StorageType}}), "{{.TableName}}")
		{{if .SearchCategory}}{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_{{.SearchCategory}}, {{.TypeName}}SearchFields)){{else}}{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_SEARCH_UNSET, {{.TypeName}}SearchFields)){{end}}
	})
	return {{.TypeName}}Schema
}
`

	data := struct {
		TypeName       string
		TableName      string
		StorageType    string
		SearchFields   []SearchFieldData
		SearchCategory string
	}{
		TypeName:       typeName,
		TableName:      walkerSchema.Table,
		StorageType:    storageType,
		SearchFields:   searchFields,
		SearchCategory: searchCategory,
	}

	t, err := template.New("schema").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// convertWalkerFieldsToSchema converts walker.Field slice to SchemaField slice
func (sg *SchemaGenerator) convertWalkerFieldsToSchema(walkerFields []walker.Field) []SchemaField {
	var schemaFields []SchemaField

	for _, walkerField := range walkerFields {
		schemaField := SchemaField{
			Name:       walkerField.Name,
			ColumnName: walkerField.ColumnName,
			Type:       walkerField.Type,
			SQLType:    walkerField.SQLType,
			DataType:   sg.convertDataTypeToString(walkerField.DataType),
			IsPointer:  false, // Not directly available from walker
			IsSlice:    false, // Not directly available from walker
		}

		// Parse options if they exist
		if walkerField.Options.PrimaryKey {
			schemaField.IsPrimaryKey = true
		}

		if len(walkerField.Options.Index) > 0 {
			schemaField.IsIndex = true
			schemaField.IndexType = walkerField.Options.Index[0].IndexType
		}

		// Check if field is searchable
		if walkerField.Search.Enabled {
			schemaField.IsSearchable = true
			schemaField.SearchField = walkerField.Search.FieldName
		}

		schemaFields = append(schemaFields, schemaField)
	}

	return schemaFields
}

// convertDataTypeToString converts postgres.DataType to string representation
func (sg *SchemaGenerator) convertDataTypeToString(dataType postgres.DataType) string {
	switch dataType {
	case postgres.String:
		return "postgres.String"
	case postgres.Bool:
		return "postgres.Bool"
	case postgres.Integer:
		return "postgres.Integer"
	case postgres.BigInteger:
		return "postgres.BigInteger"
	case postgres.Numeric:
		return "postgres.Numeric"
	case postgres.DateTime:
		return "postgres.DateTime"
	case postgres.StringArray:
		return "postgres.StringArray"
	case postgres.EnumArray:
		return "postgres.EnumArray"
	case postgres.Enum:
		return "postgres.Enum"
	case postgres.Map:
		return "postgres.Map"
	case postgres.Bytes:
		return "postgres.Bytes"
	default:
		return "postgres.String" // Default fallback
	}
}

// convertToSchemaFields converts FieldInfo slice to SchemaField slice
func (sg *SchemaGenerator) convertToSchemaFields(fields []FieldInfo) []SchemaField {
	var schemaFields []SchemaField

	for _, field := range fields {
		schemaField := SchemaField{
			Name:         field.Name,
			ColumnName:   sg.toSnakeCase(field.Name),
			Type:         field.Type,
			SQLType:      sg.getSQLType(field.Kind, field.Type),
			DataType:     sg.getDataType(field.Kind, field.Type),
			IsPointer:    field.IsPointer,
			IsSlice:      field.IsSlice,
			IsSearchable: field.SearchTag != "",
			SearchField:  field.SearchTag,
		}

		// Parse SQL tag for additional field options
		sg.parseSqlTag(&schemaField, field.SqlTag)

		schemaFields = append(schemaFields, schemaField)
	}

	return schemaFields
}

// toSnakeCase converts CamelCase to snake_case
func (sg *SchemaGenerator) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteByte('_')
		}
		result.WriteRune(r | 0x20) // to lowercase
	}
	return result.String()
}

// getSQLType returns the appropriate SQL type for a Go type
func (sg *SchemaGenerator) getSQLType(kind reflect.Kind, typeName string) string {
	if strings.HasPrefix(typeName, "[]") {
		if strings.Contains(typeName, "string") {
			return "text[]"
		}
		return "jsonb"
	}
	if strings.HasPrefix(typeName, "map[") {
		return "jsonb"
	}
	if strings.Contains(typeName, ".") && !strings.Contains(typeName, "time.Time") {
		return "jsonb" // Complex types as JSON
	}

	switch kind {
	case reflect.String:
		return "varchar"
	case reflect.Int, reflect.Int32:
		return "integer"
	case reflect.Int64:
		return "bigint"
	case reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "numeric"
	case reflect.Bool:
		return "bool"
	default:
		return "jsonb"
	}
}

// getDataType returns the appropriate postgres.DataType for a Go type
func (sg *SchemaGenerator) getDataType(kind reflect.Kind, typeName string) string {
	if strings.HasPrefix(typeName, "[]") {
		if strings.Contains(typeName, "string") {
			return "postgres.StringArray"
		}
		return "postgres.Map"
	}
	if strings.HasPrefix(typeName, "map[") {
		return "postgres.Map"
	}
	if strings.Contains(typeName, ".") && !strings.Contains(typeName, "time.Time") {
		return "postgres.Map" // Complex types as JSON
	}

	switch kind {
	case reflect.String:
		return "postgres.String"
	case reflect.Int, reflect.Int32:
		return "postgres.Integer"
	case reflect.Int64:
		return "postgres.BigInteger"
	case reflect.Uint64:
		return "postgres.BigInteger"
	case reflect.Float32, reflect.Float64:
		return "postgres.Numeric"
	case reflect.Bool:
		return "postgres.Bool"
	default:
		return "postgres.Map"
	}
}