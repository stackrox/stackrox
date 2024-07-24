package features

import "github.com/stackrox/rox/pkg/buildinfo"

type option func(*feature)

// Deprecated: once the feature needs to be enabled by default, the flag must be removed.
func withEnabledByDefault() option {
	return func(f *feature) {
		f.defaultValue = true
	}
}

func withTechPreviewStage() option {
	return func(f *feature) {
		f.techPreview = true
	}
}

func withUnchangeable(unchangeable bool) option {
	return func(f *feature) {
		f.unchangeable = unchangeable
	}
}

var (
	enabled            = withEnabledByDefault() // Deprecated: just remove the flag.
	techPreview        = withTechPreviewStage()
	unchangeableInProd = withUnchangeable(buildinfo.ReleaseBuild)
)
