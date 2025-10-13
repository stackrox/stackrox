package features

import "github.com/stackrox/rox/pkg/buildinfo"

type option func(*feature)

func withEnabledByDefault() option {
	return func(f *feature) {
		f.defaultValue = true
	}
}

func withUnchangeable(unchangeable bool) option {
	return func(f *feature) {
		f.unchangeable = unchangeable
	}
}

var (
	enabled            = withEnabledByDefault()
	unchangeableInProd = withUnchangeable(buildinfo.ReleaseBuild)
)
