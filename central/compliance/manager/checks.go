package manager

import (
	"fmt"

	_ "github.com/stackrox/rox/central/compliance/checks" // Make sure all checks are available
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/compliance/standards"
)

var (
	checksByStandardID = make(map[string][]framework.Check)
)

func init() {
	for _, standard := range standards.RegistrySingleton().AllStandards() {
		checks := getChecksForStandard(standard)
		checksByStandardID[standard.ID] = checks
		log.Infof("Compliance standard %s: found checks for %d/%d controls", standard.Name, len(checks), len(standard.AllControlIDs(false)))
	}
}

func checksForStandardID(standardID string) ([]framework.Check, error) {
	checks, ok := checksByStandardID[standardID]
	if !ok {
		return nil, fmt.Errorf("invalid standard id %q", standardID)
	}
	return checks, nil
}

func getChecksForStandard(standard *standards.Standard) []framework.Check {
	var checks []framework.Check
	for _, controlID := range standard.AllControlIDs(true) {
		check := framework.RegistrySingleton().Lookup(controlID)
		if check != nil {
			checks = append(checks, check)
		}
	}
	return checks
}
