package printer

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	UNKNOWN_FILE = "Unknown file"
)

func UpdateFileAccessAlertViolationMessage(v *storage.Alert_Violation) {
	if v.GetType() != storage.Alert_Violation_FILE_ACCESS {
		return
	}

	access := v.GetFileAccess()
	if access == nil {
		return
	}

	path := UNKNOWN_FILE
	if access.GetFile().GetActualPath() != "" {
		path = access.GetFile().GetActualPath()
	} else if access.GetFile().GetEffectivePath() != "" {
		path = access.GetFile().GetEffectivePath()
	}

	v.Message = fmt.Sprintf("'%v' accessed (%s)", path, access.GetOperation())
}

func GenerateFileAccessViolation(access *storage.FileAccess) *storage.Alert_Violation {
	violation := &storage.Alert_Violation{
		Type: storage.Alert_Violation_FILE_ACCESS,
		MessageAttributes: &storage.Alert_Violation_FileAccess{
			FileAccess: access,
		},
	}

	UpdateFileAccessAlertViolationMessage(violation)
	return violation
}
