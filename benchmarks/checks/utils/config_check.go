package utils

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

// EvaluationFunc describes which function we want to apply when checking the configuration
type EvaluationFunc int

const (
	// Contains means that the commandline argument contains the desired value
	Contains EvaluationFunc = iota
	// NotContains means that the commandline argument does not contain the desired value
	NotContains
	// Matches means that the commandline argument matches the desired value exactly
	Matches
	// NotMatches means that the commandline argument does not match the desired value
	NotMatches
	// Unset means that the commandline argument is not set
	Unset
	// Set means that the commandline argument is set regardless of value
	Set
	// SetAsAppropriate means that there are many valid values
	SetAsAppropriate
	// Skip means that the check cannot be verified
	Skip
)

// CommandCheck is a benchmark check that applies to command line arguments
type CommandCheck struct {
	Name        string
	Description string
	Process     string

	Field        string
	Default      string
	EvalFunc     EvaluationFunc
	DesiredValue string

	ConfigGetter func() (FlattenedConfig, error)
}

func contains(c ConfigParams, value string) bool {
	_, found := c.Contains(value)
	return found
}

func matches(c ConfigParams, value string) bool {
	return c.Matches(value)
}

// Definition returns the definition of the check
func (c *CommandCheck) Definition() Definition {
	return Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        c.Name,
			Description: c.Description,
		},
	}
}

func (c *CommandCheck) contains(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if contains(actualValue, c.DesiredValue) {
		Warn(result)
		c.note(result, actualValue.String(), exists, "contain")
	}
}

func (c *CommandCheck) notContains(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if contains(actualValue, c.DesiredValue) {
		Warn(result)
		c.note(result, actualValue.String(), exists, "not contain")
	}
}

func (c *CommandCheck) matches(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if !matches(actualValue, c.DesiredValue) {
		Warn(result)
		c.note(result, actualValue.String(), exists, "be set to")
	}
}

func (c *CommandCheck) notMatches(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if !matches(actualValue, c.DesiredValue) {
		Warn(result)
		c.note(result, actualValue.String(), exists, "not be set to")
	}
}

func (c *CommandCheck) set(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if !exists {
		Warn(result)
		AddNotef(result, "%v is not set for %v (default '%v')", c.Field, c.Process, c.Default)
	}
}

func (c *CommandCheck) unset(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if exists {
		Warn(result)
		AddNotef(result, "%v is set to '%v' for %v, but should not be set", c.Field, c.Process, actualValue.String())
	}
}

func (c *CommandCheck) setAsAppropriate(result *storage.BenchmarkCheckResult, actualValue ConfigParams, exists bool) {
	Pass(result)
	if exists {
		Info(result)
	} else {
		Warn(result)
	}
	c.note(result, actualValue.String(), exists, "be set as appropriate")
}

func (c *CommandCheck) note(result *storage.BenchmarkCheckResult, actualValue string, exists bool, verb string) {
	defaultFmt := "no default"
	if len(c.Default) != 0 {
		defaultFmt = fmt.Sprintf("default '%v'", c.Default)
	}
	if exists {
		AddNotef(result, "%v is set to '%v' for %v. Should %v %v (%v)", c.Field, actualValue, c.Process, verb, c.DesiredValue, defaultFmt)
	} else {
		AddNotef(result, "%v is not set for %v. Should %v %v (%v)", c.Field, c.Process, verb, c.DesiredValue, defaultFmt)
	}
}

// Run evaluates the check
func (c *CommandCheck) Run() (result storage.BenchmarkCheckResult) {
	if c.EvalFunc == Skip {
		Note(&result)
		AddNotes(&result, c.Description)
		return
	}
	config, err := c.ConfigGetter()
	if err != nil {
		NotApplicable(&result)
		AddNotef(&result, "Could not retrieve config for %v. It may not be applicable to this node", c.Process)
		return
	}
	params, exists := config[c.Field]
	if !exists {
		params = ConfigParams([]string{c.Default})
	}
	switch c.EvalFunc {
	case Contains:
		c.contains(&result, params, exists)
	case NotContains:
		c.notContains(&result, params, exists)
	case Matches:
		c.matches(&result, params, exists)
	case NotMatches:
		c.notMatches(&result, params, exists)
	case Set:
		c.set(&result, params, exists)
	case Unset:
		c.unset(&result, params, exists)
	case SetAsAppropriate:
		c.setAsAppropriate(&result, params, exists)
	default:
		Warn(&result)
		AddNotef(&result, "Error in implementation of check. There is no implementation of evaluation function %v", c.EvalFunc)
	}
	return
}

// MultipleCommandChecks is a single check that checks many parameters
type MultipleCommandChecks struct {
	Name        string
	Description string
	Process     string

	Checks []CommandCheck

	ConfigGetter func() (FlattenedConfig, error)
}

// Definition returns the definition of the check
func (c *MultipleCommandChecks) Definition() Definition {
	return Definition{
		BenchmarkCheckDefinition: storage.BenchmarkCheckDefinition{
			Name:        c.Name,
			Description: c.Description,
		},
	}
}

// Run evaluates the check
func (c *MultipleCommandChecks) Run() (result storage.BenchmarkCheckResult) {
	Pass(&result)
	for _, check := range c.Checks {
		check.ConfigGetter = c.ConfigGetter
		check.Description = c.Description
		check.Process = c.Process
		res := check.Run()
		if res.Result != storage.BenchmarkCheckStatus_PASS {
			result.Result = res.Result
		}
		AddNotes(&result, res.GetNotes()...)
	}
	return
}
