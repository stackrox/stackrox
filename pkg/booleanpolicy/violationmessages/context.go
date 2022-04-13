package violationmessages

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/set"
)

// ContextQueryFields is a map of lifecycle stage to query field names to be added for violation message context
type ContextQueryFields map[storage.LifecycleStage]set.FrozenStringSet

// Context Fields to be added to queries
var (
	ImageContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag})
	VulnContextFields = newContextFields(
		[]string{search.CVE.String(), search.CVSS.String(), search.Severity.String(), augmentedobjs.ComponentAndVersionCustomTag},
		[]string{augmentedobjs.ContainerNameCustomTag, search.CVE.String(), search.CVSS.String(), search.Severity.String(), augmentedobjs.ComponentAndVersionCustomTag})
	VolumeContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag, search.VolumeName.String(), search.VolumeSource.String(), search.VolumeDestination.String(), search.VolumeReadonly.String(), search.VolumeType.String()})
	ContainerContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag})
	ResourceContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag})
	EnvVarContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag})
	PortContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag, search.Port.String(), search.PortProtocol.String()})
	ProcessBaselineContextFields = newContextFields(
		nil,
		[]string{augmentedobjs.ContainerNameCustomTag, search.ProcessName.String()})
)

func newContextFields(buildStageContext []string, deployStageContext []string) ContextQueryFields {
	return ContextQueryFields{
		storage.LifecycleStage_BUILD:  set.NewFrozenStringSet(buildStageContext...),
		storage.LifecycleStage_DEPLOY: set.NewFrozenStringSet(deployStageContext...),
	}
}
