package benchmark

// workloadNeedsFakeCollector reports whether to enable the gRPC FakeCollector harness.
// Steady-synthetic workloads inject network and process traffic via WorkloadManager
// directly; they do not require FakeCollector.
func workloadNeedsFakeCollector(_ string) (bool, error) {
	return false, nil
}
