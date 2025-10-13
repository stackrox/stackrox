package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	exceptionsFile = "govulncheck-allowlist.json"
)

// Exception is the metadata around the vuln's exception.
type Exception struct {
	Reason string     `json:"reason"`
	Until  *time.Time `json:"until"`
}

// ExceptionConfig is a wrapper around the vuln exceptions.
type ExceptionConfig struct {
	Exceptions map[string]*Exception `json:"exceptions"`
}

func printlnAndExitf(s string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, s+"\n", args...)
	os.Exit(1)
}

func readExceptionsFile() *ExceptionConfig {
	data, err := os.ReadFile(exceptionsFile)
	if err != nil {
		printlnAndExitf("unable to read %v: %v", exceptionsFile, err)
	}
	var ec ExceptionConfig
	if err := json.Unmarshal(data, &ec); err != nil {
		printlnAndExitf("unable to unmarshal %v: %v", exceptionsFile, err)
		os.Exit(1)
	}
	return &ec
}

// Output is the output of the program
type Output struct {
	Data []map[string]interface{} `json:"data"`
}

func parseData(data map[string]interface{}, osvIDs map[string]struct{}, osvEntries map[string]map[string]interface{}) {
	// OSV entries are now not tied to the module version but rather streamed continuously by govulncheck.
	// We need to cache every OSV entry detected by govulncheck and in the end only return the
	// ones which have findings on the module level.
	keyMap, exists := data["osv"]
	if exists {
		osvEntry := keyMap.(map[string]interface{})
		osvEntries[osvEntry["id"].(string)] = osvEntry
		return
	}

	// Findings are only created for OSV entries found in the _current_ module version.
	// Meaning, if we have a finding with a particular OSV ID, the current module is affected
	// by it.
	keyMap, exists = data["finding"]
	if !exists {
		return
	}

	findingMap := keyMap.(map[string]interface{})
	// Since the findings will potentially be duplicated when multiple traces are found
	// (e.g. when multiple locations within a module are affected), we de-duplicate here.
	osvID := findingMap["osv"].(string)
	if _, exists := osvIDs[osvID]; exists {
		return
	}
	// Add the finding to the list of existing OSV IDs to ensure we do not have duplicates.
	// Save to do here since we also want to de-duplicate OSV IDs with exceptions.
	osvIDs[osvID] = struct{}{}
}

func hasException(config *ExceptionConfig, id string) bool {
	if exception, ok := config.Exceptions[id]; ok {
		// Implies this vuln is excluded forever.
		if exception.Until == nil || exception.Until.After(time.Now()) {
			return true
		}
	}
	return false
}

func collectAffectedVulnerabilities(vulnFile string, exceptionConfig *ExceptionConfig) (*Output, error) {
	file, err := os.Open(vulnFile)
	if err != nil {
		return nil, fmt.Errorf("error opening vuln file: %w", err)
	}
	defer func() { _ = file.Close() }()

	output := &Output{
		Data: make([]map[string]interface{}, 0),
	}
	decoder := json.NewDecoder(file)
	uniqueOSVIDs := map[string]struct{}{}
	osvEntries := map[string]map[string]interface{}{}
	for {
		data := make(map[string]interface{})
		err := decoder.Decode(&data)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading vuln file: %w", err)
		}

		parseData(data, uniqueOSVIDs, osvEntries)
	}

	// Go through all the OSV IDs we found in findings and return the associated OSV entry
	// to the finding. The reason we return the OSV entry over the finding is because the
	// OSV entry has all the relevant metadata like summary and details for the vulnerability,
	// whereas the finding points to a specific trace within the code.
	for osvID := range uniqueOSVIDs {
		if !hasException(exceptionConfig, osvID) {
			output.Data = append(output.Data, osvEntries[osvID])
		}
	}

	return output, nil
}

func main() {
	if len(os.Args) < 2 {
		printlnAndExitf("Missing vulnerability file in arguments")
	}

	vulnFile := os.Args[1]

	exceptionConfig := readExceptionsFile()

	output, err := collectAffectedVulnerabilities(vulnFile, exceptionConfig)
	if err != nil {
		printlnAndExitf(err.Error())
	}

	outputBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		printlnAndExitf("error marshaling output: %v", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(outputBytes))
}
