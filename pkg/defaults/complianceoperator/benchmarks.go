package complianceoperator

import (
	"embed"
	"path/filepath"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	benchmarkDir = "files"
)

var (
	log = logging.LoggerForModule()

	//go:embed files/*.json
	benchmarksFS embed.FS
)

func LoadComplianceOperatorBenchmarks() ([]*storage.ComplianceOperatorBenchmarkV2, error) {
	if !features.ComplianceEnhancements.Enabled() {
		return nil, nil
	}
	files, err := benchmarksFS.ReadDir(benchmarkDir)
	if err != nil {
		return nil, err
	}

	var benchmarks []*storage.ComplianceOperatorBenchmarkV2
	errList := errorhelpers.NewErrorList("Load Compliance Operator Benchmarks")
	for _, f := range files {
		b, err := readBenchmarksFile(filepath.Join(benchmarkDir, f.Name()))
		if err != nil {
			errList.AddError(err)
			continue
		}
		benchmarks = append(benchmarks, b)
	}
	return benchmarks, errList.ToError()
}

func readBenchmarksFile(path string) (*storage.ComplianceOperatorBenchmarkV2, error) {
	contents, err := benchmarksFS.ReadFile(path)
	utils.CrashOnError(err)

	var benchmark storage.ComplianceOperatorBenchmarkV2
	err = jsonutil.JSONBytesToProto(contents, &benchmark)
	if err != nil {
		log.Errorf("Unable to unmarshal benchmark (%s) json: %v", path, err)
		return nil, err
	}
	return &benchmark, nil
}
