package compliance

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/utils"
)

// To run this benchmark download sample data from the Compliance Checks In Nodes design doc

func BenchmarkRunChecks(b *testing.B) {
	data := readDataForBenchmark()
	run := &sensor.MsgToCompliance_TriggerRun{
		ScrapeId: "test",
		StandardIds: []string{
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}
	conf := &sensor.MsgToCompliance_ScrapeConfig{
		ContainerRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getCheckResults(run, conf, data)
	}
}

func BenchmarkCompressResults(b *testing.B) {
	data := readDataForBenchmark()
	run := &sensor.MsgToCompliance_TriggerRun{
		ScrapeId: "test",
		StandardIds: []string{
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}
	conf := &sensor.MsgToCompliance_ScrapeConfig{
		ContainerRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
	}
	results := getCheckResults(run, conf, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressResults(results)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkChecksAndCompression(b *testing.B) {
	data := readDataForBenchmark()
	run := &sensor.MsgToCompliance_TriggerRun{
		ScrapeId: "test",
		StandardIds: []string{
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}
	conf := &sensor.MsgToCompliance_ScrapeConfig{
		ContainerRuntime: storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := getCheckResults(run, conf, data)
		_, err := compressResults(results)
		if err != nil {
			panic(err)
		}
	}
}

func readDataForBenchmark() *standards.ComplianceData {
	jsonFile, err := os.Open("checks_bench_test_data.json")
	if err != nil {
		panic(err)
	}
	defer utils.IgnoreError(jsonFile.Close)

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var data standards.ComplianceData
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}
	return &data
}
