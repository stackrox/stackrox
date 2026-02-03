package printer

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

const (
	UNKNOWN_FILE = "Unknown file"
)

var (
	operationToPretty = map[storage.FileAccess_Operation]string{
		storage.FileAccess_OPEN:              "opened writable",
		storage.FileAccess_UNLINK:            "deleted",
		storage.FileAccess_CREATE:            "created",
		storage.FileAccess_OWNERSHIP_CHANGE:  "ownership changed",
		storage.FileAccess_PERMISSION_CHANGE: "permission changed",
		storage.FileAccess_RENAME:            "renamed",
	}
)

func prettifyOperation(op storage.FileAccess_Operation) string {
	if pretty, ok := operationToPretty[op]; ok {
		return pretty
	}
	return "Unknown operation"
}

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

	v.Message = fmt.Sprintf("'%v' %s", path, prettifyOperation(access.GetOperation()))
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
