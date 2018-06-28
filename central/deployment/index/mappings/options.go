package mappings

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
)

// OptionsMap is exposed for e2e test
var OptionsMap = map[string]*v1.SearchField{
	search.Cluster:    search.NewStringField("deployment.cluster_name"),
	search.Namespace:  search.NewStringField("deployment.namespace"),
	search.LabelKey:   search.NewStringField("deployment.labels.key"),
	search.LabelValue: search.NewStringField("deployment.labels.value"),

	search.CPUCoresLimit:     search.NewNumericField("deployment.containers.resources.cpu_cores_limit"),
	search.CPUCoresRequest:   search.NewNumericField("deployment.containers.resources.cpu_cores_request"),
	search.DeploymentID:      search.NewField("deployment.id", v1.SearchDataType_SEARCH_STRING, search.OptionHidden|search.OptionStore),
	search.DeploymentName:    search.NewStringField("deployment.name"),
	search.DeploymentType:    search.NewStringField("deployment.type"),
	search.AddCapabilities:   search.NewStringField("deployment.containers.security_context.add_capabilities"),
	search.DropCapabilities:  search.NewStringField("deployment.containers.security_context.drop_capabilities"),
	search.EnvironmentKey:    search.NewStringField("deployment.containers.config.env.key"),
	search.EnvironmentValue:  search.NewStringField("deployment.containers.config.env.value"),
	search.MemoryLimit:       search.NewNumericField("deployment.containers.resources.memory_mb_limit"),
	search.MemoryRequest:     search.NewNumericField("deployment.containers.resources.memory_mb_request"),
	search.Privileged:        search.NewBoolField("deployment.containers.security_context.privileged"),
	search.SecretName:        search.NewStringField("deployment.containers.secrets.name"),
	search.SecretPath:        search.NewStringField("deployment.containers.secrets.path"),
	search.VolumeName:        search.NewStringField("deployment.containers.volumes.name"),
	search.VolumeSource:      search.NewStringField("deployment.containers.volumes.source"),
	search.VolumeDestination: search.NewStringField("deployment.containers.volumes.destination"),
	search.VolumeReadonly:    search.NewBoolField("deployment.containers.volumes.read_only"),
	search.VolumeType:        search.NewStringField("deployment.containers.volumes.type"),
}
