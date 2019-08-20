package packagenames

import (
	"path"
)

const (
	rox    = "github.com/stackrox/rox"
	roxPkg = rox + "/pkg"
)

// PrefixRox prefixes the import path of StackRox to the given packageName
func PrefixRox(packageName string) string {
	return path.Join(rox, packageName)
}

// PrefixRoxPkg prefixes the import path of StackRox pkg to the given packageName.
func PrefixRoxPkg(packageName string) string {
	return path.Join(roxPkg, packageName)
}
