package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func cleanupImports(filePath string) (bool, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	originalContent := string(content)
	newContent := originalContent
	modified := false

	// Remove unused imports
	checks := []struct {
		importPath string
		pattern    string
	}{
		{`"reflect"`, `reflect\.`},
		{`"github.com/stackrox/rox/generated/storage"`, `storage\.`},
		{`"github.com/stackrox/rox/pkg/search"`, `search\.`},
	}

	for _, check := range checks {
		if strings.Contains(newContent, check.importPath) {
			// Check if the import is actually used
			used := regexp.MustCompile(check.pattern).MatchString(newContent)
			if !used {
				// Remove the import
				importRe := regexp.MustCompile(`\s*` + regexp.QuoteMeta(check.importPath) + `\n`)
				newContent = importRe.ReplaceAllString(newContent, "\n")
				modified = true
			}
		}
	}

	// Clean up import block formatting
	if modified {
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
	fmt.Println("🧹 Cleaning up unused imports...")

	// Find all schema files to process (excluding generated ones)
	files, err := filepath.Glob("pkg/postgres/schema/*.go")
	if err != nil {
		panic(err)
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
		modified, err := cleanupImports(file)
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
}