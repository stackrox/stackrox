package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/descriptor"
	"helm.sh/helm/v3/pkg/chartutil"
)

// FixSpecDescriptorOrder fixes the ordering of specDescriptors in a CSV file.
// It reads from stdin and writes to stdout, matching the Python script behavior.
func FixSpecDescriptorOrder(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println("Usage: bundle-helper fix-spec-descriptor-order < input.yaml > output.yaml")
		fmt.Println()
		fmt.Println("Fixes the ordering of specDescriptors in a ClusterServiceVersion YAML file.")
		fmt.Println("Ensures parent descriptors appear before their children.")
		return nil
	}

	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	out, err := fixSpecDescriptorOrderBytes(in)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(out)
	return err
}

// fixSpecDescriptorOrderBytes fixes the ordering of specDescriptors in CSV YAML bytes
func fixSpecDescriptorOrderBytes(in []byte) ([]byte, error) {
	csvDoc, err := chartutil.ReadValues(in)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := descriptor.FixCSVDescriptorsMap(csvDoc); err != nil {
		return nil, fmt.Errorf("failed to fix descriptors: %w", err)
	}

	var buf bytes.Buffer
	if err := encodeAndNormalizeYAML(csvDoc, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

