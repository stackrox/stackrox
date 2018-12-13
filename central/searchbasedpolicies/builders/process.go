package builders

import (
	"fmt"
	"sort"

	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// ProcessQueryBuilder builds queries for process name field.
type ProcessQueryBuilder struct {
}

// Query implements the PolicyQueryBuilder interface.
func (p ProcessQueryBuilder) Query(fields *storage.PolicyFields, optionsMap map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
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

	v = func(result search.Result, processGetter searchbasedpolicies.ProcessIndicatorGetter) []*storage.Alert_Violation {
		matches := result.Matches[processIDSearchField.GetFieldPath()]
		if len(result.Matches[processIDSearchField.GetFieldPath()]) == 0 {
			logger.Errorf("ID %s matched process query, but couldn't find the matching id", result.ID)
			return nil
		}
		if processGetter == nil {
			logger.Errorf("Ran process policy %+v but had a nil process getter.", fields)
			return nil
		}
		processes := make([]*storage.ProcessIndicator, 0, len(matches))
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

		v := &storage.Alert_Violation{Processes: processes}
		UpdateRuntimeAlertViolationMessage(v)
		return []*storage.Alert_Violation{v}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (p ProcessQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for process policy")
}

// UpdateRuntimeAlertViolationMessage updates the violation message for a violation in-place
func UpdateRuntimeAlertViolationMessage(v *storage.Alert_Violation) {
	processes := v.GetProcesses()
	if len(processes) == 0 {
		return
	}

	pathSet := set.NewStringSet()
	argsSet := set.NewStringSet()
	for _, process := range processes {
		pathSet.Add(process.GetSignal().GetExecFilePath())
		if process.GetSignal().GetArgs() != "" {
			argsSet.Add(process.GetSignal().GetArgs())
		}
	}

	var countMessage, argsMessage, pathMessage string
	if len(processes) == 1 {
		countMessage = "execution of"
	} else {
		countMessage = "executions of"
	}

	if pathSet.Cardinality() == 1 {
		pathMessage = fmt.Sprintf(" binary '%s'", processes[0].GetSignal().GetExecFilePath())
	} else if pathSet.Cardinality() > 0 {
		pathMessage = fmt.Sprintf(" %d binaries", pathSet.Cardinality())
	}

	if argsSet.Cardinality() == 1 {
		argsMessage = fmt.Sprintf(" with arguments '%s'", processes[0].GetSignal().GetArgs())
	} else if argsSet.Cardinality() > 0 {
		argsMessage = fmt.Sprintf(" with %d different arguments", argsSet.Cardinality())
	}

	v.Message = fmt.Sprintf("Detected %s%s%s", countMessage, pathMessage, argsMessage)
}
