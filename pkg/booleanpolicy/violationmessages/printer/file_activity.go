package printer

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

func UpdateFileActivityAlertViolationMessage(v *storage.Alert_FileActivityViolation) {
	activity := v.GetActivity()
	if len(activity) == 0 {
		return
	}

	pathSet := set.NewStringSet()
	for _, fa := range activity {
		pathSet.Add(fa.GetFile().GetPath())
	}

	var sb strings.Builder

	if pathSet.Cardinality() < 10 {
		for i, fa := range activity {
			if i > 0 {
				sb.WriteString("; ")
			}
			fmt.Fprintf(&sb, "'%v' accessed (%v) by %v",
				fa.GetFile().GetPath(),
				fa.GetOperation(),
				fa.GetProcess().GetSignal().GetName())
		}
	} else {
		fmt.Fprintf(&sb, "%d sensitive files accessed", pathSet.Cardinality())
	}

	v.Message = sb.String()
}
