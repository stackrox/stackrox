package k8sutil

import (
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
)

var (
	knownRuntimes = map[string]storage.ContainerRuntime{
		"docker": storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
		"cri-o":  storage.ContainerRuntime_CRIO_CONTAINER_RUNTIME,
	}
)

// ParseContainerRuntimeString parses a string prefixed with `<container runtime>://` into the container runtime (as an
// enum) and the remainder of the string. E.g., when applied to "docker://1.12" (as used for the container runtime
// version), the result will be `storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME, "1.12"`.
// Strings of these form are used by Kubernetes (at least) for the container runtime of a node, as well as for container
// IDs.
func ParseContainerRuntimeString(input string) (storage.ContainerRuntime, string) {
	runtime := storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME
	parts := strings.SplitN(input, "://", 2)
	rest := input
	if len(parts) == 2 {
		rest = parts[1]
		if rt, ok := knownRuntimes[parts[0]]; ok {
			runtime = rt
		}
	}
	return runtime, rest
}

// ParseContainerRuntimeVersion parses a Kubernetes container runtime version string such as `docker://1.13` into a
// ContainerRuntimeInfo object.
func ParseContainerRuntimeVersion(versionString string) *storage.ContainerRuntimeInfo {
	rt, version := ParseContainerRuntimeString(versionString)
	if rt == storage.ContainerRuntime_UNKNOWN_CONTAINER_RUNTIME {
		version = versionString // use the full string, e.g., `somert://1.2.3`
	}
	return &storage.ContainerRuntimeInfo{
		Type:    rt,
		Version: version,
	}
}
