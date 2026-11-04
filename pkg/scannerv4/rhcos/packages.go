// Package rhcos identifies the synthetic node-image packages shared by node
// inventory and node scan conversion.
package rhcos

const (
	// LegacyPackageID and LegacySourceID identify the synthetic rhcos package pair.
	LegacyPackageID = "rhcos-pkg"
	LegacySourceID  = "rhcos-src"
	// PackageID and SourceID identify the canonical node-image package pair.
	PackageID = "rhcos-image-pkg"
	SourceID  = "rhcos-image-src"
)
