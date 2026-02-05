package rpm

// PackageInfo represents information about an RPM package
type PackageInfo struct {
	Name    string
	Version string
	Release string
	Arch    string
}

// FullVersion returns the combined version-release string
func (p PackageInfo) FullVersion() string {
	return p.Version + "-" + p.Release
}
