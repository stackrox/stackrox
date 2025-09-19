package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func processFileForOptions(filePath string) (bool, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	originalContent := string(content)
	newContent := originalContent

	// Remove all forms of schema.SetOptionsMap calls with search.Walk
	patterns := []string{
		`\s*schema\.SetOptionsMap\(search\.Walk\([^)]+\)\)`,
		`\s*schema\.SetOptionsMap\(search\.Walk\([^)]+,\s*"[^"]+",\s*\([^)]+\)\)\)`,
		`\s*schema\.SetOptionsMap\(search\.Walk\([^)]+,\s*"[^"]+",\s*\([^)]+\)\s*\)\)`,
	}

	modified := false
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(newContent) {
			newContent = re.ReplaceAllString(newContent, "")
			modified = true
		}
	}

	// More aggressive multiline removal
	lines := strings.Split(newContent, "\n")
	var filteredLines []string
	skipNext := false

	for i, line := range lines {
		if skipNext {
			skipNext = false
			continue
		}

		// Check for SetOptionsMap calls that might span multiple lines
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "schema.SetOptionsMap(search.Walk(") {
			// Skip this line and potentially the next if it doesn't end with )
			modified = true
			if !strings.HasSuffix(trimmed, "))") && i+1 < len(lines) {
				skipNext = true
			}
			continue
		}

		filteredLines = append(filteredLines, line)
	}

	if modified {
		newContent = strings.Join(filteredLines, "\n")

		err = ioutil.WriteFile(filePath, []byte(newContent), 0644)
		if err != nil {
			return false, err
		}
	}

	return modified, nil
}

func main() {
	fmt.Println("🧹 Removing remaining schema.SetOptionsMap calls...")

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
		modified, err := processFileForOptions(file)
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

	// Count remaining walker.Walk calls
	cmd = exec.Command("grep", "-r", "walker\\.Walk", "pkg/postgres/schema/")
	output, err = cmd.CombinedOutput()

	if err != nil {
		if cmd.ProcessState.ExitCode() == 1 {
			fmt.Println("🎉 No remaining walker.Walk calls found!")
		}
	} else {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		fmt.Printf("📊 Remaining walker.Walk calls: %d\n", len(lines))
	}

	// Count remaining SetOptionsMap calls
	cmd = exec.Command("grep", "-r", "schema\\.SetOptionsMap.*search\\.Walk", "pkg/postgres/schema/")
	output, err = cmd.CombinedOutput()

	if err != nil {
		if cmd.ProcessState.ExitCode() == 1 {
			fmt.Println("🎉 No remaining schema.SetOptionsMap(search.Walk(...)) calls found!")
		}
	} else {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		fmt.Printf("📊 Remaining schema.SetOptionsMap calls: %d\n", len(lines))
		if len(lines) <= 5 {
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
		}
	}
}