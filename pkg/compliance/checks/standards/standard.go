package standards

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// Standards is the global map of standard names to checks
var (
	Standards = make(map[string]map[string]*CheckAndInterpretation)
)

// CheckAndInterpretation is a pair matching a Check to an interpretation text
type CheckAndInterpretation struct {
	CheckFunc          Check
	InterpretationText string
}

// RegisterChecksForStandard takes a standard name and some Checks and adds them to the golabl registry
func RegisterChecksForStandard(standardName string, standardChecks map[string]*CheckAndInterpretation) {
	standard, ok := Standards[standardName]
	if !ok {
		Standards[standardName] = standardChecks
		return
	}

	for checkName, check := range standardChecks {
		if _, ok := standard[checkName]; ok {
			utils.Should(errors.Errorf("duplicate check in collector: %s", checkName))
		}
		standard[checkName] = check
	}
}

// CheckName takes a standard name and a check ID and returns a properly formatted check name
func CheckName(standardName, checkName string) string {
	return standardName + ":" + checkName
}
