package env

var (
	// PruningLoadTestEnabled determines whether the TestPruningUnderHeavyLoad test in
	// central/pruning/pruning_test.go will be run during testing.
	PruningLoadTestEnabled = RegisterBooleanSetting("ROX_PRUNING_LOAD_TEST_ENABLED", false)
)
