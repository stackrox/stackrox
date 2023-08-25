package printer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// UpdateProcessAlertViolationMessage updates the violation message for a violation in-place
func UpdateProcessAlertViolationMessage(v *storage.Alert_ProcessViolation) {
	processes := v.GetProcesses()
	if len(processes) == 0 {
		return
	}

	pathSet := set.NewStringSet()
	argsSet := set.NewStringSet()
	uidSet := set.NewSet[int]()

	for _, process := range processes {
		pathSet.Add(process.GetSignal().GetExecFilePath())
		argsSet.Add(process.GetSignal().GetArgs())
		uidSet.Add(int(process.GetSignal().GetUid()))
	}

	var sb strings.Builder

	paths := pathSet.AsSlice()
	sort.Strings(paths)
	switch numPaths := pathSet.Cardinality(); {
	case numPaths == 1:
		fmt.Fprintf(&sb, "Binary '%s'", paths[0])
	case numPaths == 2:
		fmt.Fprintf(&sb, "Binaries '%s' and '%s'", paths[0], paths[1])
	case numPaths < 10:
		fmt.Fprint(&sb, "Binaries ")
		for idx, path := range paths {
			if idx < numPaths-1 {
				fmt.Fprintf(&sb, "'%s', ", path)
			} else {
				fmt.Fprintf(&sb, "and '%s'", path)
			}
		}
	default:
		fmt.Fprintf(&sb, "%d binaries", numPaths)
	}
	fmt.Fprint(&sb, " executed")

	numArgs := argsSet.Cardinality()
	if numArgs == 1 {
		arg := argsSet.GetArbitraryElem()
		if arg != "" {
			fmt.Fprintf(&sb, " with arguments '%s'", argsSet.GetArbitraryElem())
		} else {
			fmt.Fprint(&sb, " without arguments")
		}
	} else if numArgs > 0 {
		fmt.Fprintf(&sb, " with %d different arguments", numArgs)
	}

	numUIDs := uidSet.Cardinality()
	if numUIDs == 1 {
		fmt.Fprintf(&sb, " under user ID %d", uidSet.GetArbitraryElem())
	} else if numUIDs > 0 {
		fmt.Fprintf(&sb, " under %d different user IDs", uidSet.Cardinality())
	}

	v.Message = sb.String()
}
