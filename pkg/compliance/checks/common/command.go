package common

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/assert"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/compliance/msgfmt"
	pkgSet "github.com/stackrox/rox/pkg/set"
)

// KubeAPIProcessName is the string name of the kubernetes API server process
const KubeAPIProcessName = "kube-apiserver"

// FailOverride is passed as an option and will override the fail values if set
type FailOverride func(msg string) []*storage.ComplianceResultValue_Evidence

// CommandEvaluationFunc is a generic function that checks command lines
type CommandEvaluationFunc func([]string, string, string, string, ...FailOverride) []*storage.ComplianceResultValue_Evidence
type helperEvaluationFunc func([]string, string, string, string) (message string, passes bool)

// Info returns info with values set for the flag. Info is used when there is no strict determination of if the check is met
func Info(values []string, key, _, defaultVal string, _ ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	if len(values) == 0 {
		return NoteListf("%q is to the default value of %q", key, defaultVal)
	}
	return NoteListf("%q is set to %q", key, msgfmt.FormatStrings(values...))
}

// Set checks whether or not a value is set in the command line
func Set(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, valuesAreSet, overrides)
}

func getFailOverride(overrides []FailOverride) FailOverride {
	if len(overrides) == 0 {
		return nil
	}
	if len(overrides) > 1 {
		assert.Panicf("fail overrides can only have one element, but has %d", len(overrides))
	}
	return overrides[0]
}

// GetProcess returns the commandline object that matches the process name
func GetProcess(ret *standards.ComplianceData, processName string) (*compliance.CommandLine, bool) {
	for _, c := range ret.CommandLines {
		if strings.Contains(c.Process, processName) {
			return c, true
		}
	}
	return nil, false
}

// GetValuesForCommandFromFlagsAndConfig returns the values for specific key from the args and unmarshalled config
func GetValuesForCommandFromFlagsAndConfig(args []*compliance.CommandLine_Args, c map[string]interface{}, key string) []string {
	values := GetValuesForFlag(args, key)

	var value interface{}
	value, ok := c[key]
	if !ok {
		value, ok = c[key+"s"]
	}
	if !ok {
		return values
	}
	switch obj := value.(type) {
	case string:
		values = append(values, obj)
	case []string:
		values = append(values, obj...)
	default:
		panic(fmt.Sprintf("Unsupported type: %T", obj))
	}
	return values
}

// GetValuesForFlag returns the values based on the key passes
func GetValuesForFlag(args []*compliance.CommandLine_Args, key string) []string {
	var values []string
	for _, a := range args {
		if a.Key == key {
			values = append(values, a.GetValues()...)
		}
	}
	return values
}

// GetArgForFlag returns the arg that matches the passed key
func GetArgForFlag(args []*compliance.CommandLine_Args, key string) *compliance.CommandLine_Args {
	for _, a := range args {
		if a.Key == key {
			return a
		}
	}
	return nil
}

// Unset checks whether or not a value is not set in the command line
func Unset(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, unset, overrides)
}

// Matches checks whether or not a value matches the target value exactly
func Matches(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, matches, overrides)
}

// OnlyContains checks whether or not a value contains only the target values (where target values are delimited by ",")
func OnlyContains(values []string, key, targets, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, targets, defaultVal, onlyContains, overrides)
}

// NotMatches checks where or not a value matches the target value exactly
func NotMatches(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, notMatches, overrides)
}

// Contains checks where or not a value contains the target value
func Contains(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, contains, overrides)
}

// NotContains checks where or not a value contains the target value
func NotContains(values []string, key, target, defaultVal string, overrides ...FailOverride) []*storage.ComplianceResultValue_Evidence {
	return resultWrapper(values, key, target, defaultVal, notContains, overrides)
}

func resultWrapper(values []string, key, target, defaultVal string, f helperEvaluationFunc, overrides []FailOverride) []*storage.ComplianceResultValue_Evidence {
	msg, pass := f(values, key, target, defaultVal)
	if pass {
		return PassList(msg)
	}
	if override := getFailOverride(overrides); override != nil {
		return override(msg)
	}
	return FailList(msg)
}

func valuesAreSet(values []string, key, _, _ string) (string, bool) {
	if len(values) > 0 {
		return fmt.Sprintf("%q is set to %s", key, msgfmt.FormatStrings(values...)), true
	}
	return fmt.Sprintf("%q is not set", key), false
}

func unset(values []string, key, _, _ string) (string, bool) {
	if len(values) == 0 {
		return fmt.Sprintf("%q is not set", key), true
	}
	return fmt.Sprintf("%q is set to %s", key, msgfmt.FormatStrings(values...)), false
}

func matches(values []string, key, target, defaultStr string) (string, bool) {
	var matchingValues []string
	var nonMatchingValues []string
	for _, v := range values {
		if strings.EqualFold(v, target) {
			matchingValues = append(matchingValues, v)
		} else {
			nonMatchingValues = append(nonMatchingValues, v)
		}
	}
	if len(matchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s", key, msgfmt.FormatStrings(matchingValues...)), true
	} else if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q is set to %q and not the target value of %q", key, msgfmt.FormatStrings(nonMatchingValues...), target), false
	} else if target == defaultStr {
		return fmt.Sprintf("%q has a default value that matches the target value of %q", key, defaultStr), true
	}
	return fmt.Sprintf("%q has a default value of %q that does not match the target value of %q", key, defaultStr, target), false
}

