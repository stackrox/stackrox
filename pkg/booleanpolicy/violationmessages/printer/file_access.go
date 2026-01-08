package printer

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

const (
	maxPaths = 10
)

func UpdateFileAccessAlertViolationMessage(v *storage.Alert_FileAccessViolation) {
	if len(v.GetAccesses()) == 0 {
		return
	}

	// Construct a string for each distinct path, and accumulate
	// file operation, so we can show all the kinds of activity
	// for each file, to provide the most important info.
	pathToOperation := make(map[string][]string, 0)
	for _, fa := range v.GetAccesses() {
		path := fa.GetFile().GetActualPath()
		pathToOperation[path] = append(pathToOperation[path], fa.GetOperation().String())
	}

	if len(pathToOperation) >= maxPaths {
		v.Message = fmt.Sprintf("%d sensitive files accessed", len(pathToOperation))
		return
	}

	// sorted to make this more deterministic which means both that
	// the output is more consistent, and it is more testable.
	paths := slices.SortedFunc(maps.Keys(pathToOperation), func(a, b string) int {
		return strings.Compare(a, b)
	})

	parts := make([]string, 0, len(pathToOperation))
	for _, path := range paths {
		parts = append(parts, fmt.Sprintf("'%v' accessed (%s)", path, strings.Join(sliceutils.Unique(pathToOperation[path]), ", ")))
	}

	v.Message = strings.Join(parts, "; ")
}
