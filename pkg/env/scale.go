package env

var (
	// ScaleTestEnabled signifies that a scale test is being run
	ScaleTestEnabled = RegisterBooleanSetting("ROX_SCALE_TEST", false)
)
