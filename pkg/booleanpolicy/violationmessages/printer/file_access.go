package printer

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

func UpdateFileAccessAlertViolationMessage(v *storage.Alert_FileAccessViolation) {
	accesses := v.GetAccesses()
	if len(accesses) == 0 {
		return
	}

	pathSet := set.NewStringSet()
	for _, fa := range accesses {
		pathSet.Add(fa.GetFile().GetNodePath())
	}

	var sb strings.Builder

	if pathSet.Cardinality() < 10 {
		// Construct a string for each distinct path, and accumulate
		// file operation, so we can show all the kinds of activity
		// for each file, to provide the most important info.

		pathToOperation := make(map[string][]string, pathSet.Cardinality())
		for _, fa := range accesses {
			path := fa.GetFile().GetNodePath()
			pathToOperation[path] = append(pathToOperation[path], fa.GetOperation().String())
		}

		parts := make([]string, 0, pathSet.Cardinality())

		// sorted to make this more deterministic which means both that
		// the output is more consistent, and it is more testable.
		paths := pathSet.AsSortedSlice(func(a, b string) bool {
			return strings.Compare(a, b) < 0
		})

		for _, path := range paths {
			parts = append(parts, fmt.Sprintf("'%v' accessed (%s)", path, strings.Join(pathToOperation[path], ", ")))
		}

		fmt.Fprintf(&sb, "%s", strings.Join(parts, "; "))
	} else {
		fmt.Fprintf(&sb, "%d sensitive files accessed", pathSet.Cardinality())
	}

	v.Message = sb.String()
}
