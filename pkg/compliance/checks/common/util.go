package common

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
)

// Failf returns "fail" evidence from the given template.
func Failf(msg string, args ...interface{}) *storage.ComplianceResultValue_Evidence {
	return Fail(fmt.Sprintf(msg, args...))
}

// Fail returns "fail" evidence using the given string
func Fail(msg string) *storage.ComplianceResultValue_Evidence {
	return result(storage.ComplianceState_COMPLIANCE_STATE_FAILURE, msg)
}

// FailListf returns a single-element list of "fail" evidence from the given template as a convenience
func FailListf(msg string, args ...interface{}) []*storage.ComplianceResultValue_Evidence {
	return FailList(fmt.Sprintf(msg, args...))
}

// FailList returns a single-element list of "fail" evidence from the given string as a convenience
func FailList(msg string) []*storage.ComplianceResultValue_Evidence {
	return []*storage.ComplianceResultValue_Evidence{Fail(msg)}
}

// Passf returns "pass" evidence from the given template.
func Passf(msg string, args ...interface{}) *storage.ComplianceResultValue_Evidence {
	return Pass(fmt.Sprintf(msg, args...))
}

// Pass returns "pass" evidence using the given string
func Pass(msg string) *storage.ComplianceResultValue_Evidence {
	return result(storage.ComplianceState_COMPLIANCE_STATE_SUCCESS, msg)
}

// PassListf returns a single-element list of "pass" evidence from the given template as a convenience
func PassListf(msg string, args ...interface{}) []*storage.ComplianceResultValue_Evidence {
	return PassList(fmt.Sprintf(msg, args...))
}

// PassList returns a single-element list of "pass" evidence from the given string as a convenience
func PassList(msg string) []*storage.ComplianceResultValue_Evidence {
	return []*storage.ComplianceResultValue_Evidence{Pass(msg)}
}

// Notef returns "note" evidence from the given template.
func Notef(msg string, args ...interface{}) *storage.ComplianceResultValue_Evidence {
	return Note(fmt.Sprintf(msg, args...))
}

// Note returns "note" evidence using the given string
func Note(msg string) *storage.ComplianceResultValue_Evidence {
	return result(storage.ComplianceState_COMPLIANCE_STATE_NOTE, msg)
}

// NoteListf returns a single-element list of "note" evidence from the given template as a convenience
func NoteListf(msg string, args ...interface{}) []*storage.ComplianceResultValue_Evidence {
	return NoteList(fmt.Sprintf(msg, args...))
}

// NoteList returns a single-element list of "note" evidence from the given string as a convenience
func NoteList(msg string) []*storage.ComplianceResultValue_Evidence {
	return []*storage.ComplianceResultValue_Evidence{Note(msg)}
}

// SkipList returns a single-element list of "skip" evidence from the given template as a convenience
func SkipList(msg string) []*storage.ComplianceResultValue_Evidence {
	return []*storage.ComplianceResultValue_Evidence{result(storage.ComplianceState_COMPLIANCE_STATE_SKIP, msg)}
}

func result(status storage.ComplianceState, msg string) *storage.ComplianceResultValue_Evidence {
	return &storage.ComplianceResultValue_Evidence{
		State:   status,
		Message: msg,
	}
}
