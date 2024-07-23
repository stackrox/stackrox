package features

import "github.com/stackrox/rox/pkg/buildinfo"

type option func(*feature)

func withDefault(value bool) option {
	return func(f *feature) {
		f.enabled = value
	}
}

func withTechPreviewStage(stage bool) option {
	return func(f *feature) {
		f.techPreview = stage
	}
}

func withUnchangeable(unchangeable bool) option {
	return func(f *feature) {
		f.unchangeable = unchangeable
	}
}

var (
	disabled           = withDefault(false)
	enabled            = withDefault(true)
	devPreview         = withTechPreviewStage(false)
	techPreview        = withTechPreviewStage(true)
	unchangeable       = withUnchangeable(true)
	unchangeableInProd = withUnchangeable(buildinfo.ReleaseBuild)
)
