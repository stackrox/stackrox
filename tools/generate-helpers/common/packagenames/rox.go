package packagenames

import (
	"path"
)

const (
	// Rox is the root package path for the project
	Rox    = "github.com/stackrox/rox"
	roxPkg = Rox + "/pkg"
)

// PrefixRox prefixes the import path of StackRox to the given packageName
func PrefixRox(packageName string) string {
	return path.Join(Rox, packageName)
}

// PrefixRoxPkg prefixes the import path of StackRox pkg to the given packageName.
func PrefixRoxPkg(packageName string) string {
	return path.Join(roxPkg, packageName)
}
