package utils

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// Definition wraps the BenchmarkDefinition Proto and the Dependencies thare functions and cannot be serialized
type Definition struct {
	v1.BenchmarkDefinition
	Dependencies []Dependency `json:"-"`
}

// Benchmark is the interface that all benchmarks must implement
type Benchmark interface {
	Definition() Definition
	Run() v1.BenchmarkTestResult
}

// Dependency is the function stub that functions that act as dependencies must follow
type Dependency func() error

// AddNotes takes in a variadic and appends them to the notes of the benchmark
func AddNotes(result *v1.BenchmarkTestResult, notes ...string) {
	result.Notes = append(result.Notes, notes...)
}

// AddNotef allows Sprintf style formatting when adding a note
func AddNotef(result *v1.BenchmarkTestResult, template string, args ...interface{}) {
	result.Notes = append(result.Notes, fmt.Sprintf(template, args...))
}

// Pass sets the test result to Pass
func Pass(result *v1.BenchmarkTestResult) {
	result.Result = v1.BenchmarkStatus_PASS
}

// Info sets the test result to Info
func Info(result *v1.BenchmarkTestResult) {
	result.Result = v1.BenchmarkStatus_INFO
}

// Warn sets the test result to Warn
func Warn(result *v1.BenchmarkTestResult) {
	result.Result = v1.BenchmarkStatus_WARN
}

// Note sets the test result to Note
func Note(result *v1.BenchmarkTestResult) {
	result.Result = v1.BenchmarkStatus_NOTE
}
