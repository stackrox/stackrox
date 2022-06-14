package manager

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/central/compliance/standards"
	"github.com/stackrox/stackrox/central/compliance/standards/metadata"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/utils"
)

// To run this benchmark download sample data from the Compliance Checks In Nodes design doc

func BenchmarkFold(b *testing.B) {
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
