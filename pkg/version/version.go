package version

var (
	mainVersion      string
	collectorVersion string
	clairifyVersion  string
)

// GetMainVersion returns the tag of Prevent
func GetMainVersion() string {
	return mainVersion
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	return collectorVersion
}

// GetClairifyVersion returns the current clairify tag
func GetClairifyVersion() string {
	return clairifyVersion
}
