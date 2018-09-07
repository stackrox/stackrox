package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// Definition wraps the CheckDefinition Proto and the Dependencies thare functions and cannot be serialized
type Definition struct {
	v1.CheckDefinition
	Dependencies []Dependency `json:"-"`
}

// Check is the interface that all benchmarks must implement
type Check interface {
	Definition() Definition
	Run() v1.CheckResult
}

// Dependency is the function stub that functions that act as dependencies must follow
type Dependency func() error

// AddNotes takes in a variadic and appends them to the notes of the benchmark
func AddNotes(result *v1.CheckResult, notes ...string) {
	result.Notes = append(result.Notes, notes...)
}

// AddNotef allows Sprintf style formatting when adding a note
func AddNotef(result *v1.CheckResult, template string, args ...interface{}) {
	result.Notes = append(result.Notes, fmt.Sprintf(template, args...))
}

// Pass sets the test result to Pass
func Pass(result *v1.CheckResult) {
	result.Result = v1.CheckStatus_PASS
}

// Info sets the test result to Info
func Info(result *v1.CheckResult) {
	result.Result = v1.CheckStatus_INFO
}

// Warn sets the test result to Warn
func Warn(result *v1.CheckResult) {
	result.Result = v1.CheckStatus_WARN
}

// Note sets the test result to Note
func Note(result *v1.CheckResult) {
	result.Result = v1.CheckStatus_NOTE
}

// NotApplicable sets the test result to NotApplicable and should be used when the check applies to a specific type of node
func NotApplicable(result *v1.CheckResult) {
	result.Result = v1.CheckStatus_NOT_APPLICABLE
}
