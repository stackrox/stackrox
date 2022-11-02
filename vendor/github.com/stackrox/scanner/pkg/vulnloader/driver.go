package vulnloader

var (
	loaders = make(map[string]Loader)
)

// Loader represents anything that can fetch vulnerabilities and store them in some directory.
type Loader interface {
	// DownloadFeedsToPath downloads vulnerability feeds into the given path.
	DownloadFeedsToPath(string) error
}

// RegisterLoader makes a Loader available by the provided name.
//
// If called twice with the same name, the name is blank, or if the provided
// Loader is nil, this function panics.
//
// Note: this function is not thread-safe, but should only be used in `init` functions.
func RegisterLoader(name string, l Loader) {
	if name == "" {
		panic("vulnloader: could not register a Loader with an empty name")
	}

	if l == nil {
		panic("vulnloader: could not register nil Loader")
	}

	if _, dup := loaders[name]; dup {
		panic("vulnloader: RegisterLoader called twice for " + name)
	}

	loaders[name] = l
}

// Loaders returns the list of the registered Loaders.
//
// Note: this function is not thread-safe.
func Loaders() map[string]Loader {
	return loaders
}
