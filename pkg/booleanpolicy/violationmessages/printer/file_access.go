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

	path := access.GetFile().GetActualPath()
	if path == "" {
		path = access.GetFile().GetEffectivePath()
		if path == "" {
			path = UNKNOWN_FILE
		}
	}
	operation := access.GetOperation()

	v.Message = fmt.Sprintf("'%v' accessed (%s)", path, operation)
}

func GenerateFileAccessViolation(access *storage.FileAccess) (*storage.Alert_Violation, error) {
	violation := &storage.Alert_Violation{
		Type: storage.Alert_Violation_FILE_ACCESS,
		MessageAttributes: &storage.Alert_Violation_FileAccess{
			FileAccess: access,
		},
	}

	UpdateFileAccessAlertViolationMessage(violation)
	return violation, nil
}
