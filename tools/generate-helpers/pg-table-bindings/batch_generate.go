package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
)

// schemaConfig holds the configuration for a schema that needs registry generation.
type schemaConfig struct {
	Table          string
	TypeName       string
	SearchCategory string
	SourceDir      string // Directory where the schema file is located
}

// batchGenerateRegistryFiles generates enum_registry_list.go and *_search.go files.
func batchGenerateRegistryFiles() error {
	fmt.Println("Discovering schemas...")
	schemas, err := discoverSchemas()
	if err != nil {
		return fmt.Errorf("failed to discover schemas: %w", err)
	}

	fmt.Printf("Found %d schemas with search categories\n", len(schemas))

	// Aggregate enum and search field data
	allEnums := make(map[string]map[string]int32)
	searchFieldsBySchema := make(map[string]schemaSearchInfo) // schema table -> search info

	for _, cfg := range schemas {
		fmt.Printf("Processing %s...\n", cfg.Table)

		// Get the message type (strip asterisk if present)
		typeName := strings.TrimPrefix(cfg.TypeName, "*")
		mt := protoutils.MessageType(typeName)
		if mt == nil {
			// Try common case variations (e.g., K8SRole -> K8sRole)
			alternateTypeName := strings.ReplaceAll(typeName, "K8S", "K8s")
			mt = protoutils.MessageType(alternateTypeName)
			if mt == nil {
				fmt.Printf("  WARNING: Could not find message type for %s, skipping...\n", typeName)
				continue
			}
			typeName = alternateTypeName
		}

		// Create instance and get prefix
		protoInstance := reflect.New(mt.Elem()).Interface()
		prefix := strings.ToLower(cfg.Table)

		// Capture enum state before and after search.Walk
		beforeEnums := enumregistry.Snapshot()
		categoryEnum := parseSearchCategory(cfg.SearchCategory)
		optionsMap := search.Walk(categoryEnum, prefix, protoInstance)
		afterEnums := enumregistry.Snapshot()

		// Collect new enums from this schema
		for path, values := range afterEnums {
			if _, existed := beforeEnums[path]; !existed {
				allEnums[path] = values
			}
		}

		// Generate search fields file content if there are search fields
		originalFields := optionsMap.Original()
		if len(originalFields) > 0 {
			searchFieldsSource := SerializeSearchFieldsFile(cfg.Table, cfg.SearchCategory, originalFields)
			searchFieldsBySchema[cfg.Table] = schemaSearchInfo{
				Source:    searchFieldsSource,
				SourceDir: cfg.SourceDir,
			}
		}
	}

	fmt.Printf("Total enum paths: %d\n", len(allEnums))
	fmt.Printf("Schemas with search fields: %d\n", len(searchFieldsBySchema))

	// Generate enum_registry_list.go
	enumRegistrySource := SerializeEnumRegistryFile(allEnums)
	enumRegistryPath := "pkg/search/enumregistry/enum_registry_list.go"
	if err := os.WriteFile(enumRegistryPath, []byte(enumRegistrySource), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", enumRegistryPath, err)
	}
	fmt.Printf("Generated %s\n", enumRegistryPath)

	// Generate *_search.go files
	for table, info := range searchFieldsBySchema {
		searchFilePath := filepath.Join(info.SourceDir, table+"_search.go")
		if err := os.WriteFile(searchFilePath, []byte(info.Source), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", searchFilePath, err)
		}
		fmt.Printf("Generated %s\n", searchFilePath)
	}

	return nil
}

// schemaSearchInfo holds search file information including the source directory
type schemaSearchInfo struct {
	Source    string
	SourceDir string
}

// discoverSchemas parses existing schema files to find schemas with search categories.
func discoverSchemas() ([]schemaConfig, error) {
	// Search in both main schema directory and migrator test schemas
	schemaDirs := []string{
		"pkg/postgres/schema",
		"migrator/migrations/postgreshelper/schema",
	}

	var schemas []schemaConfig

	for _, schemaDir := range schemaDirs {
		files, err := filepath.Glob(filepath.Join(schemaDir, "*.go"))
		if err != nil {
			return nil, fmt.Errorf("failed to list schema files in %s: %w", schemaDir, err)
		}

		dirSchemas, err := discoverSchemasInFiles(files)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, dirSchemas...)
	}

	return schemas, nil
}

// discoverSchemasInFiles extracts schema metadata from a list of Go files.
func discoverSchemasInFiles(files []string) ([]schemaConfig, error) {
	var schemas []schemaConfig
	for _, file := range files {
		// Skip generated test files, search files, and special files
		base := filepath.Base(file)
		if strings.HasSuffix(base, "_test.go") ||
			strings.HasSuffix(base, "_search.go") ||
			base == "schema.go" ||
			base == "register.go" ||
			base == "validate_generated_test.go" {
			continue
		}

		// Read file and extract schema metadata
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)

		// Extract table name from constant definition like: AlertsTableName = "alerts"
		// Pattern: {Something}TableName = "..."
		tableNamePattern := `(\w+)TableName = "([^"]+)"`
		tableNameMatches := regexp.MustCompile(tableNamePattern).FindStringSubmatch(contentStr)
		if len(tableNameMatches) < 3 {
			continue
		}
		tableName := tableNameMatches[2]

		// Extract search category from: schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_XXX,
		// Pattern: v1.SearchCategory_(\w+)
		searchCategoryPattern := `v1\.SearchCategory_(\w+)`
		categoryMatches := regexp.MustCompile(searchCategoryPattern).FindStringSubmatch(contentStr)
		if len(categoryMatches) < 2 {
			// No search category found, skip this file
			continue
		}
		searchCategory := categoryMatches[1]

		// Extract type name from: type XXX struct {
		// Look for the Gorm model struct definition, which comes after the schema
		typePattern := `type\s+(\w+)\s+struct\s+{`
		typeMatches := regexp.MustCompile(typePattern).FindAllStringSubmatch(contentStr, -1)
		if len(typeMatches) == 0 {
			continue
		}

		// The model struct name is usually {Table}CamelCase, e.g., AlertsSchema -> Alerts
		// We need to find the storage type name, which appears in imports or type assertions
		// Pattern: storage.XXX
		storageTypePattern := `walker\.Schema\{[^}]*Type:\s*"(\*storage\.\w+)"`
		storageMatches := regexp.MustCompile(storageTypePattern).FindStringSubmatch(contentStr)
		if len(storageMatches) < 2 {
			continue
		}
		typeName := strings.TrimPrefix(storageMatches[1], "*")

		// Extract source directory from file path
		sourceDir := filepath.Dir(file)

		schemas = append(schemas, schemaConfig{
			Table:          tableName,
			TypeName:       typeName,
			SearchCategory: searchCategory,
			SourceDir:      sourceDir,
		})
	}

	return schemas, nil
}
