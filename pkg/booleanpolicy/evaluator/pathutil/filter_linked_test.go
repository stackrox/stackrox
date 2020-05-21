package pathutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type simplePathHolder Path

func (p *simplePathHolder) GetPath() *Path {
	return (*Path)(p)
}

func (p *simplePathHolder) GetValues() []string {
	return []string{fmt.Sprintf("Val%v", p.GetPath())}
}

func pathAndValuesFromSteps(t *testing.T, steps ...interface{}) PathAndValueHolder {
	return (*simplePathHolder)(PathFromSteps(t, steps...))
}

func filteredResultWithSingleMatch(fieldName string, pathAndValues PathAndValueHolder) []map[string][]string {
	return []map[string][]string{{fieldName: pathAndValues.GetValues()}}
}

func TestFilterLinkedPathAndValues(t *testing.T) {
	for _, testCase := range []struct {
		desc     string
		input    map[string][]PathAndValueHolder
		expected []map[string][]string
	}{
		{
			desc: "Simple case, no arrays at all",
			input: map[string][]PathAndValueHolder{
				"Namespace": {pathAndValuesFromSteps(t, "Namespace")},
			},
			expected: filteredResultWithSingleMatch("Namespace", pathAndValuesFromSteps(t, "Namespace")),
		},
		{
			desc: "One array, same level, no match",
			input: map[string][]PathAndValueHolder{
				"VolumeName":   {pathAndValuesFromSteps(t, "Volumes", 2, "Name")},
				"VolumeSource": {pathAndValuesFromSteps(t, "Volumes", 1, "Source")},
			},
		},
		{
			desc: "One array, same level",
			input: map[string][]PathAndValueHolder{
				"VolumeName":   {pathAndValuesFromSteps(t, "Volumes", 0, "Name"), pathAndValuesFromSteps(t, "Volumes", 2, "Name"), pathAndValuesFromSteps(t, "Volumes", 3, "Name")},
				"VolumeSource": {pathAndValuesFromSteps(t, "Volumes", 0, "Source"), pathAndValuesFromSteps(t, "Volumes", 1, "Source"), pathAndValuesFromSteps(t, "Volumes", 3, "Source")},
			},

			expected: []map[string][]string{
				{
					"VolumeName":   {"ValVolumes.0.Name"},
					"VolumeSource": {"ValVolumes.0.Source"},
				},
				{
					"VolumeName":   {"ValVolumes.3.Name"},
					"VolumeSource": {"ValVolumes.3.Source"},
				},
			},
		},
		{
			desc: "Complex case, multi-level linking but no branching, no match",
			input: map[string][]PathAndValueHolder{
				"Namespace":     {pathAndValuesFromSteps(t, "Namespace")},
				"ContainerName": {pathAndValuesFromSteps(t, "Containers", 0, "Name")},
				"VolumeName":    {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name")},
				"VolumeSource":  {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"), pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source")},
			},
		},
		{
			desc: "Complex case, multi-level linking but no branching",
			input: map[string][]PathAndValueHolder{
				"Namespace":     {pathAndValuesFromSteps(t, "Namespace")},
				"ContainerName": {pathAndValuesFromSteps(t, "Containers", 0, "Name"), pathAndValuesFromSteps(t, "Containers", 1, "Name")},
				"VolumeName":    {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name")},
				"VolumeSource":  {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"), pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source")},
			},

			expected: []map[string][]string{
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.0.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.0.Source"},
				},
			},
		},
		{
			desc: "Complex case, multi-level linking plus branching no match",
			input: map[string][]PathAndValueHolder{
				"Namespace":     {pathAndValuesFromSteps(t, "Namespace")},
				"ContainerName": {pathAndValuesFromSteps(t, "Containers", 0, "Name"), pathAndValuesFromSteps(t, "Containers", 1, "Name")},
				"VolumeName":    {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name")},
				"VolumeSource":  {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"), pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source")},
				"PortName":      {pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Name"), pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Name")},
				"PortProtocol":  {pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Protocol"), pathAndValuesFromSteps(t, "Containers", 1, "Ports", 1, "Protocol")},
			},
		},
		{
			desc: "Complex case, multi-level linking plus branching",
			input: map[string][]PathAndValueHolder{
				"Namespace":     {pathAndValuesFromSteps(t, "Namespace")},
				"ContainerName": {pathAndValuesFromSteps(t, "Containers", 0, "Name"), pathAndValuesFromSteps(t, "Containers", 1, "Name")},
				"VolumeName":    {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name")},
				"VolumeSource":  {pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"), pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source")},
				"PortName":      {pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Name"), pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Name")},
				"PortProtocol":  {pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Protocol"), pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Protocol")},
			},
			expected: []map[string][]string{
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.0.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.0.Source"},
				},
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"PortName":      {"ValContainers.1.Ports.0.Name"},
					"PortProtocol":  {"ValContainers.1.Ports.0.Protocol"},
				},
			},
		},
		{
			desc: "Complex case, multi-level linking plus branching, multiple matching sub-objects",
			input: map[string][]PathAndValueHolder{
				"Namespace": {pathAndValuesFromSteps(t, "Namespace")},
				"ContainerName": {
					pathAndValuesFromSteps(t, "Containers", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Name"),
				},
				"VolumeName": {
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Name"),
				},
				"VolumeSource": {
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 2, "Source"),
				},
				"PortName": {
					pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Name"),
				},
				"PortProtocol": {
					pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Protocol"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Protocol"),
				},
			},
			expected: []map[string][]string{
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.0.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.0.Source"},
				},
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.1.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.1.Source"},
				},
				{
					"Namespace":     {"ValNamespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"PortName":      {"ValContainers.1.Ports.0.Name"},
					"PortProtocol":  {"ValContainers.1.Ports.0.Protocol"},
				},
			},
		},
		{
			desc: "Complex case, multi-level linking plus branching, multiple matching objects and sub-objects",
			input: map[string][]PathAndValueHolder{
				"Namespace": {pathAndValuesFromSteps(t, "NestedNamespace", "Namespace")},
				"ContainerName": {
					pathAndValuesFromSteps(t, "Containers", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Name"),
					pathAndValuesFromSteps(t, "Containers", 2, "Name"),
				},
				"VolumeName": {
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Name"),
					pathAndValuesFromSteps(t, "Containers", 2, "Volumes", 2, "Name"),
				},
				"VolumeSource": {
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 0, "Source"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 1, "Source"),
					pathAndValuesFromSteps(t, "Containers", 1, "Volumes", 2, "Source"),
					pathAndValuesFromSteps(t, "Containers", 2, "Volumes", 2, "Source"),
				},
				"PortName": {
					pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Name"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 1, "Name"),
					pathAndValuesFromSteps(t, "Containers", 2, "Ports", 1, "Name"),
				},
				"PortProtocol": {
					pathAndValuesFromSteps(t, "Containers", 0, "Ports", 0, "Protocol"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 0, "Protocol"),
					pathAndValuesFromSteps(t, "Containers", 1, "Ports", 1, "Protocol"),
					pathAndValuesFromSteps(t, "Containers", 2, "Ports", 1, "Protocol"),
				},
			},
			expected: []map[string][]string{
				{
					"Namespace": {"ValNestedNamespace.Namespace"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.0.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.0.Source"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"VolumeName":    {"ValContainers.1.Volumes.1.Name"},
					"VolumeSource":  {"ValContainers.1.Volumes.1.Source"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.2.Name"},
					"VolumeName":    {"ValContainers.2.Volumes.2.Name"},
					"VolumeSource":  {"ValContainers.2.Volumes.2.Source"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"PortName":      {"ValContainers.1.Ports.0.Name"},
					"PortProtocol":  {"ValContainers.1.Ports.0.Protocol"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.1.Name"},
					"PortName":      {"ValContainers.1.Ports.1.Name"},
					"PortProtocol":  {"ValContainers.1.Ports.1.Protocol"},
				},
				{
					"Namespace":     {"ValNestedNamespace.Namespace"},
					"ContainerName": {"ValContainers.2.Name"},
					"PortName":      {"ValContainers.2.Ports.1.Name"},
					"PortProtocol":  {"ValContainers.2.Ports.1.Protocol"},
				},
			},
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()
			output, matched, err := FilterMatchesToResults(c.input)
			require.NoError(t, err)
			if c.expected == nil {
				assert.Empty(t, output)
				assert.False(t, matched)
			} else {
				assert.True(t, matched)
				assert.ElementsMatch(t, c.expected, output)
			}
		})
	}
}
