package main

// Config represents the test selection rules configuration
type Config struct {
	Version  int               `yaml:"version"`
	Jobs     []string          `yaml:"jobs"`
	Defaults map[string]string `yaml:"defaults"`
	Rules    []Rule            `yaml:"rules"`
}

// Rule represents a single test selection rule
type Rule struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	When        Condition `yaml:"when"`
	Action      Action    `yaml:"action"`
}

// Condition defines when a rule applies
type Condition struct {
	Label            string   `yaml:"label,omitempty"`
	AnyFileMatches   []string `yaml:"any_file_matches,omitempty"`
	EveryFileMatches []string `yaml:"every_file_matches,omitempty"`
	AnyFileChanged   bool     `yaml:"any_file_changed,omitempty"`
}

// Action defines what jobs to run or skip
type Action struct {
	Run    interface{} `yaml:"run,omitempty"`    // Can be "all" or []string
	Skip   interface{} `yaml:"skip,omitempty"`   // Can be "all" or []string
	Except []string    `yaml:"except,omitempty"` // Exceptions to skip
}

// Vote represents a rule's decision about a job
type Vote string

const (
	VoteRun  Vote = "run"
	VoteSkip Vote = "skip"
)
