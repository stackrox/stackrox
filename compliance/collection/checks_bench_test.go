package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/utils"
)

// To run this benchmark download sample data from the Compliance Checks In Nodes design doc

func BenchmarkRunChecks(b *testing.B) {
	data := readDataForBenchmark()
	run := &sensor.MsgToCompliance_TriggerRun{
		ScrapeId: "test",
		StandardIds: []string{
			standards.CISDocker,
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getCheckResults(run, data)
	}
}

func BenchmarkCompressResults(b *testing.B) {
	data := readDataForBenchmark()
	run := &sensor.MsgToCompliance_TriggerRun{
		ScrapeId: "test",
		StandardIds: []string{
			standards.CISDocker,
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}

	results := getCheckResults(run, data)

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
			standards.CISDocker,
			standards.CISKubernetes,
			standards.NIST800190,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := getCheckResults(run, data)
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

	bytes, err := ioutil.ReadAll(jsonFile)
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
