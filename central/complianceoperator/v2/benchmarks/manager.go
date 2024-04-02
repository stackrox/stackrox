package benchmarks

import "github.com/stackrox/rox/generated/storage"

/*
- Create Benchmark
- Create Control
- Batch Create Control
- SAC
-
*/
type Manager interface {
	ImportBenchmark(filePath string) *storage.ComplianceOperatorBenchmark
}

type managerImpl struct {
}

func (m *managerImpl) ImportBenchmark(filePath string) {

}
