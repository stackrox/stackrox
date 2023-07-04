package buildinfo

const (
	// ReleaseBuild indicates whether this build is a release build.
	ReleaseBuild bool = releaseBuild

	// TestBuild indicates whether this build is a test build.
	TestBuild bool = testBuild

	// BuildFlavor indicates the build flavor ("release" for release builds, "development" for development builds).
	BuildFlavor string = buildFlavor

	// RaceEnabled indicates whether the build was created with the race detector enabled. This usually only applies to
	// tests, and will be false for actual binary builds.
	RaceEnabled = raceEnabled
)
