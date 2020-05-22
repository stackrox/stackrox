package violationmessages

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

func dropCapabilityPrinter(fieldMap map[string][]string) ([]string, error) {
	if lenContainers := len(fieldMap[augmentedobjs.ContainerNameCustomTag]); lenContainers != 1 {
		return nil, errors.Errorf("unexpected number of container names: %d", lenContainers)
	}
	var sb strings.Builder
	sb.WriteString("Container ")
	if containerName := fieldMap[augmentedobjs.ContainerNameCustomTag][0]; containerName != "" {
		fmt.Fprintf(&sb, "%s ", containerName)
	}
	sb.WriteString("does not drop ")
	switch capLen := len(fieldMap[search.DropCapabilities.String()]); {
	case capLen == 1:
		sb.WriteString("capability ")
	case capLen > 1:
		sb.WriteString("capabilities ")
	default:
		return nil, errors.New("Missing capabilities")
	}
	sb.WriteString(stringSliceToSortedSentence(fieldMap[search.DropCapabilities.String()]))
	return []string{sb.String()}, nil
}

func addCapabilityPrinter(fieldMap map[string][]string) ([]string, error) {
	if lenContainers := len(fieldMap[augmentedobjs.ContainerNameCustomTag]); lenContainers != 1 {
		return nil, errors.Errorf("unexpected number of container names: %d", lenContainers)
	}
	var sb strings.Builder
	sb.WriteString("Container ")
	if containerName := fieldMap[augmentedobjs.ContainerNameCustomTag][0]; containerName != "" {
		fmt.Fprintf(&sb, "%s ", containerName)
	}
	sb.WriteString("adds ")
	switch capLen := len(fieldMap[search.AddCapabilities.String()]); {
	case capLen == 1:
		sb.WriteString("capability ")
	case capLen > 1:
		sb.WriteString("capabilities ")
	default:
		return nil, errors.New("Missing capabilities")
	}
	sb.WriteString(stringSliceToSortedSentence(fieldMap[search.AddCapabilities.String()]))
	return []string{sb.String()}, nil
}