func notMatches(values []string, key, target, defaultStr string) (string, bool) {
	var matchingValues []string
	var nonMatchingValues []string
	for _, v := range values {
		if strings.EqualFold(v, target) {
			matchingValues = append(matchingValues, v)
		} else {
			nonMatchingValues = append(nonMatchingValues, v)
		}
	}
	if len(matchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s which matches %q", key, msgfmt.FormatStrings(matchingValues...), target), false
	} else if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s which does not match %q", key, msgfmt.FormatStrings(nonMatchingValues...), target), true
	} else if target == defaultStr {
		return fmt.Sprintf("%q has a default value that matches the target value of %q", key, defaultStr), false
	}
	return fmt.Sprintf("%q has a default value of %q that does not match the target value of %q", key, defaultStr, target), true
}

func onlyContains(values []string, key, targets, defaults string) (string, bool) {
	var matchingValues []string
	var nonMatchingValues []string

	targetSet := pkgSet.NewStringSet(strings.Split(targets, ",")...)
	for _, v := range values {
		if targetSet.Contains(v) {
			matchingValues = append(matchingValues, v)
		} else {
			nonMatchingValues = append(nonMatchingValues, v)
		}
	}

	if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s which contains values other than target values in %q", key, msgfmt.FormatStrings(nonMatchingValues...), targets), false
	} else if len(matchingValues) > 0 {
		numMatches := "some"
		if len(matchingValues) == targetSet.Cardinality() {
			numMatches = "all"
		}
		return fmt.Sprintf("%q is set to %s which contains %s target values in %q", key, msgfmt.FormatStrings(matchingValues...), numMatches, targets), true
	}

	defaultSet := pkgSet.NewStringSet(strings.Split(defaults, ";")...)
	for t := range targetSet {
		if defaultSet.Contains(t) {
			matchingValues = append(matchingValues, t)
		} else {
			nonMatchingValues = append(nonMatchingValues, t)
		}
	}

	if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q has a default values %q which contains values other than target values in %q", key, msgfmt.FormatStrings(nonMatchingValues...), targets), false
	}
	numMatches := "some"
	if len(matchingValues) == targetSet.Cardinality() {
		numMatches = "all"
	}
	return fmt.Sprintf("%q has a default values %q which contains %s target values in %q", key, msgfmt.FormatStrings(matchingValues...), numMatches, targets), true
}

func contains(values []string, key, target, defaultStr string) (string, bool) {
	var matchingValues []string
	var nonMatchingValues []string
	for _, v := range values {
		if strings.Contains(v, target) {
			matchingValues = append(matchingValues, v)
		} else {
			nonMatchingValues = append(nonMatchingValues, v)
		}
	}
	if len(matchingValues) > 0 {
		return fmt.Sprintf("%q contains %s", key, msgfmt.FormatStrings(matchingValues...)), true
	} else if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q is set to %q and does not contain the target value of %q", key, msgfmt.FormatStrings(nonMatchingValues...), target), false
	} else if strings.Contains(defaultStr, target) {
		return fmt.Sprintf("%q has a default value that contains the target value of %q", key, defaultStr), true
	} else {
		return fmt.Sprintf("%q has a default value of %q that does not contain the target value of %q", key, defaultStr, target), false
	}
}

func notContains(values []string, key, target, defaultStr string) (string, bool) {
	var matchingValues []string
	var nonMatchingValues []string
	for _, v := range values {
		if strings.Contains(v, target) {
			matchingValues = append(matchingValues, v)
		} else {
			nonMatchingValues = append(nonMatchingValues, v)
		}
	}

	if len(matchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s which contains %q", key, msgfmt.FormatStrings(matchingValues...), target), false
	} else if len(nonMatchingValues) > 0 {
		return fmt.Sprintf("%q is set to %s which does not contain %q", key, msgfmt.FormatStrings(nonMatchingValues...), target), true
	} else if !strings.Contains(defaultStr, target) {
		return fmt.Sprintf("%q does not contain %q", key, target), true
	} else {
		return fmt.Sprintf("%q has a default value of %q that contains %q", key, defaultStr, target), false
	}
}

// MasterNodeKubernetesCommandlineCheck checks the arguments of the given process if this is the Kubernetes master node
func MasterNodeKubernetesCommandlineCheck(processName, key, target, defaultVal string, evalFunc CommandEvaluationFunc, failOverride ...FailOverride) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			process, exists := GetProcess(complianceData, processName)
			if !exists {
				if complianceData.IsMasterNode {
					return NoteListf("Process %q not found on host, therefore check is not applicable", processName)
				}
				return nil
			}
			values := GetValuesForCommandFromFlagsAndConfig(process.Args, nil, key)
			return evalFunc(values, key, target, defaultVal, failOverride...)
		},
		Metadata: &standards.Metadata{
			TargetKind: framework.ClusterKind,
		},
	}
}

// MasterAPIServerCommandLine is a master node process command line check which hard-codes the kube API server process name
func MasterAPIServerCommandLine(key, target, defaultVal string, evalFunc CommandEvaluationFunc) *standards.CheckAndMetadata {
	return MasterNodeKubernetesCommandlineCheck(KubeAPIProcessName, key, target, defaultVal, evalFunc)
}
