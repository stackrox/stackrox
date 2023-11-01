package manager

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// To run this benchmark download sample data from the Compliance Checks In Nodes design doc

func BenchmarkFold(b *testing.B) {
	b.Skip("ROX-20480: This test is failing. Skipping!")
	data := readCheckResults()
	nodes := []*storage.Node{
		{
			Id:   "test",
			Name: "test",
		},
	}
	domain := framework.NewComplianceDomain(nil, nodes, nil, nil, nil)

	nodeResults := map[string]map[string]*compliance.ComplianceStandardResult{
		"test": data,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		run := createRun("test", domain, &standards.Standard{
			Standard: metadata.Standard{
				ID:   "CIS_Docker_v1_2_0",
				Name: "CIS_Docker_v1_2_0",
			},
		})
		run.foldRemoteResults(nodeResults)
	}
}

func readCheckResults() map[string]*compliance.ComplianceStandardResult {
	jsonFile, err := os.Open("run_bench_test_data.json")
	if err != nil {
		panic(err)
	}
	defer utils.IgnoreError(jsonFile.Close)

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var data map[string]*compliance.ComplianceStandardResult
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}
	return data
}
