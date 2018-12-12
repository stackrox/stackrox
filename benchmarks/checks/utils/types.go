package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// Definition wraps the BenchmarkCheckDefinition Proto and the Dependencies thare functions and cannot be serialized
type Definition struct {
	storage.BenchmarkCheckDefinition
	Dependencies []Dependency `json:"-"`
}

// Check is the interface that all benchmarks must implement
type Check interface {
	Definition() Definition
	Run() storage.BenchmarkCheckResult
}

// Dependency is the function stub that functions that act as dependencies must follow
type Dependency func() error

// AddNotes takes in a variadic and appends them to the notes of the benchmark
func AddNotes(result *storage.BenchmarkCheckResult, notes ...string) {
	result.Notes = append(result.Notes, notes...)
}

// AddNotef allows Sprintf style formatting when adding a note
func AddNotef(result *storage.BenchmarkCheckResult, template string, args ...interface{}) {
	result.Notes = append(result.Notes, fmt.Sprintf(template, args...))
}

// Pass sets the test result to Pass
func Pass(result *storage.BenchmarkCheckResult) {
	result.Result = storage.BenchmarkCheckStatus_PASS
}

// Info sets the test result to Info
func Info(result *storage.BenchmarkCheckResult) {
	result.Result = storage.BenchmarkCheckStatus_INFO
}

// Warn sets the test result to Warn
func Warn(result *storage.BenchmarkCheckResult) {
	result.Result = storage.BenchmarkCheckStatus_WARN
}

// Note sets the test result to Note
func Note(result *storage.BenchmarkCheckResult) {
	result.Result = storage.BenchmarkCheckStatus_NOTE
}

// NotApplicable sets the test result to NotApplicable and should be used when the check applies to a specific type of node
func NotApplicable(result *storage.BenchmarkCheckResult) {
	result.Result = storage.BenchmarkCheckStatus_NOT_APPLICABLE
}
