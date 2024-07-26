package features

import "github.com/stackrox/rox/pkg/buildinfo"

type option func(*feature)

func withReleased() option {
	return func(f *feature) {
		f.released = true
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
	released           = withReleased()
	techPreview        = withTechPreviewStage()
	unchangeableInProd = withUnchangeable(buildinfo.ReleaseBuild)
)
