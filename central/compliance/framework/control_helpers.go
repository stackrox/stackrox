package framework

import "fmt"

// RecordEvidence records evidence for the compliance check active in the given context.
func RecordEvidence(ctx ComplianceContext, status Status, msg string) {
	ctx.RecordEvidence(status, msg)
}

// RecordEvidencef records evidence for the compliance check active in the given context.
func RecordEvidencef(ctx ComplianceContext, status Status, format string, args ...interface{}) {
	ctx.RecordEvidence(status, fmt.Sprintf(format, args...))
}

// Pass records "pass" evidence for the compliance check active in the given context.
func Pass(ctx ComplianceContext, msg string) {
	RecordEvidence(ctx, PassStatus, msg)
}

// Passf records "pass" evidence for the compliance check active in the given context.
func Passf(ctx ComplianceContext, format string, args ...interface{}) {
	Pass(ctx, fmt.Sprintf(format, args...))
}

// PassNow records "pass" evidence for the compliance check active in the given context, and terminates the check.
func PassNow(ctx ComplianceContext, msg string) {
	Pass(ctx, msg)
	Abort(ctx, nil)
}

// PassNowf records "pass" evidence for the compliance check active in the given context, and terminates the check.
func PassNowf(ctx ComplianceContext, format string, args ...interface{}) {
	Passf(ctx, format, args...)
	Abort(ctx, nil)
}

// Fail records "fail" evidence for the compliance check active in the given context.
func Fail(ctx ComplianceContext, msg string) {
	RecordEvidence(ctx, FailStatus, msg)
}

// Failf records "fail" evidence for the compliance check active in the given context.
func Failf(ctx ComplianceContext, format string, args ...interface{}) {
	Fail(ctx, fmt.Sprintf(format, args...))
}

// FailNow records "fail" evidence for the compliance check active in the given context, and terminates the check.
func FailNow(ctx ComplianceContext, msg string) {
	Fail(ctx, msg)
	Abort(ctx, nil)
}

// FailNowf records "fail" evidence for the compliance check active in the given context, and terminates the check.
func FailNowf(ctx ComplianceContext, format string, args ...interface{}) {
	Failf(ctx, format, args...)
	Abort(ctx, nil)
}

// Skip records "skip" evidence for the compliance check active in the given context.
func Skip(ctx ComplianceContext, msg string) {
	RecordEvidence(ctx, SkipStatus, msg)
}

// Skipf records "skip" evidence for the compliance check active in the given context.
func Skipf(ctx ComplianceContext, format string, args ...interface{}) {
	Skip(ctx, fmt.Sprintf(format, args...))
}

// SkipNow records "skip" evidence for the compliance check active in the given context, and terminates the check.
func SkipNow(ctx ComplianceContext, msg string) {
	Skip(ctx, msg)
	Abort(ctx, nil)
}

// SkipNowf records "skip" evidence for the compliance check active in the given context, and terminates the check.
func SkipNowf(ctx ComplianceContext, format string, args ...interface{}) {
	Skipf(ctx, format, args...)
	Abort(ctx, nil)
}

// Abortf aborts with an error created from the given format and args via `fmt.Errorf`.
func Abortf(ctx ComplianceContext, format string, args ...interface{}) {
	Abort(ctx, fmt.Errorf(format, args...))
}

// Note records "note" evidence for the compliance check active in the given context.
func Note(ctx ComplianceContext, msg string) {
	RecordEvidence(ctx, NoteStatus, msg)
}

// Notef records "note" evidence for the compliance check active in the given context.
func Notef(ctx ComplianceContext, format string, args ...interface{}) {
	Note(ctx, fmt.Sprintf(format, args...))
}

// NoteNow records "note" evidence for the compliance check active in the given context, and terminates the check.
func NoteNow(ctx ComplianceContext, msg string) {
	Note(ctx, msg)
	Abort(ctx, nil)
}

// NoteNowf records "note" evidence for the compliance check active in the given context, and terminates the check.
func NoteNowf(ctx ComplianceContext, format string, args ...interface{}) {
	Notef(ctx, format, args...)
	Abort(ctx, nil)
}
