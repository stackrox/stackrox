package builders

import (
	"fmt"
	"sort"
	"strings"

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

	fieldLabels := []search.FieldLabel{search.ProcessID}
	queries := []string{search.WildcardString}
	highlights := []bool{true}

	if processName != "" {
		fieldLabels = append(fieldLabels, search.ProcessName)
		queries = append(queries, search.RegexQueryString(processName))
		highlights = append(highlights, false)
	}
	if processArgs != "" {
		fieldLabels = append(fieldLabels, search.ProcessArguments)
		queries = append(queries, search.RegexQueryString(processArgs))
		highlights = append(highlights, false)
	}

	q = search.NewQueryBuilder().AddLinkedFieldsWithHighlightValues(
		fieldLabels, queries, highlights).ProtoQuery()

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
		var messageBuilder strings.Builder
		messageBuilder.WriteString("Found ")
		if len(processes) == 1 {
			messageBuilder.WriteString("process ")
		} else {
			messageBuilder.WriteString("processes ")
		}
		messageBuilder.WriteString("with ")
		if processName != "" {
			messageBuilder.WriteString(fmt.Sprintf("name matching '%s'", processName))
			if processArgs != "" {
				messageBuilder.WriteString(" and ")
			}
		}
		if processArgs != "" {
			messageBuilder.WriteString(fmt.Sprintf("args matching '%s'", processArgs))
		}
		return []*v1.Alert_Violation{{Message: messageBuilder.String(), Processes: processes}}
	}
	return
}

// Name implements the PolicyQueryBuilder interface.
func (p ProcessQueryBuilder) Name() string {
	return fmt.Sprintf("query builder for process policy")
}
