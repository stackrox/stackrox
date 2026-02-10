package internal

import "runtime/debug"

// DeriveVersionFromBuildVCS populates MainVersion and GitShortSha from Go's
// built-in buildvcs info (vcs.revision, vcs.modified). This avoids recompilation
// when the commit SHA changes — buildvcs stamps VCS info via the linker, outside
// the build cache key.
//
// Called from version_data_generated.go's init() after setting BaseVersion and
// component versions. For release builds where MainVersion is already set (via
// BUILD_TAG), this is a no-op.
func DeriveVersionFromBuildVCS() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		setFallback()
		return
	}

	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if revision == "" {
		setFallback()
		return
	}

	shortSha := revision
	if len(shortSha) > 10 {
		shortSha = shortSha[:10]
	}

	if GitShortSha == "" {
		GitShortSha = shortSha
	}

	if MainVersion == "" {
		MainVersion = BaseVersion + "-g" + shortSha
		if modified {
			MainVersion += "-dirty"
		}
	}
}

func setFallback() {
	if MainVersion == "" {
		MainVersion = BaseVersion
	}
	if GitShortSha == "" {
		GitShortSha = "unknown"
	}
}
