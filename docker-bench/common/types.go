package common

import (
	"fmt"
)

const (
	// Info is neither a pass nor fail
	Info = "INFO"
	// Warn is a failure of the test
	Warn = "WARN"
	// Note means that the test requires manual intervention
	Note = "NOTE"
	// Pass means the test was successful
	Pass = "PASS"
)

// BenchmarkPayload is the payload that packages up all the results of the tests
type BenchmarkPayload struct {
	Results   []Result `json:"results"`
	StartTime int64    `json:"start_time"`
	EndTime   int64    `json:"end_time"`
	Host      string   `json:"host"`
}

// Result is the packaged result of a test with the benchmark definition
type Result struct {
	TestResult          TestResult `json:"test_result"`
	BenchmarkDefinition Definition `json:"benchmark_definitions"`
}

// Benchmark is the interface that all benchmarks must implement
type Benchmark interface {
	Definition() Definition
	Run() TestResult
}

// Dependency is the function stub that functions that act as dependencies must follow
type Dependency func() error

// Definition is the definition of a Benchmark
type Definition struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	Dependencies []Dependency `json:"-"`
}

// TestResult is the self-explanatory
type TestResult struct {
	Result string   `json:"result"`
	Notes  []string `json:"notes"`
}

// AddNotes takes in a variadic and appends them to the notes of the benchmark
func (result *TestResult) AddNotes(note ...string) {
	result.Notes = append(result.Notes, note...)
}

// AddNotef allows Sprintf style formatting when adding a note
func (result *TestResult) AddNotef(template string, args ...interface{}) {
	result.Notes = append(result.Notes, fmt.Sprintf(template, args...))
}

// Pass sets the test result to Pass
func (result *TestResult) Pass() {
	result.Result = Pass
}

// Info sets the test result to Info
func (result *TestResult) Info() {
	result.Result = Info
}

// Warn sets the test result to Warn
func (result *TestResult) Warn() {
	result.Result = Warn
}

// Note sets the test result to Note
func (result *TestResult) Note() {
	result.Result = Note
}
