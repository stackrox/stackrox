package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

func main() {
	// Find all generated schema files
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

	fmt.Printf("Found %d table->function mappings:\n", len(mapping))
	for table, function := range mapping {
		fmt.Printf("%s -> %s\n", table, function)
	}
}