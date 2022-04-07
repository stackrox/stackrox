package env

var (
	// ScaleTesting determines that Central is running scale testing and may relax some features
	ScaleTesting = RegisterBooleanSetting("ROX_SCALE_TESTING", false)
)
