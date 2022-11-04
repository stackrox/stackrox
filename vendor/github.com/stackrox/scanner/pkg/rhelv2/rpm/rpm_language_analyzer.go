package rpm

import (
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/scanner/pkg/analyzer"
	"github.com/stackrox/scanner/pkg/component"
	"github.com/stackrox/scanner/pkg/rpm"
)

// AnnotateComponentsWithPackageManagerInfo checks for each component if it was installed by the package manager,
// and sets the `FromPackageManager` attribute accordingly.
func AnnotateComponentsWithPackageManagerInfo(files analyzer.Files, components []*component.Component) error {
	if len(components) == 0 {
		return nil
	}
	rpmDB, err := rpm.CreateDatabaseFromImage(files)
	if err != nil {
		return err
	}
	if rpmDB == nil {
		return nil
	}
	defer rpmDB.Delete()
	locationAlreadyChecked := make(map[string]bool)
	for _, c := range components {
		// This handles jar-in-jar cases as the location is manually created so we only want
		// the initial path
		normalizedLocation := stringutils.GetUpTo(c.Location, ":")
		fromPackageManager, ok := locationAlreadyChecked[normalizedLocation]
		if ok {
			c.FromPackageManager = fromPackageManager
			continue
		}
		c.FromPackageManager = rpmDB.ProvidesFile(normalizedLocation)
		locationAlreadyChecked[normalizedLocation] = c.FromPackageManager
	}
	return nil
}
