package env

var (
	// ScaleTestEnabled signifies that a scale test is being run
	ScaleTestEnabled = RegisterBooleanSetting("ROX_SCALE_TEST", false)

	// FakeWorkloadStoragePath signifies the path where we should store IDs for the fake workload to avoid reconciliation
	// If unset, then no storage will occur
	FakeWorkloadStoragePath = RegisterSetting("ROX_FAKE_WORKLOAD_STORAGE")
)
