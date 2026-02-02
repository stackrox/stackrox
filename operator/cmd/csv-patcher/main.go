package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

func main() {
	// Parse command-line flags
	version := flag.String("use-version", "", "SemVer version of the operator (required)")
	firstVersion := flag.String("first-version", "", "First version of operator ever published (required)")
	operatorImage := flag.String("operator-image", "", "Operator image reference (required)")
	relatedImagesMode := flag.String("related-images-mode", "downstream", "Mode for related images: downstream, omit, konflux")
	addSupportedArch := flag.String("add-supported-arch", "amd64,arm64,ppc64le,s390x", "Comma-separated list of supported architectures")
	echoReplacedVersionOnly := flag.Bool("echo-replaced-version-only", false, "Only compute and print replaced version")
	unreleased := flag.String("unreleased", "", "Not yet released version, if any")

	flag.Parse()

	if *version == "" || *firstVersion == "" || *operatorImage == "" {
		fmt.Fprintln(os.Stderr, "Error: --use-version, --first-version, and --operator-image are required")
		flag.Usage()
		os.Exit(1)
	}

	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal(input, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Handle --echo-replaced-version-only mode
	if *echoReplacedVersionOnly {
		if err := echoReplacedVersion(doc, *version, *firstVersion, *unreleased); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Parse supported architectures
	var arches []string
	if *addSupportedArch != "" {
		arches = splitComma(*addSupportedArch)
	}

	// Patch the CSV
	opts := PatchOptions{
		Version:             *version,
		OperatorImage:       *operatorImage,
		FirstVersion:        *firstVersion,
		RelatedImagesMode:   *relatedImagesMode,
		ExtraSupportedArchs: arches,
		Unreleased:          *unreleased,
	}

	if err := PatchCSV(doc, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error patching CSV: %v\n", err)
		os.Exit(1)
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
		os.Exit(1)
	}

	// Write to stdout
	fmt.Print(string(output))
}

func echoReplacedVersion(doc map[string]interface{}, version, firstVersion, unreleased string) error {
	metadata := doc["metadata"].(map[string]interface{})
	name := metadata["name"].(string)

	rawName := ""
	if name == "rhacs-operator.v0.0.1" {
		rawName = "rhacs-operator"
	} else {
		return fmt.Errorf("unexpected metadata.name format: %s", name)
	}

	spec := doc["spec"].(map[string]interface{})
	skips := make([]XyzVersion, 0)
	if rawSkips, ok := spec["skips"].([]interface{}); ok {
		for _, s := range rawSkips {
			skipStr := s.(string)
			skipVer := ""
			if skipStr == rawName+".v0.0.1" {
				continue
			}
			// Extract version from "rhacs-operator.vX.Y.Z"
			parts := splitString(skipStr, ".")
			if len(parts) >= 2 {
				skipVer = parts[len(parts)-3] + "." + parts[len(parts)-2] + "." + parts[len(parts)-1]
				skipVer = trimPrefix(skipVer, "v")
			}

			v, err := ParseXyzVersion(skipVer)
			if err != nil {
				return err
			}
			skips = append(skips, v)
		}
	}

	previousYStream, err := GetPreviousYStream(version)
	if err != nil {
		return err
	}

	replacedVersion, err := CalculateReplacedVersion(version, firstVersion, previousYStream, skips, unreleased)
	if err != nil {
		return err
	}

	if replacedVersion != nil {
		fmt.Println(replacedVersion.String())
	}

	return nil
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for _, p := range splitString(s, ",") {
		if trimmed := trimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := []string{}
	current := ""
	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
