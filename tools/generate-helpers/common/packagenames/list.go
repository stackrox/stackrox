package packagenames

// This block enumerates well-known Rox package names.
var (
	Metrics    = PrefixRox("central/metrics")
	Ops        = PrefixRoxPkg("metrics")
	V1         = PrefixRox("generated/api/v1")
	RoxBleve   = PrefixRoxPkg("search/blevesearch")
	RoxSearch  = PrefixRoxPkg("search")
	RoxCentral = PrefixRox("central")
	Sync       = PrefixRoxPkg("sync")
)
