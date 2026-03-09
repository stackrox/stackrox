package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/csv"
	"github.com/stackrox/rox/operator/bundle_helpers/pkg/values"
	"helm.sh/helm/v3/pkg/chartutil"
)

// PatchCSV patches a ClusterServiceVersion YAML file with version and image information.
func PatchCSV(args []string) error {
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
	validModes := []string{"downstream", "omit", "konflux"}
	if !slices.Contains(validModes, *relatedImagesMode) {
		return fmt.Errorf("--related-images-mode must be one of: downstream, omit, konflux (got: %s)", *relatedImagesMode)
	}

	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Parse YAML
	doc, err := chartutil.ReadValues(input)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Handle --echo-replaced-version-only mode
	if *echoReplacedVersionOnly {
		return echoReplacedVersion(doc, *version, *firstVersion, *unreleased)
	}

	// Parse supported architectures
	var arches []string
	if *addSupportedArch != "" {
		arches = extractArches(*addSupportedArch)
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

	// Encode to YAML and normalize through Python to match PyYAML's exact formatting
	return encodeAndNormalizeYAML(doc, os.Stdout)
}

func echoReplacedVersion(doc chartutil.Values, version, firstVersion, unreleased string) error {
	name, err := values.GetString(doc, "metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get metadata.name: %w", err)
	}

	const expectedSuffix = ".v0.0.1"

	if !strings.HasSuffix(name, expectedSuffix) {
		return fmt.Errorf("metadata.name does not have suffix %s: %s", expectedSuffix, name)
	}

	rawName := strings.TrimSuffix(name, expectedSuffix)

	spec, err := doc.Table("spec")
	if err != nil {
		return fmt.Errorf("failed to get spec: %w", err)
	}

	_, replacedVersion, err := csv.CalculateReplacedVersionForCSV(
		version,
		firstVersion,
		unreleased,
		rawName,
		spec,
	)
	if err != nil {
		return err
	}

	if replacedVersion != nil {
		fmt.Println(replacedVersion.String())
	}

	return nil
}

func extractArches(s string) []string {
	if s == "" {
		return nil
	}
	parts := []string{}
	for p := range strings.SplitSeq(s, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
