package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func getTableMappings() map[string]string {
	files, err := filepath.Glob("pkg/postgres/schema/generated_*.go")
	if err != nil {
		panic(err)
	}

	mapping := make(map[string]string)

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)

		// Find function name
		funcRe := regexp.MustCompile(`func (Get\w+Schema)\(\)`)
		funcMatch := funcRe.FindStringSubmatch(contentStr)
		if len(funcMatch) < 2 {
			continue
		}
		funcName := funcMatch[1]

		// Find table name
		tableRe := regexp.MustCompile(`Table:\s*"([^"]+)"`)
		tableMatch := tableRe.FindStringSubmatch(contentStr)
		if len(tableMatch) < 2 {
			continue
		}
		tableName := tableMatch[1]

		mapping[tableName] = funcName
	}

	return mapping
}

func createGoPatchRule(tableName, funcName string) string {
	return fmt.Sprintf(`@@
var table string = "%s"
var schema walker.Schema
@@
-schema = walker.Walk(reflect.TypeOf((*storage.$_)(nil)), table)
+schema = %s()
`, tableName, funcName)
}

func createOptionsRemovalRule() string {
	return `@@
var schema walker.Schema
@@
-schema.SetOptionsMap(search.Walk($_))
`
}

func processFile(filePath string, mappings map[string]string) (bool, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	originalContent := string(content)
	modified := false

	// Pattern to find walker.Walk calls with table names
	walkerRe := regexp.MustCompile(`schema = walker\.Walk\(reflect\.TypeOf\(\(\*storage\.\w+\)\(nil\)\), "([^"]+)"\)`)

	newContent := walkerRe.ReplaceAllStringFunc(originalContent, func(match string) string {
		submatches := walkerRe.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		tableName := submatches[1]
		if funcName, ok := mappings[tableName]; ok {
			modified = true
			return fmt.Sprintf("schema = %s()", funcName)
		}
		return match
	})

	// Remove SetOptionsMap calls (more comprehensive pattern)
	optionsRe := regexp.MustCompile(`\s*schema\.SetOptionsMap\(search\.Walk\([^)]+\)\)`)
	if optionsRe.MatchString(newContent) {
		newContent = optionsRe.ReplaceAllString(newContent, "")
		modified = true
	}

	// Also remove any remaining search.Walk calls in SetOptionsMap that span multiple lines
	multilineOptionsRe := regexp.MustCompile(`\s*schema\.SetOptionsMap\(search\.Walk\([^)]+,\s*"[^"]+",\s*\([^)]+\)\s*\)\)`)
	if multilineOptionsRe.MatchString(newContent) {
		newContent = multilineOptionsRe.ReplaceAllString(newContent, "")
		modified = true
	}

	// Remove unused imports if modifications were made
	if modified {
		// Remove reflect import if no longer used
		if !strings.Contains(newContent, "reflect.") && strings.Contains(newContent, `"reflect"`) {
			newContent = regexp.MustCompile(`\s*"reflect"\n`).ReplaceAllString(newContent, "\n")
		}

		// Remove storage import if no longer used
		if !strings.Contains(newContent, "storage.") && strings.Contains(newContent, `"github.com/stackrox/rox/generated/storage"`) {
			newContent = regexp.MustCompile(`\s*"github\.com/stackrox/rox/generated/storage"\n`).ReplaceAllString(newContent, "\n")
		}

		// Remove search import if no longer used and no search.Walk calls remain
		if !strings.Contains(newContent, "search.Walk") && !strings.Contains(newContent, "search.OptionsMapFromMap") && strings.Contains(newContent, `"github.com/stackrox/rox/pkg/search"`) {
			newContent = regexp.MustCompile(`\s*"github\.com/stackrox/rox/pkg/search"\n`).ReplaceAllString(newContent, "\n")
		}

		// Clean up import formatting
		newContent = regexp.MustCompile(`import \(\n\n+`).ReplaceAllString(newContent, "import (\n")
		newContent = regexp.MustCompile(`\n\n+\)`).ReplaceAllString(newContent, "\n)")

		err = ioutil.WriteFile(filePath, []byte(newContent), 0644)
		if err != nil {
			return false, err
		}
	}

	return modified, nil
}

func main() {
	fmt.Println("🔍 Building table->function mappings...")
	mappings := getTableMappings()
	fmt.Printf("📊 Found %d table->function mappings\n", len(mappings))

	// Find all schema files to process (excluding generated ones)
	files, err := filepath.Glob("pkg/postgres/schema/*.go")
	if err != nil {
		log.Fatal(err)
	}

	var filesToProcess []string
	for _, file := range files {
		if !strings.Contains(file, "generated_") {
			filesToProcess = append(filesToProcess, file)
		}
	}

	fmt.Printf("📁 Processing %d schema files...\n", len(filesToProcess))

	modifiedCount := 0
	for _, file := range filesToProcess {
		modified, err := processFile(file, mappings)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", file, err)
			continue
		}

		if modified {
			modifiedCount++
			fmt.Printf("✅ Modified: %s\n", filepath.Base(file))
		}
	}

	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("   Files processed: %d\n", len(filesToProcess))
	fmt.Printf("   Files modified: %d\n", modifiedCount)
	fmt.Printf("   Available mappings: %d\n", len(mappings))

	// Test compilation
	fmt.Println("\n🔨 Testing compilation...")
	cmd := exec.Command("go", "build", "./pkg/postgres/schema")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("❌ Compilation failed:")
		fmt.Println(string(output))
		return
	}

	fmt.Println("✅ Package compiles successfully!")

	// Count remaining walker.Walk calls
	cmd = exec.Command("grep", "-r", "walker\\.Walk", "pkg/postgres/schema/")
	output, err = cmd.CombinedOutput()

	if err != nil {
		if cmd.ProcessState.ExitCode() == 1 {
			fmt.Println("🎉 No remaining walker.Walk calls found!")
		} else {
			fmt.Printf("Error checking walker.Walk calls: %v\n", err)
		}
	} else {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		fmt.Printf("📊 Remaining walker.Walk calls: %d\n", len(lines))

		// Show first few remaining calls for debugging
		if len(lines) <= 10 {
			fmt.Println("Remaining calls:")
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
		}
	}
}