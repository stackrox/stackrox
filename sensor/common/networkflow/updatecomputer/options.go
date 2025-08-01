package updatecomputer

import (
	"github.com/stackrox/rox/pkg/logging"
)

var optionsLog = logging.LoggerForModule()

// NewUpdateComputer creates a new update computer of the specified type
func NewUpdateComputer(updateType UpdateComputerType) UpdateComputer {
	switch updateType {
	case LegacyUpdateComputerType:
		return NewLegacyUpdateComputer()
	case CategorizedUpdateComputerType:
		return NewCategorizedUpdateComputer()
	default:
		optionsLog.Warnf("Unknown update computer type %q, defaulting to categorized", updateType)
		return NewCategorizedUpdateComputer()
	}
}
