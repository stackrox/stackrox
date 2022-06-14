package standards

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// NodeChecks is the global map of standard names to checks
	NodeChecks = make(map[string]map[string]*CheckAndMetadata)
	// StandardDependencies defines the standards mapped to their dependencies for evaluation
	// This map is populated via init functions in the respective standard files if necessary
	StandardDependencies = make(map[string]set.StringSet)
)

// Metadata contains metadata about a Check
type Metadata struct {
	InterpretationText string
	TargetKind         framework.TargetKind
}

// CheckAndMetadata is a pair matching a Check to an interpretation text
type CheckAndMetadata struct {
	CheckFunc Check
	Metadata  *Metadata
}

// RegisterChecksForStandard takes a standard name and some Checks and adds them to the golabl registry
func RegisterChecksForStandard(standardName string, standardChecks map[string]*CheckAndMetadata) {
	for _, checkAndMetadata := range standardChecks {
		if checkAndMetadata.Metadata == nil {
			checkAndMetadata.Metadata = &Metadata{
				// All of these checks are expected to run in the nodes.  If no metadata is specified we assume the target is NodeKind.
				TargetKind: framework.NodeKind,
			}
		}
	}

	standard, ok := NodeChecks[standardName]
	if !ok {
		NodeChecks[standardName] = standardChecks
		return
	}

	for checkName, checkAndMetadata := range standardChecks {
		if _, ok := standard[checkName]; ok {
			utils.Should(errors.Errorf("duplicate check in collector: %s", checkName))
		}
		standard[checkName] = checkAndMetadata
	}
}

// CheckName takes a standard name and a check ID and returns a properly formatted check name
func CheckName(standardName, checkName string) string {
	return standardName + ":" + checkName
}
