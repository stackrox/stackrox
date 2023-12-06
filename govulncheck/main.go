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

// Exception is the metadata around the vuln's exception
type Exception struct {
	Reason string     `json:"reason"`
	Until  *time.Time `json:"until"`
}

// ExceptionConfig is a wrapper around the vuln exceptions
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

func parseData(config *ExceptionConfig, data map[string]interface{}) (map[string]interface{}, bool) {
	keyMap, exists := data["finding"]
	if exists {
		keyMapI := keyMap.(map[string]interface{})
		return keyMapI, !hasException(config, keyMapI["osv"].(string))
	}

	keyMap, exists = data["osv"]
	if exists {
		keyMapI := keyMap.(map[string]interface{})
		return keyMapI, !hasException(config, keyMapI["id"].(string))
	}
	return nil, false
}

func hasException(config *ExceptionConfig, id string) bool {
	if exception, ok := config.Exceptions[id]; ok {
		// Implies this vuln is excluded forever
		if exception.Until == nil || exception.Until.After(time.Now()) {
			return true
		}
	}
	return false
}

func main() {
	exceptionConfig := readExceptionsFile()

	file, err := os.Open(os.Args[1])
	if err != nil {
		printlnAndExitf("error opening vulns file: %v", err)
		os.Exit(1)
	}
	defer func() { _ = file.Close() }()

	var output Output
	decoder := json.NewDecoder(file)
	for {
		data := make(map[string]interface{})
		err := decoder.Decode(&data)
		if err != nil {
			if err == io.EOF {
				break
			}
			printlnAndExitf("error reading vulns file: %v", err)
			os.Exit(1)
		}

		findingMap, valid := parseData(exceptionConfig, data)
		if !valid {
			continue
		}
		output.Data = append(output.Data, findingMap)
	}

	outputBytes, err := json.MarshalIndent(&output, "", "  ")
	if err != nil {
		printlnAndExitf("error marshaling output: %v", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(outputBytes))
}
