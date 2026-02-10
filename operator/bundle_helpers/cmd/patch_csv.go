package cmd

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/csv"
	"gopkg.in/yaml.v3"
)

// PatchCSV patches a ClusterServiceVersion YAML file with version and image information.
func PatchCSV(args []string) error {
	// Create flag set for patch-csv command
	flags := flag.NewFlagSet("patch-csv", flag.ExitOnError)

	version := flags.String("use-version", "", "SemVer version of the operator (required)")
	firstVersion := flags.String("first-version", "", "First version of operator ever published (required)")
	operatorImage := flags.String("operator-image", "", "Operator image reference (required)")
	relatedImagesMode := flags.String("related-images-mode", "downstream", "Mode for related images: downstream, omit, konflux")
	addSupportedArch := flags.String("add-supported-arch", "amd64,arm64,ppc64le,s390x", "Comma-separated list of supported architectures")
	echoReplacedVersionOnly := flags.Bool("echo-replaced-version-only", false, "Only compute and print replaced version")
	unreleased := flags.String("unreleased", "", "Not yet released version, if any")

	// Custom usage function
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: bundle-helper patch-csv [options] < input.yaml > output.yaml")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Patches ClusterServiceVersion files with version updates, image replacements,")
		fmt.Fprintln(os.Stderr, "and related images configuration.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Handle help
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		flags.Usage()
		return nil
	}

	// Validate required flags
	if *version == "" || *firstVersion == "" || *operatorImage == "" {
		fmt.Fprintln(os.Stderr, "Error: --use-version, --first-version, and --operator-image are required")
		flags.Usage()
		return errors.New("missing required flags")
	}

	// Validate related-images-mode
	validModes := map[string]bool{"downstream": true, "omit": true, "konflux": true}
	if !validModes[*relatedImagesMode] {
		return fmt.Errorf("--related-images-mode must be one of: downstream, omit, konflux (got: %s)", *relatedImagesMode)
	}

	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Parse YAML
	var doc map[string]any
	if err := yaml.Unmarshal(input, &doc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Handle --echo-replaced-version-only mode
	if *echoReplacedVersionOnly {
		return echoReplacedVersion(doc, *version, *firstVersion, *unreleased)
	}

	// Parse supported architectures
	var arches []string
	if *addSupportedArch != "" {
		arches = splitComma(*addSupportedArch)
	}

	// Patch the CSV
	opts := csv.PatchOptions{
		Version:             *version,
		OperatorImage:       *operatorImage,
		FirstVersion:        *firstVersion,
		RelatedImagesMode:   *relatedImagesMode,
		ExtraSupportedArchs: arches,
		Unreleased:          *unreleased,
	}

	if err := csv.PatchCSV(doc, opts); err != nil {
		return fmt.Errorf("failed to patch CSV: %w", err)
	}

	// Encode to YAML using Go's yaml.v3
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	// Normalize through Python to match PyYAML's exact formatting
	return normalizeYAMLOutput(buf.Bytes(), os.Stdout)
}

func echoReplacedVersion(doc map[string]any, version, firstVersion, unreleased string) error {
	metadata, ok := doc["metadata"].(map[string]any)
	if !ok {
		return errors.New("metadata is not a map")
	}
	name, ok := metadata["name"].(string)
	if !ok {
		return errors.New("metadata.name is not a string")
	}

	rawName := ""
	if name == "rhacs-operator.v0.0.1" {
		rawName = "rhacs-operator"
	} else {
		return fmt.Errorf("unexpected metadata.name format: %s", name)
	}

	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		return errors.New("spec is not a map")
	}

	skips := make([]csv.XyzVersion, 0)
	if rawSkips, ok := spec["skips"].([]any); ok {
		for _, s := range rawSkips {
			skipStr, ok := s.(string)
			if !ok {
				return errors.New("skip entry is not a string")
			}
			if skipStr == rawName+".v0.0.1" {
				continue
			}
			// Extract version from "rhacs-operator.vX.Y.Z"
			skipVer := strings.TrimPrefix(skipStr, rawName+".v")

			v, err := csv.ParseXyzVersion(skipVer)
			if err != nil {
				return err
			}
			skips = append(skips, v)
		}
	}

	previousYStream, err := csv.GetPreviousYStream(version)
	if err != nil {
		return err
	}

	replacedVersion, err := csv.CalculateReplacedVersion(version, firstVersion, previousYStream, skips, unreleased)
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
	for _, p := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
