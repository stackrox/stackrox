package printer

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func dropCapabilityPrinter(fieldMap map[string][]string) ([]string, error) {
	if lenContainers := len(fieldMap[augmentedobjs.ContainerNameCustomTag]); lenContainers != 1 {
		return nil, errors.Errorf("unexpected number of container names: %d", lenContainers)
	}
	var sb strings.Builder
	sb.WriteString("Container ")
	if containerName := fieldMap[augmentedobjs.ContainerNameCustomTag][0]; containerName != "" {
		fmt.Fprintf(&sb, "'%s' ", containerName)
	}
	sb.WriteString("does not drop expected capabilities")
	dropped := fieldMap[search.DropCapabilities.String()]
	if len(dropped) == 0 {
		utils.Should(errors.New("found no values in dropped capabilities"))
		return []string{sb.String()}, nil
	}
	if len(dropped) == 1 && dropped[0] == "<empty>" {
		sb.WriteString(" (drops no capabilities)")
	} else {
		sb.WriteString(" (drops ")
		sb.WriteString(stringSliceToSortedSentence(fieldMap[search.DropCapabilities.String()]))
		sb.WriteString(")")
	}
	return []string{sb.String()}, nil
}

func addCapabilityPrinter(fieldMap map[string][]string) ([]string, error) {
	if lenContainers := len(fieldMap[augmentedobjs.ContainerNameCustomTag]); lenContainers != 1 {
		return nil, errors.Errorf("unexpected number of container names: %d", lenContainers)
	}
	var sb strings.Builder
	sb.WriteString("Container ")
	if containerName := fieldMap[augmentedobjs.ContainerNameCustomTag][0]; containerName != "" {
		fmt.Fprintf(&sb, "'%s' ", containerName)
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
