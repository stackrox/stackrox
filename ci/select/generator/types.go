package main

// Config represents the simplified job-centric test selection configuration
type Config struct {
	Version     int             `yaml:"version"`
	RunAllTests RunAllCondition `yaml:"run_all_tests"`
	Jobs        map[string]Job  `yaml:"jobs"`
}

// RunAllCondition defines global triggers that run all jobs
type RunAllCondition struct {
	When []WhenCondition `yaml:"when"`
}

// Job represents a single test job with its trigger conditions
type Job struct {
	When    []WhenCondition `yaml:"when"`
	WhenNot []WhenCondition `yaml:"when_not,omitempty"`
}

// WhenCondition defines a single trigger condition
// Exactly one field should be set: Always, Label, or Files
type WhenCondition struct {
	Always bool     `yaml:"always,omitempty"`
	Label  string   `yaml:"label,omitempty"`
	Files  []string `yaml:"files,omitempty"`
}
