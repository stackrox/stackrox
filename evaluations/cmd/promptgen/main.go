// Command promptgen reconstructs the exact LLM prompt that StackRox's
// GetDeploymentRiskAISummary sends for a deployment. It reads the raw JSON body
// of a GET /v1/deploymentswithrisk/{id} API response (from stdin or --input) and
// writes the byte-identical prompt to stdout.
//
// It reuses the production prompt/sanitization code
// (central/deployment/service/aisummary), so the evaluated prompt cannot drift
// from what Central actually sends.
//
// Usage:
//
//	promptgen --input dep_with_risk.json
//	cat dep_with_risk.json | promptgen
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/stackrox/rox/central/deployment/service/aisummary"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
)

func main() {
	input := flag.String("input", "", "path to a GET /v1/deploymentswithrisk/{id} JSON response file (default: stdin)")
	flag.Parse()

	if err := run(*input); err != nil {
		fmt.Fprintln(os.Stderr, "promptgen:", err)
		os.Exit(1)
	}
}

func run(inputPath string) error {
	data, err := readInput(inputPath)
	if err != nil {
		return err
	}

	var resp v1.GetDeploymentWithRiskResponse
	if err := jsonutil.JSONBytesToProto(data, &resp); err != nil {
		return fmt.Errorf("parsing deployment-with-risk response: %w", err)
	}

	query, err := aisummary.BuildQuery(resp.GetDeployment(), resp.GetRisk())
	if err != nil {
		return fmt.Errorf("building prompt: %w", err)
	}

	_, err = fmt.Print(query)
	return err
}

func readInput(inputPath string) ([]byte, error) {
	if inputPath == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(inputPath)
}
