package builders

import (
	"fmt"
	"sort"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
)

// ProcessQueryBuilder builds queries for process name field.
type ProcessQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (p ProcessQueryBuilder) Query(fields *v1.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	processName := fields.GetProcessPolicy().GetName()
	processArgs := fields.GetProcessPolicy().GetArgs()
	if processName == "" && processArgs == "" {
		return
	}

	_, err = getSearchFieldNotStored(search.ProcessName, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", p.Name(), err)
		return
	}
	_, err = getSearchFieldNotStored(search.ProcessArguments, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", p.Name(), search.ProcessArguments)
	}
	processIDSearchField, err := getSearchFieldNotStored(search.ProcessID, optionsMap)
	if err != nil {
		err = fmt.Errorf("%s: %s", p.Name(), err)
	}

	// Construct query for ProcessID and processArgs (if found)
	fieldLabels := []search.FieldLabel{search.ProcessID}
	queryStrings := []string{search.WildcardString}
	highlights := []bool{true}

	if processArgs != "" {
		fieldLabels = append(fieldLabels, search.ProcessArguments)
		queryStrings = append(queryStrings, search.RegexQueryString(processArgs))
		highlights = append(highlights, false)
	}

	if processName != "" {
		fieldLabels = append(fieldLabels, search.ProcessName)
		queryStrings = append(queryStrings, search.RegexQueryString(processName))
		highlights = append(highlights, false)
	}

	q = search.NewQueryBuilder().AddLinkedFieldsWithHighlightValues(fieldLabels, queryStrings, highlights).ProtoQuery()

	v = func(result search.Result, processGetter searchbasedpolicies.ProcessIndicatorGetter) []*v1.Alert_Violation {
		matches := result.Matches[processIDSearchField.GetFieldPath()]
		if len(result.Matches[processIDSearchField.GetFieldPath()]) == 0 {
			logger.Errorf("ID %s matched process query, but couldn't find the matching id", result.ID)
			return nil
		}
		if processGetter == nil {
			logger.Errorf("Ran process policy %+v but had a nil process getter.", fields)
			return nil
		}
		processes := make([]*v1.ProcessIndicator, 0, len(matches))
		for _, processID := range matches {
			process, exists, err := processGetter.GetProcessIndicator(processID)
			if err != nil {
				logger.Errorf("Error retrieving process with id %s from store", processID)
				continue
			}
			if !exists { // Likely a race condition
				continue
			}
			processes = append(processes, process)
		}
		if len(processes) == 0 {
			return nil
		}
		sort.Slice(processes, func(i, j int) bool {
			return protoconv.CompareProtoTimestamps(processes[i].GetSignal().GetTime(), processes[j].GetSignal().GetTime()) < 0
		})

		v := &v1.Alert_Violation{Processes: processes}
		UpdateRuntimeAlertViolationMessage(v)
		return []*v1.Alert_Violation{v}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (p ProcessQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for process policy")
}

// UpdateRuntimeAlertViolationMessage updates the violation message for a violation in-place
func UpdateRuntimeAlertViolationMessage(v *v1.Alert_Violation) {
	processes := v.GetProcesses()
	if len(processes) == 0 {
		return
	}

	pathSet := make(map[string]struct{})
	argsSet := make(map[string]struct{})
	for _, process := range processes {
		pathSet[process.GetSignal().GetExecFilePath()] = struct{}{}
		if process.GetSignal().GetArgs() != "" {
			argsSet[process.GetSignal().GetArgs()] = struct{}{}
		}
	}

	var countMessage, argsMessage, pathMessage string
	if len(processes) == 1 {
		countMessage = "execution of"
	} else {
		countMessage = "executions of"
	}

	if len(pathSet) == 1 {
		pathMessage = fmt.Sprintf(" binary '%s'", processes[0].GetSignal().GetExecFilePath())
	} else if len(pathSet) > 0 {
		pathMessage = fmt.Sprintf(" %d binaries", len(pathSet))
	}

	if len(argsSet) == 1 {
		argsMessage = fmt.Sprintf(" with arguments '%s'", processes[0].GetSignal().GetArgs())
	} else if len(argsSet) > 0 {
		argsMessage = fmt.Sprintf(" with %d different arguments", len(argsSet))
	}

	v.Message = fmt.Sprintf("Detected %s%s%s", countMessage, pathMessage, argsMessage)
}
